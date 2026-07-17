package choice

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
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
	selectedExpectation, err := NewSelectedTuple(record, []string{WholeProjectionFieldID}, expectation.Fields)
	if err != nil {
		t.Fatal(err)
	}
	draft := RulingDraftInput{
		Action: ActionCustomExpectation, SelectedFields: []string{WholeProjectionFieldID}, AllowedAliases: []string{},
		CustomExpectation: &selectedExpectation, CustomReviewer: "local-custom-reviewer",
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

func TestLegacyDecisionRemainsByteExactHistoryAndRefusesPortablePreparation(t *testing.T) {
	record := checkedChoicepointExample(t)
	if view, err := NewBlindView(record); err != nil || view.DTO().ProjectionMode() != legacyWholeProjectionMode {
		t.Fatal("checked legacy Choicepoint did not reconstruct its whole-projection blind authority")
	}
	exact := checkedExampleBytes(t, "decision-record.valid.json")
	parsed, err := ParseDecisionRecord(exact, record)
	if err != nil || !bytes.Equal(parsed.CanonicalBytes(), exact) {
		t.Fatalf("legacy DecisionRecord failed exact rebuild: %v", err)
	}
	if _, err := InspectPortableRuling(parsed); !IsRefusal(err, CodeLegacyWholeProjectionNotPortable) {
		t.Fatalf("legacy portable preparation error = %v, want %s", err, CodeLegacyWholeProjectionNotPortable)
	}
}

func TestFreshConstructionFromLegacyEvidenceUsesPortableMode(t *testing.T) {
	legacy := checkedChoicepointExample(t)
	legacyExact := legacy.CanonicalBytes()
	fresh, err := NewChoicepointRecord(ChoicepointInput{
		Scenario: legacy.scenario, Plan: legacy.plan, Envelope: legacy.envelope,
		CandidateBindings: legacy.bindings, OriginalStimulus: legacy.original,
		MinimizedStimulus: legacy.minimized, Confirmation: legacy.confirmation,
		CandidateReveals: legacy.reveals, EvidenceReceipts: legacy.receipts,
	})
	if err != nil {
		t.Fatalf("fresh construction from exact historical evidence: %v", err)
	}
	if fresh.mode != choicepointPortable || fresh.mode.wire() != portableProjectionMode ||
		bytes.Equal(fresh.CanonicalBytes(), legacyExact) || legacy.mode != choicepointLegacyWhole ||
		!bytes.Equal(legacy.CanonicalBytes(), legacyExact) {
		t.Fatal("fresh public construction fell back to legacy mode or mutated historical bytes")
	}
	if fresh.confirmed.OutcomeMapDigest() != legacy.confirmed.OutcomeMapDigest() ||
		fresh.confirmed.PreservationDigest() != legacy.confirmed.PreservationDigest() ||
		fresh.confirmed.OutcomeMapDigest() != legacy.confirmation.ConfirmedArtifactDigest() ||
		fresh.confirmed.PreservationDigest() != legacy.confirmation.ConfirmedMap().PreservationDigest() ||
		fresh.confirmation.Digest() != legacy.confirmation.Digest() ||
		!bytes.Equal(fresh.confirmation.CanonicalBytes(), legacy.confirmation.CanonicalBytes()) {
		t.Fatal("fresh portable interpretation changed the exact confirmed proof authority")
	}
	if len(fresh.confirmed.ordered) != len(legacy.confirmed.ordered) {
		t.Fatal("fresh portable interpretation changed the confirmed outcome cardinality")
	}
	for index := range legacy.confirmed.ordered {
		legacyOutcome := legacy.confirmed.ordered[index]
		freshOutcome := fresh.confirmed.ordered[index]
		recomputedFingerprint, err := domain.NewProjectionFingerprint(legacyOutcome.canonicalProjection)
		if err != nil {
			t.Fatal(err)
		}
		recomputedID, err := makeConfirmedOutcomeID(legacyOutcome.ref.candidate, recomputedFingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if legacyOutcome.ref.id != freshOutcome.ref.id ||
			legacyOutcome.ref.candidate != freshOutcome.ref.candidate ||
			legacyOutcome.ref.fingerprint != freshOutcome.ref.fingerprint ||
			legacyOutcome.ref.fingerprint != recomputedFingerprint ||
			legacyOutcome.ref.id != recomputedID ||
			!bytes.Equal(legacyOutcome.canonicalProjection, freshOutcome.canonicalProjection) {
			t.Fatalf("fresh portable interpretation changed confirmed proof identity at outcome %d", index)
		}
	}
	var legacyIdentity, freshIdentity choicepointIdentity
	if err := json.Unmarshal(legacy.CanonicalBytes(), &legacyIdentity); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fresh.CanonicalBytes(), &freshIdentity); err != nil {
		t.Fatal(err)
	}
	legacyIdentity.ChoiceProjectionMode = ""
	freshIdentity.ChoiceProjectionMode = ""
	if !reflect.DeepEqual(legacyIdentity, freshIdentity) {
		t.Fatal("fresh Choicepoint changed an identity member other than choice_projection_mode")
	}
	if fresh.Digest() == legacy.Digest() || fresh.confirmed.seal == legacy.confirmed.seal ||
		fresh.confirmed.registry.fieldRegistryDigest == legacy.confirmed.registry.fieldRegistryDigest ||
		!fresh.confirmed.registry.profileDigest.Valid() || legacy.confirmed.registry.profileDigest.Valid() ||
		fresh.confirmed.registry.profileDigest == fresh.confirmed.registry.projectionDefinitionDigest {
		t.Fatal("fresh portable mode did not acquire distinct registry, profile, universe, and Choicepoint identities")
	}
	parsed, err := ParseChoicepointRecord(fresh.CanonicalBytes())
	if err != nil || parsed.mode != choicepointPortable || parsed.Digest() != fresh.Digest() {
		t.Fatalf("fresh portable Choicepoint did not reconstruct exactly: %v", err)
	}
}

func TestChoicepointAuthorityGettersAreExactAndDefensiveForLegacyAndPortableRecords(t *testing.T) {
	legacy := checkedChoicepointExample(t)
	portable := freshPortableChoicepointExample(t)
	for name, record := range map[string]ChoicepointRecord{"legacy": legacy, "portable": portable} {
		t.Run(name, func(t *testing.T) {
			plan := record.WorldPlan()
			stimulus := record.MinimizedStimulus()
			profile := record.PortableProfile()
			if plan.Digest() != record.plan.Digest() || !bytes.Equal(plan.CanonicalBytes(), record.plan.CanonicalBytes()) ||
				stimulus.Kind() != record.minimized.Kind() || stimulus.Digest() != record.minimized.Digest() ||
				!bytes.Equal(stimulus.CanonicalBytes(), record.minimized.CanonicalBytes()) {
				t.Fatal("authority getter changed exact plan or minimized stimulus")
			}
			if name == "legacy" && profile.Valid() {
				t.Fatal("legacy Choicepoint unexpectedly exposed a portable profile")
			}
			if name == "portable" && (!profile.Valid() ||
				profile.Digest() != record.confirmed.registry.profileDigest ||
				!bytes.Equal(profile.CanonicalBytes(), record.confirmed.registry.expectationDomain.ProfileBytes())) {
				t.Fatal("portable profile getter changed exact prepared-ruling profile authority")
			}
			planCopy := plan.CanonicalBytes()
			stimulusCopy := stimulus.CanonicalBytes()
			planCopy[0] ^= 0xff
			stimulusCopy[0] ^= 0xff
			profileCopy := profile.CanonicalBytes()
			if len(profileCopy) > 0 {
				profileCopy[0] ^= 0xff
			}
			if !bytes.Equal(record.WorldPlan().CanonicalBytes(), record.plan.CanonicalBytes()) ||
				!bytes.Equal(record.MinimizedStimulus().CanonicalBytes(), record.minimized.CanonicalBytes()) ||
				(name == "portable" && !bytes.Equal(record.PortableProfile().CanonicalBytes(), record.confirmed.registry.expectationDomain.ProfileBytes())) ||
				!record.Valid() {
				t.Fatal("caller mutation changed Choicepoint authority")
			}
		})
	}
}

func TestPortableDraftBudgetNeverDefersStructuralOverflowToFinalize(t *testing.T) {
	record := freshPortableChoicepointExample(t)
	definitions := record.confirmed.registry.Definitions()
	selected := make([]string, len(definitions))
	for index, definition := range definitions {
		selected[index] = definition.ID
	}
	if len(selected) == 0 {
		t.Fatal("portable budget fixture has no selectable fields")
	}

	successes := 0
	proposalRefusals := 0
	sameDraftRevisionRefusals := 0
	var largestAcceptedDraft RulingDraftInput
	largestAcceptedPayload := 0
	for _, payloadBytes := range []int{1024, 16 * 1024, 32 * 1024, 40 * 1024, 48 * 1024, 56 * 1024, 60 * 1024, 64 * 1024} {
		fields := make([]FieldValue, len(definitions))
		for index, definition := range definitions {
			fields[index] = FieldValue{FieldID: definition.ID, Value: portableBudgetValue(t, definition, payloadBytes)}
		}
		expectation, err := NewSelectedTuple(record, selected, fields)
		if IsRefusal(err, CodeInputLimitExceeded) {
			continue
		}
		if err != nil {
			t.Fatalf("payload %d selected tuple: %v", payloadBytes, err)
		}
		draft := RulingDraftInput{
			Action: ActionCustomExpectation, SelectedFields: selected, AllowedAliases: []string{},
			CustomExpectation: &expectation, CustomReviewer: "portable-budget-reviewer",
			CustomReviewEvidence: reviewDigest(),
		}
		session := visitedBlindSession(t, record)
		session, err = session.Propose(draft)
		if IsRefusal(err, CodeInputLimitExceeded) {
			proposalRefusals++
			continue
		}
		if err != nil {
			t.Fatalf("payload %d proposal: %v", payloadBytes, err)
		}
		largestAcceptedDraft = draft
		largestAcceptedPayload = payloadBytes
		session, _, err = session.Reveal()
		if err != nil {
			t.Fatal(err)
		}
		session, err = session.Visit(SurfaceProvenance)
		if err != nil {
			t.Fatal(err)
		}
		session, err = session.Revise(draft, "")
		if IsRefusal(err, CodeInputLimitExceeded) {
			sameDraftRevisionRefusals++
			continue
		}
		if err != nil {
			t.Fatalf("payload %d revision: %v", payloadBytes, err)
		}
		_, decision, err := session.Finalize("x", "", []domain.ReceiptReference{})
		if err != nil {
			t.Fatalf("payload %d passed draft preflight but failed minimal finalization: %v", payloadBytes, err)
		}
		if _, err := ParseDecisionRecord(decision.CanonicalBytes(), record); err != nil {
			t.Fatalf("payload %d finalized DecisionRecord did not reconstruct: %v", payloadBytes, err)
		}
		successes++
	}
	if successes == 0 || proposalRefusals == 0 || sameDraftRevisionRefusals != 0 || largestAcceptedPayload == 0 {
		t.Fatalf("portable durable-budget boundary was not isolated: successes=%d proposal_refusals=%d same_draft_revision_refusals=%d largest_accepted=%d",
			successes, proposalRefusals, sameDraftRevisionRefusals, largestAcceptedPayload)
	}

	revisionSession := visitedBlindSession(t, record)
	revisionSession, err := revisionSession.Propose(largestAcceptedDraft)
	if err != nil {
		t.Fatal(err)
	}
	revisionSession, _, err = revisionSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	revisionSession, err = revisionSession.Visit(SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	reviseRefused := false
	for _, payloadBytes := range []int{32 * 1024, 40 * 1024, 48 * 1024, 56 * 1024, 60 * 1024, 64 * 1024} {
		if payloadBytes <= largestAcceptedPayload {
			continue
		}
		largeDraft, valid := portableBudgetDraft(t, record, payloadBytes)
		if !valid {
			continue
		}
		if _, err := newRulingDraft(record, revisionSession.view, largeDraft); err != nil {
			t.Fatalf("payload %d did not reach DecisionRecord preflight: %v", payloadBytes, err)
		}
		_, err := revisionSession.Revise(largeDraft, "The larger exact expectation replaces the provisional ruling.")
		if IsRefusal(err, CodeInputLimitExceeded) {
			reviseRefused = true
			if revisionSession.State() != SessionRevealed {
				t.Fatal("failed portable revision mutated the source session")
			}
			break
		}
		if err != nil {
			t.Fatalf("payload %d revision failed before the budget boundary: %v", payloadBytes, err)
		}
	}
	if !reviseRefused {
		t.Fatal("portable Revise never refused a structurally oversized changed final draft")
	}

	multiplicityProved := false
	for _, payloadBytes := range []int{24 * 1024, 32 * 1024, 40 * 1024, 48 * 1024, 56 * 1024, 60 * 1024, 64 * 1024} {
		draft, valid := portableBudgetDraft(t, record, payloadBytes)
		if !valid {
			continue
		}
		early, err := NewSession(record)
		if err != nil {
			t.Fatal(err)
		}
		early, err = early.Visit(SurfaceOriginalWitness)
		if err != nil {
			t.Fatal(err)
		}
		early, _, err = early.Reveal()
		if err != nil {
			t.Fatal(err)
		}
		for _, surface := range []ReviewSurface{
			SurfaceMinimizedWitness, SurfaceReductionDerivation, SurfaceProjectionOperations,
			SurfaceNonassertedFields, SurfaceProvenance,
		} {
			early, err = early.Visit(surface)
			if err != nil {
				t.Fatal(err)
			}
		}
		early, earlyErr := early.Revise(draft, "")
		if earlyErr != nil {
			if IsRefusal(earlyErr, CodeInputLimitExceeded) {
				continue
			}
			t.Fatalf("payload %d early-reveal revision: %v", payloadBytes, earlyErr)
		}
		blind := visitedBlindSession(t, record)
		if _, blindErr := blind.Propose(draft); !IsRefusal(blindErr, CodeInputLimitExceeded) {
			continue
		}
		_, decision, err := early.Finalize("x", "", []domain.ReceiptReference{})
		if err != nil {
			t.Fatalf("payload %d early-reveal finalization: %v", payloadBytes, err)
		}
		if !decision.EarlyReveal() || !reflect.DeepEqual(decision.identity.PreRevealVisitedSurfaces, []string{string(SurfaceOriginalWitness)}) {
			t.Fatal("early-reveal DecisionRecord lost its exact mixed visit partition")
		}
		if _, err := ParseDecisionRecord(decision.CanonicalBytes(), record); err != nil {
			t.Fatalf("payload %d early-reveal DecisionRecord did not reconstruct: %v", payloadBytes, err)
		}
		multiplicityProved = true
		break
	}
	if !multiplicityProved {
		t.Fatal("portable budget test did not distinguish one early-reveal draft from two blind-first copies")
	}
}

func TestPortableDecisionLateBudgetIsExactAndLegacyHistoryKeepsFullCeiling(t *testing.T) {
	portableRecord := freshPortableChoicepointExample(t)
	portableSession := finalizableRejectSession(t, portableRecord)
	base, err := buildDecisionRecord(portableSession, "x", "", []domain.ReceiptReference{}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := domain.NewDidrunReceipt("x", "c", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	probeDecision, err := buildDecisionRecord(portableSession, "x", "", []domain.ReceiptReference{probe}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	fixedReceiptGrowth := len(probeDecision.CanonicalBytes()) - len(base.CanonicalBytes()) - 1
	gradeBytes := maxPortableDecisionLateBytes - fixedReceiptGrowth
	if gradeBytes <= 0 {
		t.Fatalf("portable late-budget fixture overhead = %d", fixedReceiptGrowth)
	}
	exactReceipt, err := domain.NewDidrunReceipt(strings.Repeat("x", gradeBytes), "c", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	exactRaw, err := buildDecisionRecord(portableSession, "x", "", []domain.ReceiptReference{exactReceipt}, nil, false)
	if err != nil || len(exactRaw.CanonicalBytes())-len(base.CanonicalBytes()) != maxPortableDecisionLateBytes {
		t.Fatalf("exact portable late delta = %d, want %d: %v",
			len(exactRaw.CanonicalBytes())-len(base.CanonicalBytes()), maxPortableDecisionLateBytes, err)
	}
	_, exactDecision, err := portableSession.Finalize("x", "", []domain.ReceiptReference{exactReceipt})
	if err != nil {
		t.Fatalf("exact portable late budget was refused: %v", err)
	}
	if _, err := ParseDecisionRecord(exactDecision.CanonicalBytes(), portableRecord); err != nil {
		t.Fatalf("exact portable late-budget DecisionRecord did not reconstruct: %v", err)
	}

	overflowReceipt, err := domain.NewDidrunReceipt(strings.Repeat("x", gradeBytes+1), "c", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	overflowRaw, err := buildDecisionRecord(portableSession, "x", "", []domain.ReceiptReference{overflowReceipt}, nil, false)
	if err != nil || len(overflowRaw.CanonicalBytes()) > canon.MaxInputBytes {
		t.Fatalf("late-budget overflow control hit the generic object ceiling: bytes=%d err=%v", len(overflowRaw.CanonicalBytes()), err)
	}
	if _, _, err := portableSession.Finalize("x", "", []domain.ReceiptReference{overflowReceipt}); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("portable late budget +1 finalization error = %v, want %s", err, CodeInputLimitExceeded)
	}
	if _, err := ParseDecisionRecord(overflowRaw.CanonicalBytes(), portableRecord); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("portable late budget +1 parse error = %v, want %s", err, CodeInputLimitExceeded)
	}

	legacyRecord := checkedChoicepointExample(t)
	legacySession := finalizableRejectSession(t, legacyRecord)
	_, legacyDecision, err := legacySession.Finalize("x", "", []domain.ReceiptReference{overflowReceipt})
	if err != nil {
		t.Fatalf("legacy DecisionRecord inherited the portable late reservation: %v", err)
	}
	if _, err := ParseDecisionRecord(legacyDecision.CanonicalBytes(), legacyRecord); err != nil {
		t.Fatalf("legacy DecisionRecord above the portable late reservation did not reconstruct: %v", err)
	}
}

func TestDecisionReceiptAdmissionIsBoundedBeforeSorting(t *testing.T) {
	record := freshPortableChoicepointExample(t)
	session := finalizableRejectSession(t, record)
	tooMany := make([]domain.ReceiptReference, canon.MaxContainerMembers+1)
	if _, err := normalizeDecisionReceipts(tooMany); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("direct oversized receipt roster error = %v, want %s", err, CodeInputLimitExceeded)
	}
	if _, _, err := session.Finalize("x", "", tooMany); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized receipt roster error = %v, want %s", err, CodeInputLimitExceeded)
	}
	huge, err := domain.NewDidrunReceipt(strings.Repeat("x", canon.MaxInputBytes+1), "c", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeDecisionReceipts([]domain.ReceiptReference{huge}); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("direct oversized receipt payload error = %v, want %s", err, CodeInputLimitExceeded)
	}
	if _, _, err := session.Finalize("x", "", []domain.ReceiptReference{huge}); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized receipt payload error = %v, want %s", err, CodeInputLimitExceeded)
	}
}

func TestChoicepointReceiptAdmissionIsBoundedBeforeSorting(t *testing.T) {
	tooMany := make([]domain.ReceiptReference, canon.MaxContainerMembers+1)
	if _, err := normalizeChoicepointReceipts(tooMany); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized Choicepoint receipt roster error = %v, want %s", err, CodeInputLimitExceeded)
	}
	huge, err := domain.NewDidrunReceipt(strings.Repeat("x", canon.MaxInputBytes+1), "c", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeChoicepointReceipts([]domain.ReceiptReference{huge}); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized Choicepoint receipt payload error = %v, want %s", err, CodeInputLimitExceeded)
	}
	first, err := domain.NewDidrunReceipt("grade-a", "commit-a", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewDidrunReceipt("grade-b", "commit-b", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := normalizeChoicepointReceipts([]domain.ReceiptReference{second, first})
	if err != nil || len(normalized) != 2 || receiptIdentityKey(normalized[0]) >= receiptIdentityKey(normalized[1]) {
		t.Fatalf("valid reverse-order receipts were not normalized once: %#v, %v", normalized, err)
	}
	if _, err := normalizeChoicepointReceipts([]domain.ReceiptReference{first, first}); !IsRefusal(err, CodeInvalidConfirmedOutcomeSet) {
		t.Fatalf("duplicate Choicepoint receipt error = %v, want %s", err, CodeInvalidConfirmedOutcomeSet)
	}
	legacy := checkedChoicepointExample(t)
	record, err := NewChoicepointRecord(ChoicepointInput{
		Scenario: legacy.scenario, Plan: legacy.plan, Envelope: legacy.envelope,
		CandidateBindings: legacy.bindings, OriginalStimulus: legacy.original,
		MinimizedStimulus: legacy.minimized, Confirmation: legacy.confirmation,
		CandidateReveals: legacy.reveals, EvidenceReceipts: []domain.ReceiptReference{second, first},
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || !reflect.DeepEqual(receiptWires(reopened.receipts), receiptWires(normalized)) {
		t.Fatalf("normalized Choicepoint receipts did not reconstruct exactly: %v", err)
	}
}

func receiptWires(receipts []domain.ReceiptReference) []domain.ReceiptWire {
	wires := make([]domain.ReceiptWire, len(receipts))
	for index, receipt := range receipts {
		wires[index] = receipt.Wire()
	}
	return wires
}

func TestRulingAliasAdmissionIsBoundedBeforeNormalization(t *testing.T) {
	record := freshPortableChoicepointExample(t)
	view, err := NewBlindView(record)
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([]string, len(view.aliases)+1)
	if _, _, err := view.normalizeAliases(tooMany); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized blind alias roster error = %v, want %s", err, CodeInputLimitExceeded)
	}
	if _, _, err := view.normalizeAliases([]string{strings.Repeat("x", maxBlindAliasBytes+1)}); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized blind alias payload error = %v, want %s", err, CodeInputLimitExceeded)
	}
	cards := view.DTO().Cards()
	all := make([]string, len(cards))
	for index, card := range cards {
		all[index] = card.Alias
	}
	if normalized, resolved, err := view.normalizeAliases(all); err != nil || len(normalized) != len(cards) || len(resolved) != len(record.confirmed.ordered) {
		t.Fatalf("complete rendered alias selection was not admitted: aliases=%d refs=%d err=%v", len(normalized), len(resolved), err)
	}
	if _, _, err := view.normalizeAliases([]string{strings.Repeat("x", maxBlindAliasBytes)}); !IsRefusal(err, CodeUnconfirmedOutcomeSelection) {
		t.Fatalf("bounded foreign alias error = %v, want %s", err, CodeUnconfirmedOutcomeSelection)
	}
}

func TestDecisionAnnotationV1RetainsExactInertBytes(t *testing.T) {
	record := freshPortableChoicepointExample(t)
	session := finalizableRejectSession(t, record)
	annotation := "line one\n\x1b[31m<b>untrusted</b>\u202e\U0001f469\u200d\U0001f4bb"
	_, decision, err := session.Finalize("x", annotation, []domain.ReceiptReference{})
	if err != nil || decision.HumanAnnotation() != annotation {
		t.Fatalf("DecisionRecord v1 did not retain exact inert annotation bytes: %v", err)
	}
	reopened, err := ParseDecisionRecord(decision.CanonicalBytes(), record)
	if err != nil || reopened.Digest() != decision.Digest() || reopened.HumanAnnotation() != annotation {
		t.Fatalf("DecisionRecord v1 annotation did not reconstruct byte-exactly: %v", err)
	}
}

func portableBudgetDraft(t testing.TB, record ChoicepointRecord, payloadBytes int) (RulingDraftInput, bool) {
	t.Helper()
	definitions := record.confirmed.registry.Definitions()
	selected := make([]string, len(definitions))
	fields := make([]FieldValue, len(definitions))
	for index, definition := range definitions {
		selected[index] = definition.ID
		fields[index] = FieldValue{FieldID: definition.ID, Value: portableBudgetValue(t, definition, payloadBytes)}
	}
	expectation, err := NewSelectedTuple(record, selected, fields)
	if IsRefusal(err, CodeInputLimitExceeded) {
		return RulingDraftInput{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return RulingDraftInput{
		Action: ActionCustomExpectation, SelectedFields: selected, AllowedAliases: []string{},
		CustomExpectation: &expectation, CustomReviewer: "portable-budget-reviewer",
		CustomReviewEvidence: reviewDigest(),
	}, true
}

func finalizableRejectSession(t testing.TB, record ChoicepointRecord) Session {
	t.Helper()
	draft := RulingDraftInput{Action: ActionRejectAll, SelectedFields: []string{}, AllowedAliases: []string{}}
	session := proposedRevealedSession(t, visitedBlindSession(t, record), draft)
	var err error
	session, err = session.Revise(draft, "")
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func freshPortableChoicepointExample(t testing.TB) ChoicepointRecord {
	t.Helper()
	legacy := checkedChoicepointExample(t)
	fresh, err := NewChoicepointRecord(ChoicepointInput{
		Scenario: legacy.scenario, Plan: legacy.plan, Envelope: legacy.envelope,
		CandidateBindings: legacy.bindings, OriginalStimulus: legacy.original,
		MinimizedStimulus: legacy.minimized, Confirmation: legacy.confirmation,
		CandidateReveals: legacy.reveals, EvidenceReceipts: legacy.receipts,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func portableBudgetValue(t testing.TB, definition FieldDefinition, payloadBytes int) ExactValue {
	t.Helper()
	switch definition.Type {
	case FieldString:
		value, err := StringValue(strings.Repeat("\\", payloadBytes))
		if err != nil {
			t.Fatal(err)
		}
		return value
	case FieldBytes:
		value, err := BytesValue(bytes.Repeat([]byte{0xa5}, payloadBytes))
		if err != nil {
			t.Fatal(err)
		}
		return value
	case FieldOrderedStringList:
		value, err := OrderedStringListValue([]string{strings.Repeat("x", payloadBytes)})
		if err != nil {
			t.Fatal(err)
		}
		return value
	case FieldCanonicalJSON:
		payload := payloadBytes - len(`{"payload":""}`)
		if payload < 0 {
			payload = 0
		}
		value, err := CanonicalJSONBytes([]byte(`{"payload":"` + strings.Repeat("x", payload) + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		return value
	case FieldInteger:
		value, err := IntegerValue("401")
		if err != nil {
			t.Fatal(err)
		}
		return value
	case FieldBoolean:
		return BooleanValue(true)
	default:
		t.Fatalf("unsupported portable budget field type %s", definition.Type)
		return ExactValue{}
	}
}

func TestChoicepointParserRefusesUnknownProjectionMode(t *testing.T) {
	exact := checkedExampleBytes(t, "choicepoint.valid.json")
	unknown := bytes.Replace(
		exact,
		[]byte(`"choice_projection_mode":"WHOLE_EXACT_CANONICAL_PROJECTION_V1"`),
		[]byte(`"choice_projection_mode":"UNKNOWN_PROJECTION_MODE_V1"`),
		1,
	)
	if bytes.Equal(unknown, exact) {
		t.Fatal("unknown-mode test did not locate the exact legacy projection mode")
	}
	if _, err := ParseChoicepointRecord(unknown); !IsRefusal(err, CodeInvalidConfirmedOutcomeSet) {
		t.Fatalf("unknown Choicepoint mode error = %v, want %s", err, CodeInvalidConfirmedOutcomeSet)
	}
}

func checkedChoicepointExample(t testing.TB) ChoicepointRecord {
	t.Helper()
	body := checkedExampleBytes(t, "choicepoint.valid.json")
	record, err := ParseChoicepointRecord(body)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func checkedExampleBytes(t testing.TB, name string) []byte {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller path unavailable")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		t.Fatalf("checked %s example lacks its exact trailing newline", name)
	}
	return append([]byte(nil), body[:len(body)-1]...)
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
