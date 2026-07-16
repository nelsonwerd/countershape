package choice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

const (
	choicepointMemberCount       = 27
	maxChoicepointScenarioBytes  = 8 * 1024
	maxChoicepointRevealBytes    = 2 * 1024
	maxChoicepointNestedRawBytes = 600 * 1024
	choicepointScope             = "EXACT_FRESH_CONFIRMED_WITNESS_V1"
	legacyWholeProjectionMode    = "WHOLE_EXACT_CANONICAL_PROJECTION_V1"
	portableProjectionMode       = projectiontranslate.PortableChoiceModeV1
	choicepointPresentationNote  = "DISPLAY_REFS_AND_PRODUCER_METADATA_ARE_REVEAL_ONLY_NONEXECUTION_PROVENANCE"
)

type choicepointMode uint8

const (
	choicepointLegacyWhole choicepointMode = iota + 1
	choicepointPortable
)

func (m choicepointMode) wire() string {
	switch m {
	case choicepointLegacyWhole:
		return legacyWholeProjectionMode
	case choicepointPortable:
		return portableProjectionMode
	default:
		return ""
	}
}

// CanonicalArtifact binds exact typed stimulus bytes to their domain-separated
// digest. It deliberately accepts only the two v1 adapter stimulus kinds.
type CanonicalArtifact struct {
	kind      string
	digest    domain.Digest
	canonical []byte
}

func NewCanonicalArtifact(kind string, digest domain.Digest, exact []byte) (CanonicalArtifact, error) {
	if (kind != "CLIStimulus" && kind != "HTTPStimulus") || !digest.Valid() || len(exact) == 0 {
		return CanonicalArtifact{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint stimulus identity is incomplete")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return CanonicalArtifact{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint stimulus is outside the canonical profile")
	}
	canonicalBytes, err := value.CanonicalChecked()
	schemaValue, hasSchema := value.LookupMember("schema_version")
	kindValue, hasKind := value.LookupMember("kind")
	schema, schemaText := schemaValue.Text()
	wireKind, kindText := kindValue.Text()
	computed, digestErr := canon.DigestBytes(kind, exact)
	if err != nil || digestErr != nil || !bytes.Equal(canonicalBytes, exact) || !hasSchema || !hasKind ||
		!schemaText || !kindText || schema != domain.SchemaVersion || wireKind != kind || computed.String() != digest.String() {
		return CanonicalArtifact{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint stimulus bytes and digest disagree")
	}
	return CanonicalArtifact{kind: kind, digest: digest, canonical: append([]byte(nil), exact...)}, nil
}

func (a CanonicalArtifact) Kind() string           { return a.kind }
func (a CanonicalArtifact) Digest() domain.Digest  { return a.digest }
func (a CanonicalArtifact) CanonicalBytes() []byte { return append([]byte(nil), a.canonical...) }

func (a CanonicalArtifact) valid() bool {
	rebuilt, err := NewCanonicalArtifact(a.kind, a.digest, a.canonical)
	return err == nil && rebuilt.digest == a.digest && bytes.Equal(rebuilt.canonical, a.canonical)
}

// CandidateReveal is immutable provenance shown only after the explicit reveal
// transition. It never contributes to scheduling, comparison, aliases, or
// blind ordering.
type CandidateReveal struct {
	CandidateExecutionKey domain.CandidateExecutionKey
	DisplayRef            string
	ProducerMetadata      string
}

type ChoicepointInput struct {
	Scenario          string
	Plan              domain.WorldPlan
	Envelope          domain.ComparisonEnvelope
	CandidateBindings []domain.CandidateExecutionBinding
	OriginalStimulus  CanonicalArtifact
	MinimizedStimulus CanonicalArtifact
	Confirmation      confirmation.Record
	CandidateReveals  []CandidateReveal
	EvidenceReceipts  []domain.ReceiptReference
}

// ChoicepointRecord is a strict, inert semantic archive. Current decision
// authority is held by the store-backed promotion.Ready capability, not by
// these serializable bytes.
type ChoicepointRecord struct {
	digest         domain.Digest
	canonicalBytes []byte
	scenario       string
	plan           domain.WorldPlan
	envelope       domain.ComparisonEnvelope
	bindings       []domain.CandidateExecutionBinding
	original       CanonicalArtifact
	minimized      CanonicalArtifact
	confirmation   confirmation.Record
	confirmed      ConfirmedOutcomeSet
	mode           choicepointMode
	reveals        []CandidateReveal
	receipts       []domain.ReceiptReference
}

type candidateRevealIdentity struct {
	CandidateExecutionKey string `json:"candidate_execution_key"`
	DisplayRef            string `json:"display_ref"`
	ProducerMetadata      string `json:"producer_metadata"`
}

type choicepointIdentity struct {
	SchemaVersion                  string                    `json:"schema_version"`
	Kind                           string                    `json:"kind"`
	Scenario                       string                    `json:"scenario"`
	WorldPlanBase64                string                    `json:"world_plan_base64"`
	WorldPlanDigest                string                    `json:"world_plan_digest"`
	ComparisonEnvelopeBase64       string                    `json:"comparison_envelope_base64"`
	ComparisonEnvelopeDigest       string                    `json:"comparison_envelope_digest"`
	ProjectionBindingBase64        string                    `json:"projection_binding_base64"`
	ProjectionDefinitionDigest     string                    `json:"projection_definition_digest"`
	CandidateBindingsBase64        []string                  `json:"candidate_bindings_base64"`
	CandidateExecutionKeys         []string                  `json:"candidate_execution_keys"`
	CandidateReveals               []candidateRevealIdentity `json:"candidate_reveals"`
	OriginalStimulusKind           string                    `json:"original_stimulus_kind"`
	OriginalStimulusBase64         string                    `json:"original_stimulus_base64"`
	OriginalStimulusDigest         string                    `json:"original_stimulus_digest"`
	MinimizedStimulusKind          string                    `json:"minimized_stimulus_kind"`
	MinimizedStimulusBase64        string                    `json:"minimized_stimulus_base64"`
	MinimizedStimulusDigest        string                    `json:"minimized_stimulus_digest"`
	FreshConfirmationBase64        string                    `json:"fresh_confirmation_base64"`
	FreshConfirmationDigest        string                    `json:"fresh_confirmation_digest"`
	ConfirmedOutcomeMapDigest      string                    `json:"confirmed_outcome_map_digest"`
	ConfirmedPreservationMapDigest string                    `json:"confirmed_preservation_map_digest"`
	ChoiceProjectionMode           string                    `json:"choice_projection_mode"`
	EvidenceReceipts               []domain.ReceiptWire      `json:"evidence_receipts"`
	EvidenceStatusWhenEmpty        string                    `json:"evidence_status_when_empty"`
	ReadinessScope                 string                    `json:"readiness_scope"`
	PresentationProvenanceNonclaim string                    `json:"presentation_provenance_nonclaim"`
}

func NewChoicepointRecord(input ChoicepointInput) (ChoicepointRecord, error) {
	return buildChoicepointRecord(input, choicepointPortable, nil)
}

func buildChoicepointRecord(input ChoicepointInput, mode choicepointMode, expected []byte) (ChoicepointRecord, error) {
	receipts, err := normalizeChoicepointReceipts(input.EvidenceReceipts)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	if err := validateChoicepointLineage(input); err != nil {
		return ChoicepointRecord{}, err
	}
	bindings := append([]domain.CandidateExecutionBinding(nil), input.CandidateBindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Key().String() < bindings[j].Key().String() })
	reveals := append([]CandidateReveal(nil), input.CandidateReveals...)
	sort.Slice(reveals, func(i, j int) bool {
		return reveals[i].CandidateExecutionKey.String() < reveals[j].CandidateExecutionKey.String()
	})
	confirmedMap, err := compare.RequireConfirmedOutcomeMap(input.Confirmation.ConfirmedMap())
	if err != nil {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "confirmation record lacks a strict confirmed map")
	}
	confirmationProofs := input.Confirmation.ProjectionProofs()
	var confirmed ConfirmedOutcomeSet
	var expectedStimulusKind string
	switch mode {
	case choicepointPortable:
		proofs := make([]projectiontranslate.ProjectionProof, len(confirmationProofs))
		for index, proof := range confirmationProofs {
			proofs[index] = projectiontranslate.ProjectionProof{
				CandidateExecutionKey: proof.CandidateExecutionKey(), CanonicalProjection: proof.CanonicalProjection(),
			}
		}
		translations, translateErr := projectiontranslate.TranslateConfirmed(
			input.Plan.ProjectionDefinitionBinding(), confirmedMap.ProjectionRoster(), proofs,
		) // MUTANT_P07B_FALLBACK_TO_LEGACY
		if translateErr != nil {
			return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable proof-first projection translation failed: "+translateErr.Error())
		}
		confirmed, err = confirmedOutcomeSetFromTranslations(translations)
		expectedStimulusKind = translations.ExpectedStimulusKind()
	case choicepointLegacyWhole:
		registry, registryErr := newWholeProjectionRegistry(input.Plan.ProjectionDefinitionDigest())
		if registryErr != nil {
			return ChoicepointRecord{}, registryErr
		}
		proofs := make([]ProjectionProofInput, len(confirmationProofs))
		for index, proof := range confirmationProofs {
			proofs[index] = ProjectionProofInput{
				CandidateExecutionKey: proof.CandidateExecutionKey(), CanonicalProjection: proof.CanonicalProjection(),
			}
		}
		confirmed, err = newLegacyConfirmedOutcomeSet(registry, confirmedMap.ProjectionRoster(), proofs)
		expectedStimulusKind = input.Plan.Adapter().Domain.CanonicalStimulusKind()
	default:
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint projection mode is unknown")
	}
	if err != nil {
		return ChoicepointRecord{}, err
	}
	if input.OriginalStimulus.kind != expectedStimulusKind || input.MinimizedStimulus.kind != expectedStimulusKind {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint stimulus kind differs from its exact projection authority")
	}

	identity := choicepointIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "Choicepoint", Scenario: input.Scenario,
		WorldPlanBase64: base64.StdEncoding.EncodeToString(input.Plan.CanonicalBytes()), WorldPlanDigest: input.Plan.Digest().String(),
		ComparisonEnvelopeBase64:   base64.StdEncoding.EncodeToString(input.Envelope.CanonicalBytes()),
		ComparisonEnvelopeDigest:   input.Envelope.Digest().String(),
		ProjectionBindingBase64:    base64.StdEncoding.EncodeToString(input.Plan.ProjectionDefinitionBinding().CanonicalBytes()),
		ProjectionDefinitionDigest: input.Plan.ProjectionDefinitionDigest().String(),
		CandidateBindingsBase64:    make([]string, len(bindings)), CandidateExecutionKeys: make([]string, len(bindings)),
		CandidateReveals:               make([]candidateRevealIdentity, len(reveals)),
		OriginalStimulusKind:           input.OriginalStimulus.kind,
		OriginalStimulusBase64:         base64.StdEncoding.EncodeToString(input.OriginalStimulus.canonical),
		OriginalStimulusDigest:         input.OriginalStimulus.digest.String(),
		MinimizedStimulusKind:          input.MinimizedStimulus.kind,
		MinimizedStimulusBase64:        base64.StdEncoding.EncodeToString(input.MinimizedStimulus.canonical),
		MinimizedStimulusDigest:        input.MinimizedStimulus.digest.String(),
		FreshConfirmationBase64:        base64.StdEncoding.EncodeToString(input.Confirmation.CanonicalBytes()),
		FreshConfirmationDigest:        input.Confirmation.Digest().String(),
		ConfirmedOutcomeMapDigest:      input.Confirmation.ConfirmedArtifactDigest().String(),
		ConfirmedPreservationMapDigest: input.Confirmation.ConfirmedMap().PreservationDigest().String(),
		ChoiceProjectionMode:           mode.wire(), EvidenceReceipts: make([]domain.ReceiptWire, len(receipts)),
		EvidenceStatusWhenEmpty: domain.Unreceipted(), ReadinessScope: choicepointScope,
		PresentationProvenanceNonclaim: choicepointPresentationNote,
	}
	for index, binding := range bindings {
		identity.CandidateBindingsBase64[index] = base64.StdEncoding.EncodeToString(binding.CanonicalBytes())
		identity.CandidateExecutionKeys[index] = binding.Key().String()
	}
	for index, reveal := range reveals {
		identity.CandidateReveals[index] = candidateRevealIdentity{
			CandidateExecutionKey: reveal.CandidateExecutionKey.String(), DisplayRef: reveal.DisplayRef,
			ProducerMetadata: reveal.ProducerMetadata,
		}
	}
	for index, receipt := range receipts {
		identity.EvidenceReceipts[index] = receipt.Wire()
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("Choicepoint", identity)
	if err != nil || len(canonicalBytes) > canon.MaxInputBytes {
		return ChoicepointRecord{}, refusal(CodeInputLimitExceeded, "choicepoint exceeds the durable canonical object ceiling")
	}
	if expected != nil && !bytes.Equal(expected, canonicalBytes) {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint wire does not reconstruct exactly")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return ChoicepointRecord{}, err
	}
	return ChoicepointRecord{
		digest: digest, canonicalBytes: canonicalBytes, scenario: input.Scenario, plan: input.Plan,
		envelope: input.Envelope, bindings: bindings, original: input.OriginalStimulus,
		minimized: input.MinimizedStimulus, confirmation: input.Confirmation,
		confirmed: confirmed, mode: mode, reveals: reveals, receipts: receipts,
	}, nil
}

func validateChoicepointLineage(input ChoicepointInput) error {
	if input.Scenario == "" || len(input.Scenario) > maxChoicepointScenarioBytes || !utf8.ValidString(input.Scenario) ||
		strings.TrimSpace(input.Scenario) == "" || containsControl(input.Scenario) ||
		!input.Plan.Digest().Valid() || len(input.Plan.CanonicalBytes()) == 0 ||
		!input.Envelope.Digest().Valid() || len(input.Envelope.CanonicalBytes()) == 0 ||
		!input.OriginalStimulus.valid() || !input.MinimizedStimulus.valid() || !input.Confirmation.Valid() {
		return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint input is incomplete or outside the bounded profile")
	}
	record := input.Confirmation
	originalMap, reducedMap, confirmedMap := record.OriginalBaselineMap(), record.ReducedBaselineMap(), record.ConfirmedMap()
	if record.PlanDigest() != input.Plan.Digest() || input.Plan.ComparisonEnvelopeDigest() != input.Envelope.Digest() ||
		originalMap.PlanDigest() != input.Plan.Digest() || reducedMap.PlanDigest() != input.Plan.Digest() ||
		confirmedMap.PlanDigest() != input.Plan.Digest() || originalMap.EnvelopeDigest() != input.Envelope.Digest() ||
		reducedMap.EnvelopeDigest() != input.Envelope.Digest() || confirmedMap.EnvelopeDigest() != input.Envelope.Digest() ||
		originalMap.ProjectionDefinitionDigest() != input.Plan.ProjectionDefinitionDigest() ||
		reducedMap.ProjectionDefinitionDigest() != input.Plan.ProjectionDefinitionDigest() ||
		confirmedMap.ProjectionDefinitionDigest() != input.Plan.ProjectionDefinitionDigest() ||
		originalMap.CapturePolicyDigest() != input.Plan.CapturePolicyDigest() ||
		reducedMap.CapturePolicyDigest() != input.Plan.CapturePolicyDigest() ||
		confirmedMap.CapturePolicyDigest() != input.Plan.CapturePolicyDigest() {
		return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint plan, envelope, projection, capture, or confirmation lineage differs")
	}
	if input.OriginalStimulus.kind != input.MinimizedStimulus.kind ||
		input.OriginalStimulus.digest != originalMap.StimulusDigest() || input.MinimizedStimulus.digest != reducedMap.StimulusDigest() ||
		reducedMap.StimulusDigest() != confirmedMap.StimulusDigest() {
		return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint stimulus bytes do not bind the exact original/minimized lineage")
	}
	if assessment := compare.AssessPreservation(reducedMap, confirmedMap); !assessment.Valid() || assessment.Relation() != compare.PreservationEqual {
		return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint confirmation does not preserve the exact labeled map")
	}
	roster := confirmedMap.CandidateRoster()
	if len(input.CandidateBindings) != len(roster) || len(input.CandidateReveals) != len(roster) {
		return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint bindings and reveals must cover the confirmed roster exactly")
	}
	bindings := append([]domain.CandidateExecutionBinding(nil), input.CandidateBindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Key().String() < bindings[j].Key().String() })
	for index, binding := range bindings {
		identity := binding.Identity()
		if !binding.Valid() || binding.Key() != roster[index] || identity.WorldPlanDigest != input.Plan.Digest() ||
			identity.MaterializationPolicyDigest != input.Plan.MaterializationPolicyDigest() ||
			identity.AdapterDigest != input.Plan.AdapterDigest() || identity.RunnerDigest != input.Plan.Adapter().RunnerDigest ||
			identity.ProjectionDefinitionDigest != input.Plan.ProjectionDefinitionDigest() ||
			(index > 0 && binding.Key().String() <= bindings[index-1].Key().String()) {
			return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint candidate binding differs from the exact plan roster")
		}
	}
	reveals := append([]CandidateReveal(nil), input.CandidateReveals...)
	sort.Slice(reveals, func(i, j int) bool {
		return reveals[i].CandidateExecutionKey.String() < reveals[j].CandidateExecutionKey.String()
	})
	for index, reveal := range reveals {
		if reveal.CandidateExecutionKey != roster[index] || reveal.DisplayRef == "" ||
			len(reveal.DisplayRef)+len(reveal.ProducerMetadata) > maxChoicepointRevealBytes ||
			!utf8.ValidString(reveal.DisplayRef) || !utf8.ValidString(reveal.ProducerMetadata) ||
			containsControl(reveal.DisplayRef) || containsControl(reveal.ProducerMetadata) {
			return refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint reveal provenance is incomplete or invalid")
		}
	}
	retained := len(input.Plan.CanonicalBytes()) + len(input.Envelope.CanonicalBytes()) +
		len(input.Plan.ProjectionDefinitionBinding().CanonicalBytes()) + len(input.OriginalStimulus.canonical) +
		len(input.MinimizedStimulus.canonical) + len(input.Confirmation.CanonicalBytes())
	for _, binding := range bindings {
		retained += len(binding.CanonicalBytes())
	}
	if retained > maxChoicepointNestedRawBytes {
		return refusal(CodeInputLimitExceeded, "choicepoint nested evidence exceeds the durable v1 resource profile")
	}
	return nil
}

func normalizeChoicepointReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {
	if len(receipts) > canon.MaxContainerMembers {
		return nil, refusal(CodeInputLimitExceeded, "choicepoint receipt count exceeds the canonical container profile")
	}
	remaining := canon.MaxInputBytes
	for _, receipt := range receipts {
		wire := receipt.Wire()
		if !consumeReceiptWireBudget(wire, &remaining) {
			return nil, refusal(CodeInputLimitExceeded, "choicepoint receipt payload exceeds the canonical input profile")
		}
	}
	type keyedReceipt struct {
		receipt domain.ReceiptReference
		key     string
	}
	keyed := make([]keyedReceipt, len(receipts))
	for index, receipt := range receipts {
		wire := receipt.Wire()
		rebuilt, err := domain.ReceiptFromWire(wire)
		if err != nil || rebuilt.Wire() != wire {
			return nil, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint receipt reference is invalid")
		}
		keyed[index] = keyedReceipt{receipt: receipt, key: receiptIdentityKey(receipt)}
	}
	sort.Slice(keyed, func(i, j int) bool { return keyed[i].key < keyed[j].key })
	result := make([]domain.ReceiptReference, len(keyed))
	for index, entry := range keyed {
		if index > 0 && entry.key == keyed[index-1].key {
			return nil, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint receipt reference is duplicated")
		}
		result[index] = entry.receipt
	}
	return result, nil
}

func consumeReceiptWireBudget(wire domain.ReceiptWire, remaining *int) bool {
	if remaining == nil || *remaining < 0 {
		return false
	}
	for _, member := range [...]string{wire.Authority, wire.GradeVerbatim, wire.CommitOID, wire.CommandDigest} {
		if len(member) > *remaining {
			return false
		}
		*remaining -= len(member)
	}
	return true
}

func ParseChoicepointRecord(exact []byte) (ChoicepointRecord, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint wire is outside the canonical profile")
	}
	canonicalBytes, err := value.CanonicalChecked()
	members, object := value.Members()
	if err != nil || !bytes.Equal(canonicalBytes, exact) || !object || len(members) != choicepointMemberCount {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint wire is nonexact or has unknown/missing members")
	}
	var identity choicepointIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint typed wire decode failed")
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "Choicepoint" ||
		identity.EvidenceStatusWhenEmpty != domain.Unreceipted() ||
		identity.ReadinessScope != choicepointScope || identity.PresentationProvenanceNonclaim != choicepointPresentationNote {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint closed wire facts disagree")
	}
	var mode choicepointMode
	switch identity.ChoiceProjectionMode {
	case legacyWholeProjectionMode:
		mode = choicepointLegacyWhole // MUTANT_P07B_LEGACY_REBUILT_PORTABLE
	case portableProjectionMode:
		mode = choicepointPortable
	default:
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint projection mode is unknown")
	}
	projectionBytes, err := decodeChoiceBase64(identity.ProjectionBindingBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	projection, err := domain.ParseProjectionDefinitionBinding(projectionBytes)
	if err != nil || projection.Digest().String() != identity.ProjectionDefinitionDigest {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint projection binding differs")
	}
	planBytes, err := decodeChoiceBase64(identity.WorldPlanBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	plan, err := domain.ParseWorldPlan(planBytes, projection)
	if err != nil || plan.Digest().String() != identity.WorldPlanDigest {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint world plan differs")
	}
	envelopeBytes, err := decodeChoiceBase64(identity.ComparisonEnvelopeBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	envelope, err := domain.ParseComparisonEnvelope(envelopeBytes)
	if err != nil || envelope.Digest().String() != identity.ComparisonEnvelopeDigest {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint comparison envelope differs")
	}
	confirmationBytes, err := decodeChoiceBase64(identity.FreshConfirmationBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	confirmationRecord, err := confirmation.ParseRecord(confirmationBytes)
	if err != nil || confirmationRecord.Digest().String() != identity.FreshConfirmationDigest ||
		confirmationRecord.ConfirmedArtifactDigest().String() != identity.ConfirmedOutcomeMapDigest ||
		confirmationRecord.ConfirmedMap().PreservationDigest().String() != identity.ConfirmedPreservationMapDigest {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint fresh confirmation differs")
	}
	original, err := parseChoiceArtifact(identity.OriginalStimulusKind, identity.OriginalStimulusDigest, identity.OriginalStimulusBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	minimized, err := parseChoiceArtifact(identity.MinimizedStimulusKind, identity.MinimizedStimulusDigest, identity.MinimizedStimulusBase64)
	if err != nil {
		return ChoicepointRecord{}, err
	}
	if len(identity.CandidateBindingsBase64) != len(identity.CandidateExecutionKeys) {
		return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint candidate binding lists differ")
	}
	bindings := make([]domain.CandidateExecutionBinding, len(identity.CandidateBindingsBase64))
	for index, encoded := range identity.CandidateBindingsBase64 {
		bindingBytes, decodeErr := decodeChoiceBase64(encoded)
		if decodeErr != nil {
			return ChoicepointRecord{}, decodeErr
		}
		bindings[index], err = domain.ParseCandidateExecutionBinding(bindingBytes)
		if err != nil || bindings[index].Key().String() != identity.CandidateExecutionKeys[index] ||
			(index > 0 && identity.CandidateExecutionKeys[index] <= identity.CandidateExecutionKeys[index-1]) {
			return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint candidate binding order differs")
		}
	}
	reveals := make([]CandidateReveal, len(identity.CandidateReveals))
	for index, reveal := range identity.CandidateReveals {
		candidate, parseErr := domain.ParseCandidateExecutionKey(reveal.CandidateExecutionKey)
		if parseErr != nil || (index > 0 && reveal.CandidateExecutionKey <= identity.CandidateReveals[index-1].CandidateExecutionKey) {
			return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint reveal order differs")
		}
		reveals[index] = CandidateReveal{CandidateExecutionKey: candidate, DisplayRef: reveal.DisplayRef, ProducerMetadata: reveal.ProducerMetadata}
	}
	receipts := make([]domain.ReceiptReference, len(identity.EvidenceReceipts))
	for index, wire := range identity.EvidenceReceipts {
		receipts[index], err = domain.ReceiptFromWire(wire)
		if err != nil || (index > 0 && receiptIdentityKey(receipts[index]) <= receiptIdentityKey(receipts[index-1])) {
			return ChoicepointRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint receipt order differs")
		}
	}
	return buildChoicepointRecord(ChoicepointInput{
		Scenario: identity.Scenario, Plan: plan, Envelope: envelope, CandidateBindings: bindings,
		OriginalStimulus: original, MinimizedStimulus: minimized, Confirmation: confirmationRecord,
		CandidateReveals: reveals, EvidenceReceipts: receipts,
	}, mode, exact)
}

func (r ChoicepointRecord) Valid() bool {
	parsed, err := ParseChoicepointRecord(r.canonicalBytes)
	return err == nil && parsed.digest == r.digest && bytes.Equal(parsed.canonicalBytes, r.canonicalBytes)
}

func (r ChoicepointRecord) Digest() domain.Digest             { return r.digest }
func (r ChoicepointRecord) CanonicalBytes() []byte            { return append([]byte(nil), r.canonicalBytes...) }
func (r ChoicepointRecord) Scenario() string                  { return r.scenario }
func (r ChoicepointRecord) ConfirmationDigest() domain.Digest { return r.confirmation.Digest() }
func (r ChoicepointRecord) ConfirmationRecord() confirmation.Record {
	parsed, _ := confirmation.ParseRecord(r.confirmation.CanonicalBytes())
	return parsed
}
func (r ChoicepointRecord) ConfirmedOutcomeMapDigest() compare.OutcomeArtifactDigest {
	return r.confirmation.ConfirmedArtifactDigest()
}

func parseChoiceArtifact(kind, rawDigest, encoded string) (CanonicalArtifact, error) {
	digest, err := domain.ParseDigest(rawDigest)
	if err != nil {
		return CanonicalArtifact{}, err
	}
	body, err := decodeChoiceBase64(encoded)
	if err != nil {
		return CanonicalArtifact{}, err
	}
	return NewCanonicalArtifact(kind, digest, body)
}

func decodeChoiceBase64(raw string) ([]byte, error) {
	body, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || base64.StdEncoding.EncodeToString(body) != raw || len(body) == 0 {
		return nil, refusal(CodeInvalidConfirmedOutcomeSet, "choicepoint nested bytes are noncanonical base64")
	}
	return body, nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func receiptIdentityKey(receipt domain.ReceiptReference) string {
	wire := receipt.Wire()
	return lengthPrefix(wire.Authority) + lengthPrefix(wire.CommitOID) +
		lengthPrefix(wire.CommandDigest) + lengthPrefix(wire.GradeVerbatim)
}
