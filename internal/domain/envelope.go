package domain

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
)

type MeasuredSource string

const (
	MeasuredWorldInstance   MeasuredSource = "WORLD_INSTANCE"
	MeasuredToolReceipt     MeasuredSource = "TOOL_RECEIPT"
	MeasuredFilesystemProbe MeasuredSource = "FILESYSTEM_PROBE"
	MeasuredProcessReceipt  MeasuredSource = "PROCESS_RECEIPT"
)

type MeasuredComparison string

const (
	CompareExact            MeasuredComparison = "EXACT"
	CompareRecordedOnly     MeasuredComparison = "RECORDED_ONLY"
	CompareRejectOnVariance MeasuredComparison = "REJECT_ON_VARIANCE"
)

type MeasuredDimension struct {
	Name       string             `json:"name"`
	Source     MeasuredSource     `json:"source"`
	Comparison MeasuredComparison `json:"comparison"`
}

type RequiredEqualDimension struct {
	Name string `json:"name"`
}

type Tolerance string

const (
	MayDifferRecorded           Tolerance = "MAY_DIFFER_AND_IS_RECORDED"
	ProjectedCapturePlaceholder Tolerance = "PROJECTED_PLACEHOLDER_IN_CAPTURE_ONLY"
)

type ToleratedDimension struct {
	Name      string    `json:"name"`
	Tolerance Tolerance `json:"tolerance"`
}

type RejectedDimension struct {
	Name       string `json:"name"`
	ReasonCode string `json:"reason_code"`
}

type ComparisonEnvelopeConfig struct {
	Version       string
	Measured      []MeasuredDimension
	RequiredEqual []RequiredEqualDimension
	Tolerated     []ToleratedDimension
	Rejected      []RejectedDimension
	Uncontrolled  []string
}

// ComparisonEnvelope is only a policy. It records how measured dimensions are
// to be handled; it cannot itself claim that any concrete worlds were equal.
type ComparisonEnvelope struct {
	digest         Digest
	canonicalBytes []byte
	config         ComparisonEnvelopeConfig
}

type toleratedIdentity struct {
	Name      string    `json:"name"`
	Tolerance Tolerance `json:"tolerance"`
	Nonclaim  string    `json:"nonclaim"`
}

type requiredIdentity struct {
	Name       string `json:"name"`
	Comparator string `json:"comparator"`
}

type comparisonEnvelopeIdentity struct {
	SchemaVersion                      string              `json:"schema_version"`
	Kind                               string              `json:"kind"`
	EnvelopeVersion                    string              `json:"envelope_version"`
	MeasuredDimensions                 []MeasuredDimension `json:"measured_dimensions"`
	RequiredEqualDimensions            []requiredIdentity  `json:"required_equal_dimensions"`
	ToleratedDimensions                []toleratedIdentity `json:"tolerated_dimensions"`
	RejectedDimensions                 []RejectedDimension `json:"rejected_dimensions"`
	UncontrolledDimensions             []string            `json:"uncontrolled_dimensions"`
	AdmissionClaim                     string              `json:"admission_claim"`
	BehavioralCompatibilityEstablished bool                `json:"behavioral_compatibility_established"`
	PlaceholderControlFlowIrrelevance  bool                `json:"placeholder_control_flow_irrelevance_established"`
}

func NewComparisonEnvelope(config ComparisonEnvelopeConfig) (ComparisonEnvelope, error) {
	if config.Version == "" || len(config.Measured) == 0 || len(config.RequiredEqual) == 0 || len(config.Uncontrolled) == 0 {
		return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "required dimension set is empty")
	}
	measured := make(map[string]MeasuredDimension, len(config.Measured))
	for _, dimension := range config.Measured {
		if dimension.Name == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "empty measured dimension")
		}
		if _, exists := measured[dimension.Name]; exists {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "duplicate measured dimension")
		}
		switch dimension.Source {
		case MeasuredWorldInstance, MeasuredToolReceipt, MeasuredFilesystemProbe, MeasuredProcessReceipt:
		default:
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "unknown measured source")
		}
		switch dimension.Comparison {
		case CompareExact, CompareRecordedOnly, CompareRejectOnVariance:
		default:
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "unknown measured comparison")
		}
		measured[dimension.Name] = dimension
	}

	dispositions := map[string]string{}
	claim := func(name, disposition string, want MeasuredComparison) error {
		dimension, exists := measured[name]
		if !exists {
			return refuse("INVALID_COMPARISON_ENVELOPE", disposition+" dimension is not measured")
		}
		if previous := dispositions[name]; previous != "" {
			return refuse("INVALID_COMPARISON_ENVELOPE", "overlapping dimension disposition")
		}
		if dimension.Comparison != want {
			return refuse("INVALID_COMPARISON_ENVELOPE", disposition+" disposition disagrees with measured comparator")
		}
		dispositions[name] = disposition
		return nil
	}
	for _, dimension := range config.RequiredEqual {
		if dimension.Name == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "empty required-equal dimension")
		}
		if err := claim(dimension.Name, "required", CompareExact); err != nil {
			return ComparisonEnvelope{}, err
		}
	}
	for _, dimension := range config.Tolerated {
		if dimension.Name == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "empty tolerated dimension")
		}
		if dimension.Tolerance != MayDifferRecorded && dimension.Tolerance != ProjectedCapturePlaceholder {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "unknown tolerance")
		}
		if err := claim(dimension.Name, "tolerated", CompareRecordedOnly); err != nil {
			return ComparisonEnvelope{}, err
		}
	}
	for _, dimension := range config.Rejected {
		if dimension.Name == "" || dimension.ReasonCode == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "invalid rejected dimension")
		}
		if err := claim(dimension.Name, "rejected", CompareRejectOnVariance); err != nil {
			return ComparisonEnvelope{}, err
		}
	}
	for name := range measured {
		if dispositions[name] == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "measured dimension has no disposition")
		}
	}
	seenUncontrolled := map[string]struct{}{}
	for _, dimension := range config.Uncontrolled {
		if dimension == "" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "empty uncontrolled dimension")
		}
		if _, duplicate := seenUncontrolled[dimension]; duplicate {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "duplicate uncontrolled dimension")
		}
		if _, isMeasured := measured[dimension]; isMeasured {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "measured dimension cannot be uncontrolled")
		}
		seenUncontrolled[dimension] = struct{}{}
	}

	copyConfig := cloneEnvelopeConfig(config)
	sort.Slice(copyConfig.Measured, func(i, j int) bool { return copyConfig.Measured[i].Name < copyConfig.Measured[j].Name })
	sort.Slice(copyConfig.RequiredEqual, func(i, j int) bool { return copyConfig.RequiredEqual[i].Name < copyConfig.RequiredEqual[j].Name })
	sort.Slice(copyConfig.Tolerated, func(i, j int) bool { return copyConfig.Tolerated[i].Name < copyConfig.Tolerated[j].Name })
	sort.Slice(copyConfig.Rejected, func(i, j int) bool { return copyConfig.Rejected[i].Name < copyConfig.Rejected[j].Name })
	sort.Strings(copyConfig.Uncontrolled)

	required := make([]requiredIdentity, len(copyConfig.RequiredEqual))
	for index, item := range copyConfig.RequiredEqual {
		required[index] = requiredIdentity{Name: item.Name, Comparator: "EXACT"}
	}
	tolerated := make([]toleratedIdentity, len(copyConfig.Tolerated))
	for index, item := range copyConfig.Tolerated {
		tolerated[index] = toleratedIdentity{
			Name:      item.Name,
			Tolerance: item.Tolerance,
			Nonclaim:  "CONTROL_FLOW_IRRELEVANCE_NOT_ESTABLISHED",
		}
	}
	identity := comparisonEnvelopeIdentity{
		SchemaVersion:                      SchemaVersion,
		Kind:                               "ComparisonEnvelope",
		EnvelopeVersion:                    copyConfig.Version,
		MeasuredDimensions:                 copyConfig.Measured,
		RequiredEqualDimensions:            required,
		ToleratedDimensions:                tolerated,
		RejectedDimensions:                 copyConfig.Rejected,
		UncontrolledDimensions:             copyConfig.Uncontrolled,
		AdmissionClaim:                     "MEASURED_ASSESSMENT_ONLY",
		BehavioralCompatibilityEstablished: false,
		PlaceholderControlFlowIrrelevance:  false,
	}
	digest, canonicalBytes, err := digestTyped("ComparisonEnvelope", identity)
	if err != nil {
		return ComparisonEnvelope{}, err
	}
	return ComparisonEnvelope{digest: digest, canonicalBytes: canonicalBytes, config: copyConfig}, nil
}

func cloneEnvelopeConfig(config ComparisonEnvelopeConfig) ComparisonEnvelopeConfig {
	result := config
	result.Measured = append([]MeasuredDimension(nil), config.Measured...)
	result.RequiredEqual = append([]RequiredEqualDimension(nil), config.RequiredEqual...)
	result.Tolerated = append([]ToleratedDimension(nil), config.Tolerated...)
	result.Rejected = append([]RejectedDimension(nil), config.Rejected...)
	result.Uncontrolled = append([]string(nil), config.Uncontrolled...)
	return result
}

func (e ComparisonEnvelope) Digest() Digest { return e.digest }

func (e ComparisonEnvelope) CanonicalBytes() []byte {
	return append([]byte(nil), e.canonicalBytes...)
}

func (e ComparisonEnvelope) Config() ComparisonEnvelopeConfig {
	return cloneEnvelopeConfig(e.config)
}

// ParseComparisonEnvelope strictly reconstructs the closed policy object.
func ParseComparisonEnvelope(exact []byte) (ComparisonEnvelope, error) {
	var identity comparisonEnvelopeIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "wire decode failed")
	}
	if identity.SchemaVersion != SchemaVersion || identity.Kind != "ComparisonEnvelope" ||
		identity.AdmissionClaim != "MEASURED_ASSESSMENT_ONLY" || identity.BehavioralCompatibilityEstablished ||
		identity.PlaceholderControlFlowIrrelevance {
		return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "closed wire facts disagree")
	}
	config := ComparisonEnvelopeConfig{
		Version: identity.EnvelopeVersion, Measured: identity.MeasuredDimensions,
		RequiredEqual: make([]RequiredEqualDimension, len(identity.RequiredEqualDimensions)),
		Tolerated:     make([]ToleratedDimension, len(identity.ToleratedDimensions)),
		Rejected:      identity.RejectedDimensions, Uncontrolled: identity.UncontrolledDimensions,
	}
	for index, required := range identity.RequiredEqualDimensions {
		if required.Comparator != "EXACT" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "required comparator differs")
		}
		config.RequiredEqual[index] = RequiredEqualDimension{Name: required.Name}
	}
	for index, tolerated := range identity.ToleratedDimensions {
		if tolerated.Nonclaim != "CONTROL_FLOW_IRRELEVANCE_NOT_ESTABLISHED" {
			return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "tolerance nonclaim differs")
		}
		config.Tolerated[index] = ToleratedDimension{Name: tolerated.Name, Tolerance: tolerated.Tolerance}
	}
	rebuilt, err := NewComparisonEnvelope(config)
	if err != nil || !bytes.Equal(rebuilt.canonicalBytes, exact) {
		return ComparisonEnvelope{}, refuse("INVALID_COMPARISON_ENVELOPE", "wire is nonexact")
	}
	return rebuilt, nil
}

type MeasurementValue struct {
	Name   string
	Source MeasuredSource
	Value  canon.Value
}

type measuredValue struct {
	source    MeasuredSource
	canonical []byte
}

// InstanceMeasurements is an exact, complete measured-dimension row for one
// runtime subject. It is structural input from U2, not proof that the runtime
// measurement was honestly obtained.
type InstanceMeasurements struct {
	digest                Digest
	canonicalBytes        []byte
	envelopeDigest        Digest
	subjectDigest         Digest
	planDigest            Digest
	stimulusDigest        Digest
	attemptArtifactDigest Digest
	candidateKey          CandidateExecutionKey
	purpose               AttemptPurpose
	scheduleOrdinal       int
	requiredFreshTrials   int
	values                map[string]measuredValue
}

func NewInstanceMeasurements(
	envelope ComparisonEnvelope,
	world WorldInstance,
	values []MeasurementValue,
) (InstanceMeasurements, error) {
	if !envelope.Digest().Valid() || !world.Digest().Valid() {
		return InstanceMeasurements{}, refuse("INVALID_INSTANCE_MEASUREMENTS", "missing envelope or subject")
	}
	if world.EnvelopeDigest() != envelope.Digest() {
		return InstanceMeasurements{}, refuse("MEASUREMENT_ENVELOPE_PLAN_MISMATCH", "envelope is not the one bound by the world plan")
	}
	subjectDigest := world.Digest()
	config := envelope.Config()
	if len(values) != len(config.Measured) {
		return InstanceMeasurements{}, refuse("INCOMPLETE_INSTANCE_MEASUREMENTS", "measured dimension coverage differs from policy")
	}
	expected := make(map[string]MeasuredSource, len(config.Measured))
	for _, dimension := range config.Measured {
		expected[dimension.Name] = dimension.Source
	}
	stored := make(map[string]measuredValue, len(values))
	type valueIdentity struct {
		Name          string `json:"name"`
		Source        string `json:"source"`
		CanonicalJSON string `json:"canonical_json"`
	}
	identityValues := make([]valueIdentity, 0, len(values))
	for _, value := range values {
		source, exists := expected[value.Name]
		if !exists || source != value.Source {
			return InstanceMeasurements{}, refuse("INVALID_INSTANCE_MEASUREMENTS", "unknown dimension or source mismatch")
		}
		if _, duplicate := stored[value.Name]; duplicate {
			return InstanceMeasurements{}, refuse("INVALID_INSTANCE_MEASUREMENTS", "duplicate measured dimension")
		}
		canonical, canonicalErr := value.Value.CanonicalChecked()
		if canonicalErr != nil {
			return InstanceMeasurements{}, canonicalErr
		}
		stored[value.Name] = measuredValue{source: value.Source, canonical: append([]byte(nil), canonical...)}
		identityValues = append(identityValues, valueIdentity{value.Name, string(value.Source), string(canonical)})
	}
	sort.Slice(identityValues, func(i, j int) bool { return identityValues[i].Name < identityValues[j].Name })
	identity := struct {
		SchemaVersion  string          `json:"schema_version"`
		Kind           string          `json:"kind"`
		EnvelopeDigest string          `json:"comparison_envelope_digest"`
		SubjectDigest  string          `json:"measurement_subject_digest"`
		Values         []valueIdentity `json:"values"`
	}{SchemaVersion, "InstanceMeasurements", envelope.Digest().String(), subjectDigest.String(), identityValues}
	digest, canonicalBytes, err := digestTyped("InstanceMeasurements", identity)
	if err != nil {
		return InstanceMeasurements{}, err
	}
	return InstanceMeasurements{
		digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...),
		envelopeDigest: envelope.Digest(), subjectDigest: subjectDigest, planDigest: world.PlanDigest(),
		stimulusDigest: world.StimulusDigest(), attemptArtifactDigest: world.AttemptArtifactDigest(),
		candidateKey: world.CandidateKey(), purpose: world.Purpose(), scheduleOrdinal: world.ScheduleOrdinal(),
		requiredFreshTrials: world.RequiredFreshTrials(), values: stored,
	}, nil
}

func (m InstanceMeasurements) Digest() Digest        { return m.digest }
func (m InstanceMeasurements) SubjectDigest() Digest { return m.subjectDigest }

func (m InstanceMeasurements) CanonicalBytes() []byte {
	return append([]byte(nil), m.canonicalBytes...)
}

type ComparisonAssessment interface {
	comparisonAssessment()
	Digest() Digest
	EnvelopeDigest() Digest
	CanonicalBytes() []byte
}

type AdmissionToken struct {
	digest                    Digest
	comparisonAdmissionDigest Digest
	comparisonBasisDigest     Digest
	envelopeDigest            Digest
	measurementDigest         Digest
	subjectDigest             Digest
	planDigest                Digest
	stimulusDigest            Digest
	purpose                   AttemptPurpose
	requiredFreshTrials       int
	candidateRoster           []CandidateExecutionKey
}

func (t AdmissionToken) Valid() bool {
	return t.digest.Valid() && t.comparisonAdmissionDigest.Valid() && t.comparisonBasisDigest.Valid() && t.envelopeDigest.Valid() &&
		t.measurementDigest.Valid() && t.subjectDigest.Valid() && t.planDigest.Valid() && t.stimulusDigest.Valid() &&
		t.purpose.Valid() && t.requiredFreshTrials >= 1 && t.requiredFreshTrials <= 5 && validAdmissionRoster(t.candidateRoster)
}

func (t AdmissionToken) Digest() Digest                    { return t.digest }
func (t AdmissionToken) ComparisonAdmissionDigest() Digest { return t.comparisonAdmissionDigest }
func (t AdmissionToken) ComparisonBasisDigest() Digest     { return t.comparisonBasisDigest }
func (t AdmissionToken) EnvelopeDigest() Digest            { return t.envelopeDigest }
func (t AdmissionToken) MeasurementDigest() Digest         { return t.measurementDigest }
func (t AdmissionToken) SubjectDigest() Digest             { return t.subjectDigest }
func (t AdmissionToken) PlanDigest() Digest                { return t.planDigest }
func (t AdmissionToken) StimulusDigest() Digest            { return t.stimulusDigest }
func (t AdmissionToken) Purpose() AttemptPurpose           { return t.purpose }
func (t AdmissionToken) RequiredFreshTrials() int          { return t.requiredFreshTrials }
func (t AdmissionToken) CandidateRoster() []CandidateExecutionKey {
	return append([]CandidateExecutionKey(nil), t.candidateRoster...)
}

type AdmittedComparison struct {
	digest                Digest
	canonicalBytes        []byte
	comparisonBasisDigest Digest
	envelopeDigest        Digest
	planDigest            Digest
	stimulusDigest        Digest
	purpose               AttemptPurpose
	requiredFreshTrials   int
	candidateRoster       []CandidateExecutionKey
	measurementDigests    []Digest
	tokens                map[Digest]AdmissionToken
}

func (a AdmittedComparison) comparisonAssessment()  {}
func (a AdmittedComparison) Digest() Digest         { return a.digest }
func (a AdmittedComparison) EnvelopeDigest() Digest { return a.envelopeDigest }
func (a AdmittedComparison) CanonicalBytes() []byte {
	return append([]byte(nil), a.canonicalBytes...)
}
func (a AdmittedComparison) ComparisonBasisDigest() Digest { return a.comparisonBasisDigest }
func (a AdmittedComparison) CandidateRoster() []CandidateExecutionKey {
	return append([]CandidateExecutionKey(nil), a.candidateRoster...)
}

func (a AdmittedComparison) MeasurementDigests() []Digest {
	return append([]Digest(nil), a.measurementDigests...)
}

func (a AdmittedComparison) AdmissionFor(measurements InstanceMeasurements) (AdmissionToken, error) {
	if !a.digest.Valid() || !measurements.digest.Valid() || !validAdmissionRoster(a.candidateRoster) {
		return AdmissionToken{}, refuse("INVALID_COMPARISON_ADMISSION", "zero assessment or measurement")
	}
	token, exists := a.tokens[measurements.digest]
	if !exists || token.measurementDigest != measurements.digest || token.subjectDigest != measurements.subjectDigest ||
		measurements.planDigest != a.planDigest || measurements.stimulusDigest != a.stimulusDigest ||
		measurements.envelopeDigest != a.envelopeDigest || measurements.purpose != a.purpose ||
		measurements.requiredFreshTrials != a.requiredFreshTrials {
		return AdmissionToken{}, refuse("MEASUREMENTS_NOT_IN_COMPARISON", measurements.digest.String())
	}
	return token, nil
}

type RejectedComparison struct {
	digest         Digest
	canonicalBytes []byte
	envelopeDigest Digest
	reasonCodes    []string
}

func (r RejectedComparison) comparisonAssessment()  {}
func (r RejectedComparison) Digest() Digest         { return r.digest }
func (r RejectedComparison) EnvelopeDigest() Digest { return r.envelopeDigest }
func (r RejectedComparison) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r RejectedComparison) ReasonCodes() []string { return append([]string(nil), r.reasonCodes...) }

// AssessComparison is the sole constructor of admission tokens. Required and
// rejected dimensions are compared from exact canonical values; tolerated
// dimensions are recorded in InstanceMeasurements but do not block admission.
func AssessComparison(envelope ComparisonEnvelope, instances []InstanceMeasurements) (ComparisonAssessment, error) {
	if !envelope.Digest().Valid() || len(instances) < 2 || len(instances) > 4 {
		return nil, refuse("INVALID_COMPARISON_MATRIX", "comparison requires 2..4 measured instances")
	}
	binding := instances[0]
	if !binding.planDigest.Valid() || !binding.stimulusDigest.Valid() || !binding.purpose.Valid() ||
		binding.requiredFreshTrials < 1 || binding.requiredFreshTrials > 5 {
		return nil, refuse("INVALID_COMPARISON_MATRIX", "comparison binding is incomplete")
	}
	seen := map[Digest]struct{}{}
	seenSubjects := map[Digest]struct{}{}
	seenAttempts := map[Digest]struct{}{}
	seenOrdinals := map[int]struct{}{}
	seenCandidates := map[CandidateExecutionKey]struct{}{}
	for _, instance := range instances {
		if !instance.digest.Valid() || instance.envelopeDigest != envelope.Digest() || !instance.subjectDigest.Valid() {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "instance is unbound or belongs to another policy")
		}
		if instance.planDigest != binding.planDigest || instance.stimulusDigest != binding.stimulusDigest ||
			instance.purpose != binding.purpose || instance.requiredFreshTrials != binding.requiredFreshTrials {
			return nil, refuse("COMPARISON_MATRIX_BINDING_MISMATCH", "plan, stimulus, purpose, or repeat requirement differs")
		}
		if _, duplicate := seen[instance.digest]; duplicate {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "duplicate measurement row")
		}
		seen[instance.digest] = struct{}{}
		if _, duplicate := seenSubjects[instance.subjectDigest]; duplicate {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "one runtime subject has multiple measurement rows")
		}
		seenSubjects[instance.subjectDigest] = struct{}{}
		if !instance.attemptArtifactDigest.Valid() {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "measurement row has invalid attempt artifact")
		}
		if _, duplicate := seenAttempts[instance.attemptArtifactDigest]; duplicate {
			return nil, refuse("REUSED_COMPARISON_ATTEMPT_EVIDENCE", instance.attemptArtifactDigest.String())
		}
		seenAttempts[instance.attemptArtifactDigest] = struct{}{}
		if instance.scheduleOrdinal < 0 {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "measurement row has invalid schedule ordinal")
		}
		if _, duplicate := seenOrdinals[instance.scheduleOrdinal]; duplicate {
			return nil, refuse("REUSED_COMPARISON_SCHEDULE_ORDINAL", "comparison subjects must occupy distinct schedule positions")
		}
		seenOrdinals[instance.scheduleOrdinal] = struct{}{}
		if !instance.candidateKey.Valid() {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "measurement row has invalid candidate")
		}
		if _, duplicate := seenCandidates[instance.candidateKey]; duplicate {
			return nil, refuse("INVALID_COMPARISON_MATRIX", "candidate has multiple measurement rows")
		}
		seenCandidates[instance.candidateKey] = struct{}{}
	}

	config := envelope.Config()
	reasonSet := map[string]struct{}{}
	type basisValue struct {
		Name          string `json:"name"`
		CanonicalJSON string `json:"canonical_json"`
	}
	basis := make([]basisValue, 0, len(config.RequiredEqual)+len(config.Rejected))
	checkEqual := func(name, reason string) {
		first, exists := instances[0].values[name]
		if !exists {
			reasonSet["MISSING_MEASURED_DIMENSION:"+name] = struct{}{}
			return
		}
		basis = append(basis, basisValue{Name: name, CanonicalJSON: string(first.canonical)})
		for _, instance := range instances[1:] {
			value, present := instance.values[name]
			if !present || !bytes.Equal(first.canonical, value.canonical) {
				reasonSet[reason] = struct{}{}
			}
		}
	}
	for _, dimension := range config.RequiredEqual {
		checkEqual(dimension.Name, "REQUIRED_EQUAL_VARIANCE:"+dimension.Name)
	}
	for _, dimension := range config.Rejected {
		checkEqual(dimension.Name, dimension.ReasonCode)
	}
	sort.Slice(basis, func(i, j int) bool { return basis[i].Name < basis[j].Name })
	if len(reasonSet) > 0 {
		reasons := make([]string, 0, len(reasonSet))
		for reason := range reasonSet {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)
		measurementDigests := make([]string, len(instances))
		for index, instance := range instances {
			measurementDigests[index] = instance.digest.String()
		}
		sort.Strings(measurementDigests)
		identity := struct {
			SchemaVersion       string   `json:"schema_version"`
			Kind                string   `json:"kind"`
			PlanDigest          string   `json:"world_plan_digest"`
			StimulusDigest      string   `json:"stimulus_digest"`
			EnvelopeDigest      string   `json:"comparison_envelope_digest"`
			Purpose             string   `json:"attempt_purpose"`
			RequiredFreshTrials int      `json:"required_fresh_trials"`
			MeasurementDigests  []string `json:"measurement_digests"`
			ReasonCodes         []string `json:"reason_codes"`
		}{
			SchemaVersion: SchemaVersion, Kind: "RejectedComparison",
			PlanDigest: binding.planDigest.String(), StimulusDigest: binding.stimulusDigest.String(),
			EnvelopeDigest: envelope.Digest().String(), Purpose: string(binding.purpose),
			RequiredFreshTrials: binding.requiredFreshTrials,
			MeasurementDigests:  measurementDigests, ReasonCodes: reasons,
		}
		digest, canonicalBytes, err := digestTyped("RejectedComparison", identity)
		if err != nil {
			return nil, err
		}
		return RejectedComparison{
			digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...),
			envelopeDigest: envelope.Digest(), reasonCodes: reasons,
		}, nil
	}

	candidateRoster := make([]CandidateExecutionKey, 0, len(seenCandidates))
	for candidate := range seenCandidates {
		candidateRoster = append(candidateRoster, candidate)
	}
	sort.Slice(candidateRoster, func(i, j int) bool {
		return candidateRoster[i].String() < candidateRoster[j].String()
	})
	basisIdentity := struct {
		SchemaVersion   string       `json:"schema_version"`
		Kind            string       `json:"kind"`
		PlanDigest      string       `json:"world_plan_digest"`
		EnvelopeDigest  string       `json:"comparison_envelope_digest"`
		CandidateRoster []string     `json:"candidate_roster"`
		EqualBasis      []basisValue `json:"required_and_rejected_equal_basis"`
		Nonclaim        string       `json:"nonclaim"`
	}{
		SchemaVersion:   SchemaVersion,
		Kind:            "ComparisonBasis",
		PlanDigest:      binding.planDigest.String(),
		EnvelopeDigest:  envelope.Digest().String(),
		CandidateRoster: make([]string, len(candidateRoster)),
		EqualBasis:      basis,
		Nonclaim:        "STIMULUS_AND_ATTEMPT_PURPOSE_INDEPENDENT_COMPARISON_BASIS",
	}
	for index, candidate := range candidateRoster {
		basisIdentity.CandidateRoster[index] = candidate.String()
	}
	basisDigest, _, basisErr := digestTyped("ComparisonBasis", basisIdentity)
	if basisErr != nil {
		return nil, basisErr
	}
	measurementDigests := make([]string, len(instances))
	measurementDigestValues := make([]Digest, len(instances))
	for index, instance := range instances {
		measurementDigests[index] = instance.digest.String()
		measurementDigestValues[index] = instance.digest
	}
	sort.Strings(measurementDigests)
	sort.Slice(measurementDigestValues, func(i, j int) bool {
		return measurementDigestValues[i].String() < measurementDigestValues[j].String()
	})
	identity := struct {
		SchemaVersion         string       `json:"schema_version"`
		Kind                  string       `json:"kind"`
		PlanDigest            string       `json:"world_plan_digest"`
		StimulusDigest        string       `json:"stimulus_digest"`
		EnvelopeDigest        string       `json:"comparison_envelope_digest"`
		Purpose               string       `json:"attempt_purpose"`
		RequiredFreshTrials   int          `json:"required_fresh_trials"`
		ComparisonBasisDigest string       `json:"comparison_basis_digest"`
		CandidateRoster       []string     `json:"candidate_roster"`
		MeasurementDigests    []string     `json:"measurement_digests"`
		EqualBasis            []basisValue `json:"required_and_rejected_equal_basis"`
		Nonclaim              string       `json:"nonclaim"`
	}{
		SchemaVersion: SchemaVersion, Kind: "ComparisonAdmission",
		PlanDigest: binding.planDigest.String(), StimulusDigest: binding.stimulusDigest.String(),
		EnvelopeDigest: envelope.Digest().String(), Purpose: string(binding.purpose),
		RequiredFreshTrials:   binding.requiredFreshTrials,
		ComparisonBasisDigest: basisDigest.String(),
		CandidateRoster:       make([]string, len(candidateRoster)),
		// MUTANT_U1_DOMAIN_DROP_MEASUREMENT_SET_FROM_ADMISSION: concrete matrix identity is authority-bearing.
		MeasurementDigests: measurementDigests,
		EqualBasis:         basis,
		Nonclaim:           "BEHAVIORAL_COMPATIBILITY_NOT_ESTABLISHED",
	}
	for index, candidate := range candidateRoster {
		identity.CandidateRoster[index] = candidate.String()
	}
	admissionDigest, canonicalBytes, err := digestTyped("ComparisonAdmission", identity)
	if err != nil {
		return nil, err
	}
	admitted := AdmittedComparison{
		digest: admissionDigest, canonicalBytes: append([]byte(nil), canonicalBytes...),
		comparisonBasisDigest: basisDigest,
		envelopeDigest:        envelope.Digest(), planDigest: binding.planDigest, stimulusDigest: binding.stimulusDigest,
		purpose: binding.purpose, requiredFreshTrials: binding.requiredFreshTrials,
		candidateRoster:    append([]CandidateExecutionKey(nil), candidateRoster...),
		measurementDigests: append([]Digest(nil), measurementDigestValues...),
		tokens:             map[Digest]AdmissionToken{},
	}
	for _, instance := range instances {
		tokenIdentity := struct {
			SchemaVersion             string `json:"schema_version"`
			Kind                      string `json:"kind"`
			ComparisonAdmissionDigest string `json:"comparison_admission_digest"`
			MeasurementDigest         string `json:"measurement_digest"`
			SubjectDigest             string `json:"measurement_subject_digest"`
		}{SchemaVersion, "AdmissionToken", admissionDigest.String(), instance.digest.String(), instance.subjectDigest.String()}
		tokenDigest, _, tokenErr := digestTyped("AdmissionToken", tokenIdentity)
		if tokenErr != nil {
			return nil, tokenErr
		}
		admitted.tokens[instance.digest] = AdmissionToken{
			digest: tokenDigest, comparisonAdmissionDigest: admissionDigest, envelopeDigest: envelope.Digest(),
			comparisonBasisDigest: basisDigest,
			measurementDigest:     instance.digest, subjectDigest: instance.subjectDigest,
			planDigest: binding.planDigest, stimulusDigest: binding.stimulusDigest,
			purpose: binding.purpose, requiredFreshTrials: binding.requiredFreshTrials,
			candidateRoster: append([]CandidateExecutionKey(nil), candidateRoster...),
		}
	}
	return admitted, nil
}

func validAdmissionRoster(roster []CandidateExecutionKey) bool {
	if len(roster) < 2 || len(roster) > 4 {
		return false
	}
	for index, candidate := range roster {
		if !candidate.Valid() || (index > 0 && candidate.String() <= roster[index-1].String()) {
			return false
		}
	}
	return true
}

// WorldInstance is immutable structural allocation identity. It is constructed
// before measurement; an admission token later binds back to its digest.
type WorldInstanceConfig struct {
	StimulusDigest        Digest
	AttemptArtifactDigest Digest
	Purpose               AttemptPurpose
	InstanceNonce         string
	ScheduleOrdinal       int
}

type WorldInstance struct {
	digest                     Digest
	canonicalBytes             []byte
	planDigest                 Digest
	candidateKey               CandidateExecutionKey
	stimulusDigest             Digest
	envelopeDigest             Digest
	capturePolicyDigest        Digest
	projectionDefinitionDigest Digest
	attemptArtifactDigest      Digest
	purpose                    AttemptPurpose
	instanceNonce              string
	scheduleOrdinal            int
	requiredFreshTrials        int
}

func NewWorldInstance(plan WorldPlan, binding CandidateExecutionBinding, config WorldInstanceConfig) (WorldInstance, error) {
	if !plan.digest.Valid() || len(plan.canonicalBytes) == 0 || !binding.Valid() ||
		!config.StimulusDigest.Valid() || !config.AttemptArtifactDigest.Valid() ||
		config.InstanceNonce == "" || config.ScheduleOrdinal < 0 {
		return WorldInstance{}, refuse("INVALID_WORLD_INSTANCE", "measured identity is incomplete")
	}
	// MUTANT_U1_DOMAIN_BYPASS_WORLD_PLAN_BINDING: retained candidate authority must match every executable plan boundary.
	if binding.identity.WorldPlanDigest != plan.digest ||
		binding.identity.MaterializationPolicyDigest != plan.materializationPolicyDigest ||
		binding.identity.AdapterDigest != plan.adapterDigest ||
		binding.identity.RunnerDigest != plan.adapter.RunnerDigest ||
		binding.identity.ProjectionDefinitionDigest != plan.projectionDefinition.Digest() {
		return WorldInstance{}, refuse("WORLD_INSTANCE_PLAN_BINDING_MISMATCH", "candidate binding disagrees with the bound world plan")
	}
	var requiredFreshTrials int
	switch config.Purpose {
	case AttemptDiscovery, AttemptReduction:
		requiredFreshTrials = plan.repeatSchedule.DiscoveryRepeats
	case AttemptFinalSweep, AttemptConfirmation, AttemptConformance:
		requiredFreshTrials = plan.repeatSchedule.ConfirmationRepeats
	default:
		return WorldInstance{}, refuse("INVALID_WORLD_INSTANCE", "unknown attempt purpose")
	}
	identity := struct {
		SchemaVersion              string `json:"schema_version"`
		Kind                       string `json:"kind"`
		PlanDigest                 string `json:"world_plan_digest"`
		CandidateExecutionKey      string `json:"candidate_execution_key"`
		StimulusDigest             string `json:"stimulus_digest"`
		EnvelopeDigest             string `json:"comparison_envelope_digest"`
		CapturePolicyDigest        string `json:"capture_policy_digest"`
		ProjectionDefinitionDigest string `json:"projection_definition_digest"`
		AttemptArtifactDigest      string `json:"attempt_artifact_digest"`
		Purpose                    string `json:"attempt_purpose"`
		InstanceNonce              string `json:"instance_nonce"`
		ScheduleOrdinal            int    `json:"schedule_ordinal"`
		RequiredFreshTrials        int    `json:"required_fresh_trials"`
	}{
		SchemaVersion:              SchemaVersion,
		Kind:                       "WorldInstance",
		PlanDigest:                 plan.digest.String(),
		CandidateExecutionKey:      binding.key.String(),
		StimulusDigest:             config.StimulusDigest.String(),
		EnvelopeDigest:             plan.comparisonEnvelopeDigest.String(),
		CapturePolicyDigest:        plan.capturePolicyDigest.String(),
		ProjectionDefinitionDigest: plan.projectionDefinition.Digest().String(),
		AttemptArtifactDigest:      config.AttemptArtifactDigest.String(),
		Purpose:                    string(config.Purpose),
		InstanceNonce:              config.InstanceNonce,
		ScheduleOrdinal:            config.ScheduleOrdinal,
		RequiredFreshTrials:        requiredFreshTrials,
	}
	digest, canonicalBytes, err := digestTyped("WorldInstance", identity)
	if err != nil {
		return WorldInstance{}, err
	}
	return WorldInstance{
		digest:                     digest,
		canonicalBytes:             append([]byte(nil), canonicalBytes...),
		planDigest:                 plan.digest,
		candidateKey:               binding.key,
		stimulusDigest:             config.StimulusDigest,
		envelopeDigest:             plan.comparisonEnvelopeDigest,
		capturePolicyDigest:        plan.capturePolicyDigest,
		projectionDefinitionDigest: plan.projectionDefinition.Digest(),
		attemptArtifactDigest:      config.AttemptArtifactDigest,
		purpose:                    config.Purpose,
		instanceNonce:              config.InstanceNonce,
		scheduleOrdinal:            config.ScheduleOrdinal,
		requiredFreshTrials:        requiredFreshTrials,
	}, nil
}

func (i WorldInstance) Digest() Digest { return i.digest }

func (i WorldInstance) CanonicalBytes() []byte { return append([]byte(nil), i.canonicalBytes...) }

func (i WorldInstance) PlanDigest() Digest { return i.planDigest }

func (i WorldInstance) CandidateKey() CandidateExecutionKey { return i.candidateKey }

func (i WorldInstance) StimulusDigest() Digest { return i.stimulusDigest }

func (i WorldInstance) EnvelopeDigest() Digest { return i.envelopeDigest }

func (i WorldInstance) CapturePolicyDigest() Digest { return i.capturePolicyDigest }

func (i WorldInstance) ProjectionDefinitionDigest() Digest { return i.projectionDefinitionDigest }

func (i WorldInstance) AttemptArtifactDigest() Digest { return i.attemptArtifactDigest }

func (i WorldInstance) Purpose() AttemptPurpose { return i.purpose }

func (i WorldInstance) InstanceNonce() string { return i.instanceNonce }

func (i WorldInstance) ScheduleOrdinal() int { return i.scheduleOrdinal }

func (i WorldInstance) RequiredFreshTrials() int { return i.requiredFreshTrials }
