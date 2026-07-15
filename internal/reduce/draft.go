package reduce

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type completedSweepEvaluationIdentity struct {
	EvaluationID                  string   `json:"evaluation_id"`
	ProposalDigest                string   `json:"proposal_digest"`
	NeighborStimulusDigest        string   `json:"neighbor_stimulus_digest"`
	NeighborMeasureDigest         string   `json:"neighbor_measure_digest"`
	ObservedOutcomeMapDigest      string   `json:"observed_outcome_map_digest"`
	ObservedPreservationMapDigest string   `json:"observed_preservation_map_digest"`
	Decision                      string   `json:"decision"`
	ReasonCode                    string   `json:"reason_code"`
	BatchDigests                  []string `json:"batch_digests"`
	AttemptDigests                []string `json:"attempt_digests"`
	WorldDigests                  []string `json:"world_digests"`
	ObservationDigests            []string `json:"observation_digests"`
}

type completedSweepDraftIdentity struct {
	SchemaVersion                 string                             `json:"schema_version"`
	Kind                          string                             `json:"kind"`
	RunDigest                     string                             `json:"run_digest"`
	TranscriptDigest              string                             `json:"transcript_digest"`
	BaselineOutcomeMapDigest      string                             `json:"baseline_outcome_map_digest"`
	BaselinePreservationMapDigest string                             `json:"baseline_preservation_map_digest"`
	CurrentStimulusDigest         string                             `json:"current_stimulus_digest"`
	CurrentMeasureDefinition      string                             `json:"current_measure_definition_digest"`
	CurrentMeasureComponents      []int64                            `json:"current_measure_components"`
	CurrentMeasureDigest          string                             `json:"current_measure_digest"`
	ReducerSetDigest              string                             `json:"reducer_set_digest"`
	EnumerationCompleted          bool                               `json:"enumeration_completed"`
	Cancelled                     bool                               `json:"cancelled"`
	BudgetExhausted               bool                               `json:"budget_exhausted"`
	BaselineStale                 bool                               `json:"baseline_stale"`
	EnumeratedNeighborDigests     []string                           `json:"enumerated_neighbor_digests"`
	Evaluations                   []completedSweepEvaluationIdentity `json:"evaluations"`
}

// CompletedSweepDraft is pure logical authority. Possessing or copying it is
// explicitly insufficient for ONE_MINIMAL_UNDER; only a store-issued opaque
// completion capability over these exact bytes can cross that boundary.
type CompletedSweepDraft struct {
	digest               domain.Digest
	canonicalBytes       []byte
	runDigest            domain.Digest
	transcriptDigest     domain.Digest
	currentStimulus      domain.Digest
	currentMeasure       Measure
	reducerSetDigest     domain.Digest
	baselineMapDigest    compare.OutcomeArtifactDigest
	baselinePreservation compare.PreservationMapDigest
	neighborDigests      []domain.Digest
}

func NewCompletedSweepDraft(
	runDigest, transcriptDigest domain.Digest,
	sweep LogicalCompleteSweep,
	neighbors []Neighbor,
	evaluations []Evaluation,
) (CompletedSweepDraft, error) {
	if !runDigest.Valid() || !transcriptDigest.Valid() || !validLogicalCompleteSweep(sweep) {
		return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_DRAFT"}
	}
	byStimulus := make(map[domain.Digest]Neighbor, len(neighbors))
	for _, neighbor := range neighbors {
		if !validNeighbor(neighbor) || neighbor.currentStimulusDigest != sweep.currentStimulus ||
			neighbor.currentMeasure.digest != sweep.currentMeasure.digest || neighbor.reducerSetDigest != sweep.reducerSetDigest {
			return CompletedSweepDraft{}, &domain.Error{Code: "COMPLETED_SWEEP_NEIGHBOR_BINDING_MISMATCH"}
		}
		if _, duplicate := byStimulus[neighbor.stimulusDigest]; duplicate {
			return CompletedSweepDraft{}, &domain.Error{Code: "DUPLICATE_COMPLETED_SWEEP_NEIGHBOR"}
		}
		byStimulus[neighbor.stimulusDigest] = neighbor
	}
	if len(byStimulus) != len(sweep.neighborDigests) || len(evaluations) != len(sweep.neighborDigests) {
		return CompletedSweepDraft{}, &domain.Error{Code: "COMPLETED_SWEEP_SET_MISMATCH"}
	}

	identities := make([]completedSweepEvaluationIdentity, 0, len(evaluations))
	seenStimuli := map[domain.Digest]struct{}{}
	for _, evaluation := range evaluations {
		neighbor, present := byStimulus[evaluation.neighbor.stimulusDigest]
		if !present || !sameNeighbor(neighbor, evaluation.neighbor) || evaluation.purpose != domain.AttemptFinalSweep ||
			evaluation.decision != Changes || evaluation.baselineOutcomeMapDigest != sweep.baselineMapDigest ||
			evaluation.baselinePreservationDigest != sweep.baselinePreservationDigest || !evaluation.logicalNonReuseWithBaseline {
			return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_EVALUATION"}
		}
		if _, duplicate := seenStimuli[neighbor.stimulusDigest]; duplicate {
			return CompletedSweepDraft{}, &domain.Error{Code: "DUPLICATE_COMPLETED_SWEEP_EVALUATION"}
		}
		seenStimuli[neighbor.stimulusDigest] = struct{}{}
		outcome, outcomePresent := evaluation.ObservedOutcomeMapDigest()
		preservation, preservationPresent := evaluation.ObservedPreservationDigest()
		if !outcomePresent || !preservationPresent {
			return CompletedSweepDraft{}, &domain.Error{Code: "MISSING_COMPLETED_SWEEP_MAP_DIGEST"}
		}
		identities = append(identities, completedSweepEvaluationIdentity{
			EvaluationID: evaluation.id, ProposalDigest: neighbor.digest.String(),
			NeighborStimulusDigest: neighbor.stimulusDigest.String(), NeighborMeasureDigest: neighbor.measure.digest.String(),
			ObservedOutcomeMapDigest: outcome.String(), ObservedPreservationMapDigest: preservation.String(),
			Decision: string(evaluation.decision), ReasonCode: evaluation.reasonCode,
			BatchDigests:       digestStrings(evaluation.observedBatchDigests),
			AttemptDigests:     digestStrings(evaluation.observedAttemptDigests),
			WorldDigests:       digestStrings(evaluation.observedWorldDigests),
			ObservationDigests: digestStrings(evaluation.observedObservationDigests),
		})
	}
	sort.Slice(identities, func(left, right int) bool {
		return identities[left].NeighborStimulusDigest < identities[right].NeighborStimulusDigest
	})
	measureComponents := make([]int64, len(sweep.currentMeasure.components))
	for index, component := range sweep.currentMeasure.components {
		measureComponents[index] = int64(component)
	}
	identity := completedSweepDraftIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CompletedSweepDraft", RunDigest: runDigest.String(),
		TranscriptDigest: transcriptDigest.String(), BaselineOutcomeMapDigest: sweep.baselineMapDigest.String(),
		BaselinePreservationMapDigest: sweep.baselinePreservationDigest.String(), CurrentStimulusDigest: sweep.currentStimulus.String(),
		CurrentMeasureDefinition: sweep.currentMeasure.definitionDigest.String(), CurrentMeasureComponents: measureComponents,
		CurrentMeasureDigest: sweep.currentMeasure.digest.String(), ReducerSetDigest: sweep.reducerSetDigest.String(),
		EnumerationCompleted: true, Cancelled: false, BudgetExhausted: false, BaselineStale: false,
		EnumeratedNeighborDigests: digestStrings(sweep.neighborDigests), Evaluations: identities,
	}
	digest, canonicalBytes, err := digestTyped("CompletedSweepDraft", identity)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	return CompletedSweepDraft{
		digest: digest, canonicalBytes: canonicalBytes, runDigest: runDigest, transcriptDigest: transcriptDigest,
		currentStimulus: sweep.currentStimulus, currentMeasure: sweep.currentMeasure, reducerSetDigest: sweep.reducerSetDigest,
		baselineMapDigest: sweep.baselineMapDigest, baselinePreservation: sweep.baselinePreservationDigest,
		neighborDigests: append([]domain.Digest(nil), sweep.neighborDigests...),
	}, nil
}

func validLogicalCompleteSweep(sweep LogicalCompleteSweep) bool {
	return sweep.currentStimulus.Valid() && sweep.currentMeasure.Valid() && sweep.reducerSetDigest.Valid() &&
		sweep.baselineMapDigest.Valid() && sweep.baselinePreservationDigest.Valid() && !hasInvalidOrDuplicate(sweep.neighborDigests)
}

func digestStrings(values []domain.Digest) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.String()
	}
	return result
}

func ParseCompletedSweepDraft(exactCanonicalBytes []byte) (CompletedSweepDraft, error) {
	value, err := canon.Parse(exactCanonicalBytes)
	if err != nil {
		return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_BYTES", Detail: err.Error()}
	}
	canonicalBytes, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonicalBytes, exactCanonicalBytes) {
		return CompletedSweepDraft{}, &domain.Error{Code: "NONCANONICAL_COMPLETED_SWEEP_BYTES"}
	}
	var identity completedSweepDraftIdentity
	if err := json.Unmarshal(exactCanonicalBytes, &identity); err != nil {
		return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_WIRE", Detail: err.Error()}
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "CompletedSweepDraft" ||
		!identity.EnumerationCompleted || identity.Cancelled || identity.BudgetExhausted || identity.BaselineStale ||
		identity.EnumeratedNeighborDigests == nil || identity.Evaluations == nil {
		return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_STATE"}
	}
	runDigest, err := domain.ParseDigest(identity.RunDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	transcriptDigest, err := domain.ParseDigest(identity.TranscriptDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	currentStimulus, err := domain.ParseDigest(identity.CurrentStimulusDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	measureDefinition, err := domain.ParseDigest(identity.CurrentMeasureDefinition)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	components := make([]uint64, len(identity.CurrentMeasureComponents))
	for index, component := range identity.CurrentMeasureComponents {
		if component < 0 {
			return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_MEASURE"}
		}
		components[index] = uint64(component)
	}
	measure, err := NewMeasure(measureDefinition, components)
	if err != nil || measure.digest.String() != identity.CurrentMeasureDigest {
		return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_MEASURE"}
	}
	reducerSetDigest, err := domain.ParseDigest(identity.ReducerSetDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	baselineMapDigest, err := compare.ParseOutcomeArtifactDigest(identity.BaselineOutcomeMapDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	baselinePreservationDigest, err := compare.ParsePreservationMapDigest(identity.BaselinePreservationMapDigest)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	neighborDigests, err := parseSortedUniqueDigests(identity.EnumeratedNeighborDigests)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	if len(identity.Evaluations) != len(neighborDigests) {
		return CompletedSweepDraft{}, &domain.Error{Code: "COMPLETED_SWEEP_SET_MISMATCH"}
	}
	seenEvidence := map[string]struct{}{}
	for index, evaluation := range identity.Evaluations {
		if evaluation.Decision != string(Changes) || evaluation.EvaluationID == "" || evaluation.ReasonCode == "" ||
			evaluation.NeighborStimulusDigest != neighborDigests[index].String() {
			return CompletedSweepDraft{}, &domain.Error{Code: "INVALID_COMPLETED_SWEEP_EVALUATION"}
		}
		for _, raw := range []string{evaluation.ProposalDigest, evaluation.NeighborMeasureDigest,
			evaluation.ObservedOutcomeMapDigest, evaluation.ObservedPreservationMapDigest} {
			if _, err := domain.ParseDigest(raw); err != nil {
				return CompletedSweepDraft{}, err
			}
		}
		for _, set := range [][]string{evaluation.BatchDigests, evaluation.AttemptDigests, evaluation.WorldDigests, evaluation.ObservationDigests} {
			if len(set) == 0 {
				return CompletedSweepDraft{}, &domain.Error{Code: "MISSING_COMPLETED_SWEEP_EVIDENCE"}
			}
			for _, raw := range set {
				if _, err := domain.ParseDigest(raw); err != nil {
					return CompletedSweepDraft{}, err
				}
				if _, duplicate := seenEvidence[raw]; duplicate {
					return CompletedSweepDraft{}, &domain.Error{Code: "REUSED_COMPLETED_SWEEP_EVIDENCE"}
				}
				seenEvidence[raw] = struct{}{}
			}
		}
	}
	digest, canonicalBytes, err := digestTyped("CompletedSweepDraft", identity)
	if err != nil {
		return CompletedSweepDraft{}, err
	}
	if !bytes.Equal(canonicalBytes, exactCanonicalBytes) {
		return CompletedSweepDraft{}, &domain.Error{Code: "UNKNOWN_OR_NONEXACT_COMPLETED_SWEEP_FIELD"}
	}
	return CompletedSweepDraft{
		digest: digest, canonicalBytes: canonicalBytes, runDigest: runDigest, transcriptDigest: transcriptDigest,
		currentStimulus: currentStimulus, currentMeasure: measure, reducerSetDigest: reducerSetDigest,
		baselineMapDigest: baselineMapDigest, baselinePreservation: baselinePreservationDigest,
		neighborDigests: neighborDigests,
	}, nil
}

func parseSortedUniqueDigests(raw []string) ([]domain.Digest, error) {
	result := make([]domain.Digest, len(raw))
	for index, value := range raw {
		digest, err := domain.ParseDigest(value)
		if err != nil {
			return nil, err
		}
		if index > 0 && result[index-1].String() >= digest.String() {
			return nil, &domain.Error{Code: "UNSORTED_OR_DUPLICATE_COMPLETED_SWEEP_SET"}
		}
		result[index] = digest
	}
	return result, nil
}

func (d CompletedSweepDraft) Valid() bool {
	parsed, err := ParseCompletedSweepDraft(d.canonicalBytes)
	return err == nil && parsed.digest == d.digest && bytes.Equal(parsed.canonicalBytes, d.canonicalBytes)
}
func (d CompletedSweepDraft) Digest() domain.Digest                { return d.digest }
func (d CompletedSweepDraft) CanonicalBytes() []byte               { return append([]byte(nil), d.canonicalBytes...) }
func (d CompletedSweepDraft) RunDigest() domain.Digest             { return d.runDigest }
func (d CompletedSweepDraft) TranscriptDigest() domain.Digest      { return d.transcriptDigest }
func (d CompletedSweepDraft) CurrentStimulusDigest() domain.Digest { return d.currentStimulus }
func (d CompletedSweepDraft) CurrentMeasure() Measure              { return d.currentMeasure }
func (d CompletedSweepDraft) ReducerSetDigest() domain.Digest      { return d.reducerSetDigest }
func (d CompletedSweepDraft) BaselineMapDigest() compare.OutcomeArtifactDigest {
	return d.baselineMapDigest
}
func (d CompletedSweepDraft) BaselinePreservationMapDigest() compare.PreservationMapDigest {
	return d.baselinePreservation
}
func (d CompletedSweepDraft) NeighborDigests() []domain.Digest {
	return append([]domain.Digest(nil), d.neighborDigests...)
}
