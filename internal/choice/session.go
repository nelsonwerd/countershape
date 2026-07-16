package choice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	CodeInvalidSessionState        RefusalCode = "INVALID_SESSION_STATE"
	CodeUnknownReviewSurface       RefusalCode = "UNKNOWN_REVIEW_SURFACE"
	CodeRequiredSurfaceNotVisited  RefusalCode = "REQUIRED_SURFACE_NOT_VISITED"
	CodeChangeRationaleRequired    RefusalCode = "CHANGE_RATIONALE_REQUIRED"
	CodeUnnecessaryChangeRationale RefusalCode = "UNNECESSARY_CHANGE_RATIONALE"
	CodeInvalidDecisionAttribution RefusalCode = "INVALID_DECISION_ATTRIBUTION"
)

type SessionState string

const (
	SessionBlindOpen           SessionState = "BLIND_OPEN"
	SessionProvisionalRecorded SessionState = "PROVISIONAL_RECORDED"
	SessionRevealed            SessionState = "REVEALED"
	SessionPostRevealRecorded  SessionState = "POST_REVEAL_RECORDED"
	SessionFinalized           SessionState = "FINALIZED"
)

type ReviewSurface string

const (
	SurfaceOriginalWitness      ReviewSurface = "ORIGINAL_WITNESS"
	SurfaceMinimizedWitness     ReviewSurface = "MINIMIZED_WITNESS"
	SurfaceReductionDerivation  ReviewSurface = "REDUCTION_DERIVATION"
	SurfaceProjectionOperations ReviewSurface = "PROJECTION_OPERATIONS"
	SurfaceNonassertedFields    ReviewSurface = "NONASSERTED_FIELDS"
	SurfaceProvenance           ReviewSurface = "PROVENANCE_REVEAL"
)

var requiredReviewSurfaces = []ReviewSurface{
	SurfaceOriginalWitness,
	SurfaceMinimizedWitness,
	SurfaceReductionDerivation,
	SurfaceProjectionOperations,
	SurfaceNonassertedFields,
	SurfaceProvenance,
}

type RulingDraftInput struct {
	Action               Action
	SelectedFields       []string
	AllowedAliases       []string
	CustomExpectation    *CompleteTuple
	CustomReviewer       string
	CustomReviewEvidence domain.Digest
}

type exactValueWire struct {
	Tag                 string `json:"tag"`
	Text                string `json:"text"`
	Boolean             bool   `json:"boolean"`
	CanonicalJSONBase64 string `json:"canonical_json_base64"`
}

type fieldValueWire struct {
	FieldID string         `json:"field_id"`
	Value   exactValueWire `json:"value"`
}

type tupleWire struct {
	Fields []fieldValueWire `json:"fields"`
}

type rulingDraftIdentity struct {
	SchemaVersion              string     `json:"schema_version"`
	Kind                       string     `json:"kind"`
	ChoicepointDigest          string     `json:"choicepoint_digest"`
	Action                     string     `json:"action"`
	SelectedFields             []string   `json:"selected_fields"`
	AllowedAliases             []string   `json:"allowed_aliases"`
	CustomExpectation          *tupleWire `json:"custom_expectation"`
	CustomReviewer             string     `json:"custom_reviewer"`
	CustomReviewEvidenceDigest string     `json:"custom_review_evidence_digest"`
}

type rulingDraft struct {
	digest         domain.Digest
	canonicalBytes []byte
	input          RulingDraftInput
	validated      ValidatedRuling
}

type Session struct {
	record          ChoicepointRecord
	view            BlindView
	state           SessionState
	visits          map[ReviewSurface]struct{}
	preRevealVisits map[ReviewSurface]struct{}
	provisional     *rulingDraft
	final           *rulingDraft
	revealed        bool
	rationale       string
}

func NewSession(record ChoicepointRecord) (Session, error) {
	view, err := NewBlindView(record)
	if err != nil {
		return Session{}, err
	}
	return Session{
		record: record, view: view, state: SessionBlindOpen,
		visits: map[ReviewSurface]struct{}{}, preRevealVisits: map[ReviewSurface]struct{}{},
	}, nil
}

func (s Session) State() SessionState { return s.state }
func (s Session) BlindDTO() BlindDTO  { return s.view.DTO() }

func (s Session) Visit(surface ReviewSurface) (Session, error) {
	if !validReviewSurface(surface) {
		return Session{}, refusal(CodeUnknownReviewSurface, "review surface is unknown")
	}
	if s.state == "" || !s.record.Valid() || s.visits == nil || s.preRevealVisits == nil {
		return Session{}, refusal(CodeInvalidSessionState, "decision session is invalid")
	}
	if s.state == SessionFinalized {
		return Session{}, refusal(CodeInvalidSessionState, "a finalized session is immutable")
	}
	if surface == SurfaceProvenance && !s.revealed {
		return Session{}, refusal(CodeInvalidSessionState, "provenance cannot be visited before reveal")
	}
	result := s.clone()
	result.visits[surface] = struct{}{}
	if !s.revealed {
		result.preRevealVisits[surface] = struct{}{}
	}
	return result, nil
}

func (s Session) Propose(input RulingDraftInput) (Session, error) {
	if s.state != SessionBlindOpen || s.revealed {
		return Session{}, refusal(CodeInvalidSessionState, "a blind proposal is legal only before reveal")
	}
	for _, surface := range requiredReviewSurfaces {
		if surface == SurfaceProvenance {
			continue
		}
		if _, visited := s.preRevealVisits[surface]; !visited {
			return Session{}, refusal(CodeRequiredSurfaceNotVisited, "blind proposal requires pre-reveal "+string(surface))
		}
	}
	draft, err := newRulingDraft(s.record, s.view, input)
	if err != nil {
		return Session{}, err
	}
	result := s.clone()
	result.provisional = &draft
	result.final = &draft
	result.state = SessionProvisionalRecorded
	return result, nil
}

func (s Session) Reveal() (Session, RevealDTO, error) {
	if s.state != SessionBlindOpen && s.state != SessionProvisionalRecorded {
		return Session{}, RevealDTO{}, refusal(CodeInvalidSessionState, "reveal is legal once from a blind state")
	}
	reveal, err := buildRevealDTO(s.record, s.view)
	if err != nil {
		return Session{}, RevealDTO{}, err
	}
	result := s.clone()
	result.revealed = true
	result.state = SessionRevealed
	return result, reveal, nil
}

// Revise records the required post-reveal affirmation. A changed full draft
// requires a rationale; an unchanged draft refuses an unnecessary rationale.
func (s Session) Revise(input RulingDraftInput, rationale string) (Session, error) {
	if s.state != SessionRevealed || !s.revealed {
		return Session{}, refusal(CodeInvalidSessionState, "post-reveal ruling is legal only immediately after reveal")
	}
	draft, err := newRulingDraft(s.record, s.view, input)
	if err != nil {
		return Session{}, err
	}
	changed := s.provisional != nil && s.provisional.digest != draft.digest
	if changed && (rationale == "" || strings.TrimSpace(rationale) == "" || len(rationale) > 8*1024 ||
		!utf8.ValidString(rationale) || containsControl(rationale)) {
		return Session{}, refusal(CodeChangeRationaleRequired, "changed post-reveal ruling requires a bounded explicit rationale")
	}
	if !changed && rationale != "" {
		return Session{}, refusal(CodeUnnecessaryChangeRationale, "unchanged or first post-reveal ruling cannot carry a change rationale")
	}
	result := s.clone()
	result.final = &draft
	result.rationale = rationale
	result.state = SessionPostRevealRecorded
	return result, nil
}

func (s Session) Finalize(
	actor string,
	annotation string,
	receipts []domain.ReceiptReference,
) (Session, DecisionRecord, error) {
	if s.state != SessionPostRevealRecorded || !s.revealed || s.final == nil {
		return Session{}, DecisionRecord{}, refusal(CodeInvalidSessionState, "finalization requires a post-reveal validated ruling")
	}
	for _, surface := range requiredReviewSurfaces {
		if _, visited := s.visits[surface]; !visited {
			return Session{}, DecisionRecord{}, refusal(CodeRequiredSurfaceNotVisited, string(surface))
		}
	}
	decision, err := newDecisionRecord(s, actor, annotation, receipts, nil)
	if err != nil {
		return Session{}, DecisionRecord{}, err
	}
	result := s.clone()
	result.state = SessionFinalized
	return result, decision, nil
}

func newRulingDraft(record ChoicepointRecord, view BlindView, input RulingDraftInput) (rulingDraft, error) {
	if input.SelectedFields == nil || input.AllowedAliases == nil {
		return rulingDraft{}, refusal(CodeOmittedSelectedFields, "ruling draft selections must be explicit arrays")
	}
	selectedFields := make([]string, len(input.SelectedFields))
	copy(selectedFields, input.SelectedFields)
	sort.Strings(selectedFields)
	for index := 1; index < len(selectedFields); index++ {
		if selectedFields[index] == selectedFields[index-1] {
			return rulingDraft{}, refusal(CodeDuplicateSelectedField, "ruling draft field is duplicated")
		}
	}
	aliases := make([]string, len(input.AllowedAliases))
	copy(aliases, input.AllowedAliases)
	sort.Strings(aliases)
	allowed, err := view.resolveAliases(aliases)
	if err != nil {
		return rulingDraft{}, err
	}
	normalized := RulingDraftInput{
		Action: input.Action, SelectedFields: selectedFields, AllowedAliases: aliases,
		CustomExpectation: cloneTuplePointer(input.CustomExpectation), CustomReviewer: input.CustomReviewer,
		CustomReviewEvidence: input.CustomReviewEvidence,
	}
	rulingInput := RulingInput{
		Action: input.Action, SelectedFields: selectedFields, AllowedObserved: allowed,
		CustomExpectation: cloneTuplePointer(input.CustomExpectation),
	}
	if input.Action == ActionCustomExpectation {
		if input.CustomExpectation == nil {
			return rulingDraft{}, refusal(CodeCustomExpectationCardinality, "custom ruling draft lacks an expectation")
		}
		review, reviewErr := NewCustomExpectationReview(
			record.confirmed, selectedFields, *input.CustomExpectation, input.CustomReviewer, input.CustomReviewEvidence,
		)
		if reviewErr != nil {
			return rulingDraft{}, reviewErr
		}
		rulingInput.CustomReview = review
	} else if input.CustomReviewer != "" || input.CustomReviewEvidence.Valid() {
		return rulingDraft{}, refusal(CodeUnexpectedCustomExpectation, "non-custom ruling draft carries custom review authority")
	}
	validated, err := ValidateRuling(record.confirmed, rulingInput)
	if err != nil {
		return rulingDraft{}, err
	}
	identity, err := rulingDraftIdentityFor(record.digest, normalized)
	if err != nil {
		return rulingDraft{}, err
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("RulingDraft", identity)
	if err != nil {
		return rulingDraft{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return rulingDraft{}, err
	}
	return rulingDraft{digest: digest, canonicalBytes: canonicalBytes, input: normalized, validated: validated}, nil
}

func parseRulingDraft(exact []byte, record ChoicepointRecord, view BlindView) (rulingDraft, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return rulingDraft{}, refusal(CodeInvalidSessionState, "ruling draft wire is invalid")
	}
	canonicalBytes, err := value.CanonicalChecked()
	members, object := value.Members()
	if err != nil || !bytes.Equal(canonicalBytes, exact) || !object || len(members) != 9 {
		return rulingDraft{}, refusal(CodeInvalidSessionState, "ruling draft wire is nonexact or has unknown members")
	}
	var identity rulingDraftIdentity
	if err := json.Unmarshal(exact, &identity); err != nil || identity.SchemaVersion != domain.SchemaVersion ||
		identity.Kind != "RulingDraft" || identity.ChoicepointDigest != record.digest.String() {
		return rulingDraft{}, refusal(CodeInvalidSessionState, "ruling draft closed wire facts disagree")
	}
	input := RulingDraftInput{
		Action: Action(identity.Action), SelectedFields: identity.SelectedFields,
		AllowedAliases: identity.AllowedAliases, CustomReviewer: identity.CustomReviewer,
	}
	if identity.CustomReviewEvidenceDigest != "" {
		input.CustomReviewEvidence, err = domain.ParseDigest(identity.CustomReviewEvidenceDigest)
		if err != nil {
			return rulingDraft{}, err
		}
	}
	if identity.CustomExpectation != nil {
		tuple, tupleErr := tupleFromWire(*identity.CustomExpectation)
		if tupleErr != nil {
			return rulingDraft{}, tupleErr
		}
		input.CustomExpectation = &tuple
	}
	draft, err := newRulingDraft(record, view, input)
	if err != nil || !bytes.Equal(draft.canonicalBytes, exact) {
		return rulingDraft{}, refusal(CodeInvalidSessionState, "ruling draft did not reconstruct exactly")
	}
	return draft, nil
}

func rulingDraftIdentityFor(choicepoint domain.Digest, input RulingDraftInput) (rulingDraftIdentity, error) {
	identity := rulingDraftIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "RulingDraft", ChoicepointDigest: choicepoint.String(),
		Action: string(input.Action), SelectedFields: cloneExplicitStrings(input.SelectedFields),
		AllowedAliases: cloneExplicitStrings(input.AllowedAliases), CustomReviewer: input.CustomReviewer,
	}
	if input.CustomReviewEvidence.Valid() {
		identity.CustomReviewEvidenceDigest = input.CustomReviewEvidence.String()
	}
	if input.CustomExpectation != nil {
		wire, err := tupleToWire(*input.CustomExpectation)
		if err != nil {
			return rulingDraftIdentity{}, err
		}
		identity.CustomExpectation = &wire
	}
	return identity, nil
}

func tupleToWire(tuple CompleteTuple) (tupleWire, error) {
	fields := append([]FieldValue(nil), tuple.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].FieldID < fields[j].FieldID })
	result := tupleWire{Fields: make([]fieldValueWire, len(fields))}
	for index, field := range fields {
		if index > 0 && field.FieldID == fields[index-1].FieldID {
			return tupleWire{}, refusal(CodeDuplicateTupleField, "tuple wire field is duplicated")
		}
		value := field.Value
		wire := exactValueWire{Tag: string(value.Tag()), Text: value.Text(), Boolean: value.Boolean()}
		if value.Tag() == ValueCanonicalJSON {
			wire.Text = ""
			wire.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.CanonicalBytes())
		}
		result.Fields[index] = fieldValueWire{FieldID: field.FieldID, Value: wire}
	}
	return result, nil
}

func tupleFromWire(wire tupleWire) (CompleteTuple, error) {
	if wire.Fields == nil {
		return CompleteTuple{}, refusal(CodeIncompleteTuple, "tuple wire fields are omitted")
	}
	result := CompleteTuple{Fields: make([]FieldValue, len(wire.Fields))}
	for index, field := range wire.Fields {
		if field.FieldID == "" || (index > 0 && field.FieldID <= wire.Fields[index-1].FieldID) {
			return CompleteTuple{}, refusal(CodeDuplicateTupleField, "tuple wire order or field id is invalid")
		}
		value, err := exactValueFromWire(field.Value)
		if err != nil {
			return CompleteTuple{}, err
		}
		result.Fields[index] = FieldValue{FieldID: field.FieldID, Value: value}
	}
	return result, nil
}

func exactValueFromWire(wire exactValueWire) (ExactValue, error) {
	switch ValueTag(wire.Tag) {
	case ValueMissing:
		if wire.Text != "" || wire.Boolean || wire.CanonicalJSONBase64 != "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "missing wire carries a payload")
		}
		return MissingValue(), nil
	case ValueNull:
		if wire.Text != "" || wire.Boolean || wire.CanonicalJSONBase64 != "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "null wire carries a payload")
		}
		return NullValue(), nil
	case ValueString:
		if wire.Boolean || wire.CanonicalJSONBase64 != "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "string wire carries another payload")
		}
		return StringValue(wire.Text)
	case ValueInteger:
		if wire.Boolean || wire.CanonicalJSONBase64 != "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "integer wire carries another payload")
		}
		return IntegerValue(wire.Text)
	case ValueBoolean:
		if wire.Text != "" || wire.CanonicalJSONBase64 != "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "boolean wire carries another payload")
		}
		return BooleanValue(wire.Boolean), nil
	case ValueCanonicalJSON:
		if wire.Text != "" || wire.Boolean || wire.CanonicalJSONBase64 == "" {
			return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON wire payload is invalid")
		}
		body, err := base64.StdEncoding.Strict().DecodeString(wire.CanonicalJSONBase64)
		if err != nil || base64.StdEncoding.EncodeToString(body) != wire.CanonicalJSONBase64 {
			return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON wire base64 is invalid")
		}
		return CanonicalJSONBytes(body)
	default:
		return ExactValue{}, refusal(CodeInvalidFieldType, "tuple wire has an unknown exact-value tag")
	}
}

func cloneTuplePointer(input *CompleteTuple) *CompleteTuple {
	if input == nil {
		return nil
	}
	value := cloneTuple(*input)
	return &value
}

func cloneExplicitStrings(input []string) []string {
	if input == nil {
		return nil
	}
	result := make([]string, len(input))
	copy(result, input)
	return result
}

func (s Session) clone() Session {
	result := s
	result.visits = make(map[ReviewSurface]struct{}, len(s.visits))
	for surface := range s.visits {
		result.visits[surface] = struct{}{}
	}
	result.preRevealVisits = make(map[ReviewSurface]struct{}, len(s.preRevealVisits))
	for surface := range s.preRevealVisits {
		result.preRevealVisits[surface] = struct{}{}
	}
	return result
}

func validReviewSurface(surface ReviewSurface) bool {
	for _, required := range requiredReviewSurfaces {
		if surface == required {
			return true
		}
	}
	return false
}

type RevealedCandidate struct {
	CandidateExecutionKey string `json:"candidate_execution_key"`
	TreeIdentityDigest    string `json:"tree_identity_digest"`
	DisplayRef            string `json:"display_ref"`
	ProducerMetadata      string `json:"producer_metadata"`
}

type RevealedGroup struct {
	Alias      string              `json:"alias"`
	Candidates []RevealedCandidate `json:"candidates"`
}

type RevealedExclusion struct {
	Candidate      RevealedCandidate `json:"candidate"`
	Classification string            `json:"classification"`
}

type revealIdentity struct {
	SchemaVersion     string              `json:"schema_version"`
	Kind              string              `json:"kind"`
	ChoicepointDigest string              `json:"choicepoint_digest"`
	Groups            []RevealedGroup     `json:"groups"`
	Exclusions        []RevealedExclusion `json:"exclusions"`
}

type RevealDTO struct {
	digest    domain.Digest
	canonical []byte
	identity  revealIdentity
}

func buildRevealDTO(record ChoicepointRecord, view BlindView) (RevealDTO, error) {
	bindings := make(map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding, len(record.bindings))
	for _, binding := range record.bindings {
		bindings[binding.Key()] = binding
	}
	reveals := make(map[domain.CandidateExecutionKey]CandidateReveal, len(record.reveals))
	for _, reveal := range record.reveals {
		reveals[reveal.CandidateExecutionKey] = reveal
	}
	groups := make(map[string][]RevealedCandidate, len(view.aliases))
	for _, outcome := range record.confirmed.ordered {
		alias, err := blindAlias(record.digest, outcome.ref.fingerprint)
		if err != nil {
			return RevealDTO{}, err
		}
		candidate, err := revealedCandidate(outcome.ref.candidate, bindings, reveals)
		if err != nil {
			return RevealDTO{}, err
		}
		groups[alias] = append(groups[alias], candidate)
	}
	identity := revealIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ChoicepointReveal", ChoicepointDigest: record.digest.String(),
		Groups: make([]RevealedGroup, 0, len(groups)), Exclusions: []RevealedExclusion{},
	}
	for _, card := range view.dto.identity.Cards {
		candidates := append([]RevealedCandidate(nil), groups[card.Alias]...)
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].CandidateExecutionKey < candidates[j].CandidateExecutionKey })
		identity.Groups = append(identity.Groups, RevealedGroup{Alias: card.Alias, Candidates: candidates})
	}
	for _, exclusion := range record.confirmation.ConfirmedMap().Exclusions() {
		candidate, err := revealedCandidate(exclusion.CandidateKey, bindings, reveals)
		if err != nil {
			return RevealDTO{}, err
		}
		identity.Exclusions = append(identity.Exclusions, RevealedExclusion{Candidate: candidate, Classification: string(exclusion.Classification)})
	}
	sort.Slice(identity.Exclusions, func(i, j int) bool {
		return identity.Exclusions[i].Candidate.CandidateExecutionKey < identity.Exclusions[j].Candidate.CandidateExecutionKey
	})
	digestRaw, canonicalBytes, err := canon.DigestTyped("ChoicepointReveal", identity)
	if err != nil {
		return RevealDTO{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return RevealDTO{}, err
	}
	return RevealDTO{digest: digest, canonical: canonicalBytes, identity: identity}, nil
}

func revealedCandidate(
	key domain.CandidateExecutionKey,
	bindings map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding,
	reveals map[domain.CandidateExecutionKey]CandidateReveal,
) (RevealedCandidate, error) {
	binding, hasBinding := bindings[key]
	reveal, hasReveal := reveals[key]
	if !hasBinding || !hasReveal {
		return RevealedCandidate{}, refusal(CodeInvalidConfirmedOutcomeSet, "reveal lacks candidate provenance")
	}
	return RevealedCandidate{
		CandidateExecutionKey: key.String(), TreeIdentityDigest: binding.Identity().TreeIdentityDigest.String(),
		DisplayRef: reveal.DisplayRef, ProducerMetadata: reveal.ProducerMetadata,
	}, nil
}

func (r RevealDTO) Digest() domain.Digest  { return r.digest }
func (r RevealDTO) CanonicalBytes() []byte { return append([]byte(nil), r.canonical...) }

func (r RevealDTO) Groups() []RevealedGroup {
	result := make([]RevealedGroup, len(r.identity.Groups))
	for index, group := range r.identity.Groups {
		result[index] = RevealedGroup{
			Alias:      group.Alias,
			Candidates: append([]RevealedCandidate(nil), group.Candidates...),
		}
	}
	return result
}

func (r RevealDTO) Exclusions() []RevealedExclusion {
	return append([]RevealedExclusion(nil), r.identity.Exclusions...)
}
