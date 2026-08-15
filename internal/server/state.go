package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/domain"
)

var errInvalidServer = errors.New("studio server is invalid")

type stateDefinition struct {
	StudyState       string
	CandidateState   string
	ChoicepointState string
	Summary          string
	NextAction       string
	Mutable          bool
	FutureAuthority  bool
}

var stateDefinitions = map[PresentationState]stateDefinition{
	StateEmpty:             {"EMPTY", "NOT_OBSERVED", "UNAVAILABLE", "No study is loaded in this presentation fixture.", "Load one exact sealed Choicepoint.", false, true},
	StatePreparing:         {"PREPARING", "NOT_CLASSIFIED", "UNAVAILABLE", "Exact source and runtime prerequisites are being admitted.", "Inspect the declared plan or cancel before execution.", false, true},
	StateActive:            {"ACTIVE", "NOT_CLASSIFIED", "UNAVAILABLE", "Fresh declared trials are in progress; no provisional outcome is selectable.", "Watch the declared budgets or cancel while preserving completed evidence.", false, true},
	StatePartial:           {"PARTIAL", "INCOMPLETE", "UNAVAILABLE", "The declared run stopped with completed evidence and unfinished work.", "Inspect the missing work; a retry must allocate new attempts.", false, true},
	StateError:             {"ERROR", "UNCOMPARABLE", "UNAVAILABLE", "A detected control failure prevented a decision-ready comparison.", "Repair the named control boundary before deriving a new run.", false, true},
	StateCompleted:         {"COMPLETED", "DISPOSITION_RECORDED", "RESOLVED", "Declared budgets ended and every item has an honest disposition.", "Inspect the local evidence export; this does not mean verified software.", false, true},
	StateUnstable:          {"COMPLETED", "UNSTABLE", "UNAVAILABLE", "At least two eligible fingerprints were observed for one candidate.", "Inspect the histogram or derive a new declared trial budget.", false, true},
	StateUncomparable:      {"COMPLETED", "UNCOMPARABLE", "UNAVAILABLE", "A candidate was excluded by a detected materialization, execution, capture, or projection control.", "Repair the comparison boundary and rerun every required candidate.", false, true},
	StateIncomplete:        {"PARTIAL", "INCOMPLETE", "UNAVAILABLE", "The repeat budget ended before enough eligible trials existed.", "Derive a new run with an explicit budget; do not select partial output.", false, true},
	StateDiscovered:        {"ACTIVE", "OBSERVED_SPLIT", "DISCOVERED", "A disagreement was observed, but reduction or fresh confirmation is unfinished.", "Continue the declared study; resolution is disabled.", false, true},
	StateDecisionReady:     {"COMPLETED", "OBSERVED_STABLE(3/3)", "CHOICEPOINT_READY", "A fresh-confirmed exact-witness Choicepoint is ready for a local ruling.", "Inspect every trust surface and record a provisional action.", true, false},
	StatePredicateEditing:  {"COMPLETED", "OBSERVED_STABLE(3/3)", "CHOICEPOINT_READY", "Every differing field must be acknowledged as Assert or Context only.", "Complete the exact tuple preview before revealing provenance.", true, false},
	StateIdentityReveal:    {"COMPLETED", "OBSERVED_STABLE(3/3)", "CHOICEPOINT_READY", "The provisional ruling is recorded and candidate provenance is now revealed.", "Affirm the ruling or explain a post-reveal change.", true, false},
	StateResolved:          {"COMPLETED", "OBSERVED_STABLE(3/3)", "RULING", "A local caller-attributed exact-witness ruling record is finalized.", "Inspect the selected and nonasserted fields; U8 emits no files.", false, false},
	StateRejectAllResolved: {"COMPLETED", "OBSERVED_STABLE(3/3)", "RULING", "Reject all was recorded without a positive executable predicate.", "Return to implementation work; no contract file was emitted.", false, false},
	StateDeferred:          {"COMPLETED", "OBSERVED_STABLE(3/3)", "RULING", "Defer was recorded as a noncompilable ruling.", "Preserve the rationale; deferred-session reopening is unavailable.", false, false},
	StateStale:             {"COMPLETED", "STALE", "STALE", "A semantic ancestor changed; the historical ruling is inspectable but not current.", "Derive a new Choicepoint; historical bytes are immutable.", false, true},
	StateInvalidated:       {"COMPLETED", "INVALIDATED", "INVALIDATED", "An explicit successor disposition invalidated this presentation fixture.", "Inspect the replacement reference; this record cannot be edited into freshness.", false, true},
}

type studioState struct {
	mu sync.Mutex

	record       choice.ChoicepointRecord
	session      choice.Session
	blind        choice.BlindDTO
	reveal       json.RawMessage
	decision     choice.DecisionRecord
	presentation PresentationState
	definition   stateDefinition
	revisionKey  []byte
	revision     uint64
	acknowledged []FieldAcknowledgement
	visited      []string
	lastDraft    DraftInput
}

type SessionResponse struct {
	SchemaVersion  string `json:"schema_version"`
	StudyID        string `json:"study_id"`
	CSRF           string `json:"csrf"`
	RevisionDigest string `json:"revision_digest"`
	Presentation   string `json:"presentation_state"`
}

type BenchResponse struct {
	SchemaVersion    string                 `json:"schema_version"`
	Kind             string                 `json:"kind"`
	StudyID          string                 `json:"study_id"`
	Presentation     string                 `json:"presentation_state"`
	StudyState       string                 `json:"study_state"`
	CandidateState   string                 `json:"candidate_state"`
	ChoicepointState string                 `json:"choicepoint_state"`
	Summary          string                 `json:"summary"`
	NextAction       string                 `json:"next_action"`
	Mutable          bool                   `json:"mutable"`
	FutureAuthority  bool                   `json:"future_authority_unimplemented"`
	TrustWarning     string                 `json:"trust_warning"`
	NetworkMode      string                 `json:"network_mode"`
	RevisionDigest   string                 `json:"revision_digest"`
	SessionState     string                 `json:"session_state"`
	Blind            json.RawMessage        `json:"blind,omitempty"`
	Reveal           json.RawMessage        `json:"reveal,omitempty"`
	Acknowledgements []FieldAcknowledgement `json:"field_acknowledgements"`
	ReviewedSurfaces []string               `json:"reviewed_surfaces"`
	Draft            *DraftInput            `json:"draft,omitempty"`
	Result           *DecisionResult        `json:"result,omitempty"`
	Nonclaims        []string               `json:"nonclaims"`
}

type FieldAcknowledgement struct {
	FieldID     string `json:"field_id"`
	Disposition string `json:"disposition"`
}

type CustomValueInput struct {
	FieldID        string          `json:"field_id"`
	Tag            string          `json:"tag"`
	Text           string          `json:"text"`
	Boolean        bool            `json:"boolean"`
	BytesBase64    string          `json:"bytes_base64"`
	OrderedStrings []string        `json:"ordered_strings"`
	CanonicalJSON  json.RawMessage `json:"canonical_json"`
}

type DraftInput struct {
	Action                string                 `json:"action"`
	AllowedAliases        []string               `json:"allowed_aliases"`
	FieldAcknowledgements []FieldAcknowledgement `json:"field_acknowledgements"`
	CustomValues          []CustomValueInput     `json:"custom_values"`
	CustomReviewer        string                 `json:"custom_reviewer"`
}

type MutationEnvelope struct {
	ExpectedRevision string      `json:"expected_revision"`
	Surface          string      `json:"surface,omitempty"`
	Draft            *DraftInput `json:"draft,omitempty"`
	Rationale        string      `json:"rationale,omitempty"`
	Actor            string      `json:"actor,omitempty"`
	Annotation       string      `json:"annotation,omitempty"`
}

type DecisionResult struct {
	DecisionDigest     string   `json:"decision_digest"`
	Action             string   `json:"action"`
	SelectedFields     []string `json:"selected_fields"`
	NonassertedFields  []string `json:"nonasserted_fields"`
	EarlyReveal        bool     `json:"early_reveal"`
	ChangedAfterReveal bool     `json:"changed_after_reveal"`
	ChangeRationale    string   `json:"change_rationale"`
	CompilationStatus  string   `json:"compilation_status"`
	EmittedFiles       []string `json:"emitted_files"`
	Authority          string   `json:"authority"`
	AuthorityCeiling   string   `json:"authority_ceiling"`
}

type APIError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Code + ": " + e.Detail
}

func newStudioState(state PresentationState, random io.Reader) (*studioState, error) {
	definition, present := stateDefinitions[state]
	if !present {
		return nil, &APIError{Code: "STUDIO_STATE_REFUSED", Detail: "the requested presentation state is outside the closed fixture roster"}
	}
	record, err := loadSeedChoicepoint()
	if err != nil {
		return nil, err
	}
	session, err := choice.NewSession(record)
	if err != nil {
		return nil, fmt.Errorf("STUDIO_DECISION_SESSION_REFUSED")
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(random, key); err != nil {
		return nil, fmt.Errorf("STUDIO_REVISION_AUTHORITY_REFUSED")
	}
	result := &studioState{
		record: record, session: session, blind: session.BlindDTO(), presentation: state,
		definition: definition, revisionKey: key,
	}
	if err := result.seedPresentation(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *studioState) seedPresentation() error {
	switch s.presentation {
	case StatePredicateEditing:
		return s.visitPreRevealSurfaces()
	case StateIdentityReveal:
		if err := s.visitPreRevealSurfaces(); err != nil {
			return err
		}
		if err := s.applySeedProposal(choice.ActionAllowObserved); err != nil {
			return err
		}
		return s.applyReveal()
	case StateResolved, StateRejectAllResolved, StateDeferred:
		if err := s.visitPreRevealSurfaces(); err != nil {
			return err
		}
		action := choice.ActionAllowObserved
		if s.presentation == StateRejectAllResolved {
			action = choice.ActionRejectAll
		}
		if s.presentation == StateDeferred {
			action = choice.ActionDefer
		}
		if err := s.applySeedProposal(action); err != nil {
			return err
		}
		if err := s.applyReveal(); err != nil {
			return err
		}
		var err error
		s.session, err = s.session.Visit(choice.SurfaceProvenance)
		if err != nil {
			return err
		}
		s.markVisited(choice.SurfaceProvenance)
		draft, err := s.choiceDraft(s.lastDraft)
		if err != nil {
			return err
		}
		s.session, err = s.session.Revise(draft, "")
		if err != nil {
			return err
		}
		_, s.decision, err = s.session.Finalize("u8-seed-operator", "Synthetic presentation fixture; actor authenticity is not established.", []domain.ReceiptReference{})
		return err
	default:
		return nil
	}
}

func (s *studioState) visitPreRevealSurfaces() error {
	var err error
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
	} {
		s.session, err = s.session.Visit(surface)
		if err != nil {
			return err
		}
		s.markVisited(surface)
	}
	return nil
}

func (s *studioState) applySeedProposal(action choice.Action) error {
	differing := s.blind.DifferingFields()
	acks := make([]FieldAcknowledgement, len(differing))
	for index, field := range differing {
		acks[index] = FieldAcknowledgement{FieldID: field, Disposition: "CONTEXT"}
	}
	draft := DraftInput{Action: string(action), AllowedAliases: []string{}, FieldAcknowledgements: acks, CustomValues: []CustomValueInput{}}
	if action == choice.ActionAllowObserved {
		if len(differing) == 0 || len(s.blind.Cards()) == 0 {
			return fmt.Errorf("STUDIO_SEED_RULING_REFUSED")
		}
		for index := range acks {
			acks[index].Disposition = "ASSERT"
		}
		draft.AllowedAliases = []string{s.blind.Cards()[0].Alias}
	}
	choiceInput, err := s.choiceDraft(draft)
	if err != nil {
		return err
	}
	s.session, err = s.session.Propose(choiceInput)
	if err != nil {
		return err
	}
	s.acknowledged = append([]FieldAcknowledgement(nil), acks...)
	s.lastDraft = cloneDraft(draft)
	return nil
}

func (s *studioState) applyReveal() error {
	next, reveal, err := s.session.Reveal()
	if err != nil {
		return err
	}
	s.session = next
	s.reveal = append(json.RawMessage(nil), reveal.CanonicalBytes()...)
	return nil
}

func (s *studioState) revisionDigest() string {
	mac := hmac.New(sha256.New, s.revisionKey)
	_, _ = mac.Write([]byte(s.record.Digest().String()))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(strconv.FormatUint(s.revision, 10)))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(s.session.State()))
	return "revision:" + hex.EncodeToString(mac.Sum(nil))
}

func (s *studioState) sessionResponse(csrf string) SessionResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SessionResponse{SchemaVersion: SchemaVersion, StudyID: StudyID, CSRF: csrf, RevisionDigest: s.revisionDigest(), Presentation: string(s.presentation)}
}

func (s *studioState) benchResponse() BenchResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.benchResponseLocked()
}

func (s *studioState) benchResponseLocked() BenchResponse {
	response := BenchResponse{
		SchemaVersion: SchemaVersion, Kind: "StudioBench", StudyID: StudyID,
		Presentation: string(s.presentation), StudyState: s.definition.StudyState,
		CandidateState: s.definition.CandidateState, ChoicepointState: s.definition.ChoicepointState,
		Summary: s.definition.Summary, NextAction: s.definition.NextAction, Mutable: s.definition.Mutable,
		FutureAuthority: s.definition.FutureAuthority, TrustWarning: TrustedCodeWarning,
		NetworkMode: "HOST_ALLOWED", RevisionDigest: s.revisionDigest(), SessionState: string(s.session.State()),
		Acknowledgements: append([]FieldAcknowledgement(nil), s.acknowledged...),
		ReviewedSurfaces: append([]string(nil), s.visited...),
		Nonclaims: []string{
			"Observed stability is finite evidence, not determinism.",
			"Candidate identity is hidden in the blind payload, not guaranteed anonymous.",
			"Visited evidence surfaces establish presentation, not comprehension.",
			"The studio does not establish correctness, safety, containment, or receipt authority.",
		},
	}
	if s.presentation == StateDecisionReady || s.presentation == StatePredicateEditing || s.presentation == StateIdentityReveal ||
		s.presentation == StateResolved || s.presentation == StateRejectAllResolved || s.presentation == StateDeferred {
		response.Blind = append(json.RawMessage(nil), s.blind.CanonicalBytes()...)
	}
	if len(s.reveal) > 0 {
		response.Reveal = append(json.RawMessage(nil), s.reveal...)
	}
	if s.lastDraft.Action != "" {
		draft := cloneDraft(s.lastDraft)
		response.Draft = &draft
	}
	if s.decision.Valid() {
		response.Result = decisionResult(s.decision)
	}
	return response
}

func decisionResult(decision choice.DecisionRecord) *DecisionResult {
	status := "NONCOMPILABLE_ACTION"
	if _, ok := decision.CompilableRuling(); ok {
		status = "COMPILE_ELIGIBLE_EXACT_PREDICATE"
	}
	return &DecisionResult{
		DecisionDigest: decision.Digest().String(), Action: string(decision.Action()),
		SelectedFields: explicitStrings(decision.SelectedFields()), NonassertedFields: explicitStrings(decision.NonassertedFields()),
		EarlyReveal: decision.EarlyReveal(), ChangedAfterReveal: decision.ChangedAfterReveal(),
		ChangeRationale: decision.PostRevealChangeRationale(), CompilationStatus: status,
		EmittedFiles: []string{}, Authority: "PACKAGE_VALIDATED_LOCAL_CALLER_ATTRIBUTED_EXACT_WITNESS_DECISION_RECORD",
		AuthorityCeiling: "NO_HUMAN_AUTHENTICITY_CORRECTNESS_SAFETY_COMPREHENSION_CONTRACT_EMISSION_OR_RECEIPT_AUTHORITY",
	}
}

func explicitStrings(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func (s *studioState) visit(expected string, surface choice.ReviewSurface) (BenchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireMutableCAS(expected); err != nil {
		return BenchResponse{}, err
	}
	next, err := s.session.Visit(surface)
	if err != nil {
		return BenchResponse{}, apiChoiceError(err)
	}
	s.session = next
	s.markVisited(surface)
	s.revision++
	return s.benchResponseLocked(), nil
}

func (s *studioState) markVisited(surface choice.ReviewSurface) {
	value := string(surface)
	for _, existing := range s.visited {
		if existing == value {
			return
		}
	}
	s.visited = append(s.visited, value)
}

func (s *studioState) propose(expected string, input DraftInput) (BenchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireMutableCAS(expected); err != nil {
		return BenchResponse{}, err
	}
	draft, err := s.choiceDraft(input)
	if err != nil {
		return BenchResponse{}, err
	}
	next, err := s.session.Propose(draft)
	if err != nil {
		return BenchResponse{}, apiChoiceError(err)
	}
	s.session = next
	s.acknowledged = cloneAcknowledgements(input.FieldAcknowledgements)
	s.lastDraft = cloneDraft(input)
	s.presentation = StatePredicateEditing
	s.definition = stateDefinitions[s.presentation]
	s.revision++
	return s.benchResponseLocked(), nil
}

func (s *studioState) revealProvenance(expected string) (BenchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireMutableCAS(expected); err != nil {
		return BenchResponse{}, err
	}
	if err := s.applyReveal(); err != nil {
		return BenchResponse{}, apiChoiceError(err)
	}
	s.presentation = StateIdentityReveal
	s.definition = stateDefinitions[s.presentation]
	s.revision++
	return s.benchResponseLocked(), nil
}

func (s *studioState) revise(expected string, input DraftInput, rationale string) (BenchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireMutableCAS(expected); err != nil {
		return BenchResponse{}, err
	}
	draft, err := s.choiceDraft(input)
	if err != nil {
		return BenchResponse{}, err
	}
	next, err := s.session.Revise(draft, rationale)
	if err != nil {
		return BenchResponse{}, apiChoiceError(err)
	}
	s.session = next
	s.acknowledged = cloneAcknowledgements(input.FieldAcknowledgements)
	s.lastDraft = cloneDraft(input)
	s.revision++
	return s.benchResponseLocked(), nil
}

func (s *studioState) finalize(expected, actor, annotation string) (BenchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireMutableCAS(expected); err != nil {
		return BenchResponse{}, err
	}
	if !boundedVisibleText(actor, 512) || !boundedVisibleTextAllowEmpty(annotation, 32*1024) {
		return BenchResponse{}, &APIError{Code: "STUDIO_DECISION_ATTRIBUTION_REFUSED", Detail: "actor and annotation must be bounded visible UTF-8 text"}
	}
	next, decision, err := s.session.Finalize(actor, annotation, []domain.ReceiptReference{})
	if err != nil {
		return BenchResponse{}, apiChoiceError(err)
	}
	s.session, s.decision = next, decision
	s.presentation = StateResolved
	if decision.Action() == choice.ActionRejectAll {
		s.presentation = StateRejectAllResolved
	} else if decision.Action() == choice.ActionDefer {
		s.presentation = StateDeferred
	}
	s.definition = stateDefinitions[s.presentation]
	s.revision++
	return s.benchResponseLocked(), nil
}

func (s *studioState) requireMutableCAS(expected string) error {
	if !s.definition.Mutable {
		return &APIError{Code: "STUDIO_STATE_IMMUTABLE", Detail: "the selected presentation state has no mutable durable workflow"}
	}
	if expected == "" || !hmac.Equal([]byte(expected), []byte(s.revisionDigest())) {
		return &APIError{Code: "STUDIO_STALE_REVISION", Detail: "the decision session changed; reload the package-issued state before retrying"}
	}
	return nil
}

func (s *studioState) choiceDraft(input DraftInput) (choice.RulingDraftInput, error) {
	if input.AllowedAliases == nil || input.FieldAcknowledgements == nil || input.CustomValues == nil {
		return choice.RulingDraftInput{}, &APIError{Code: "STUDIO_DRAFT_SHAPE_REFUSED", Detail: "draft arrays must be explicit"}
	}
	action := choice.Action(input.Action)
	selected, err := validateAcknowledgements(s.blind.DifferingFields(), input.FieldAcknowledgements, action)
	if err != nil {
		return choice.RulingDraftInput{}, err
	}
	aliases := make([]string, len(input.AllowedAliases))
	copy(aliases, input.AllowedAliases)
	result := choice.RulingDraftInput{Action: action, SelectedFields: selected, AllowedAliases: aliases}
	if action == choice.ActionCustomExpectation {
		fields, err := s.customFields(selected, input.CustomValues)
		if err != nil {
			return choice.RulingDraftInput{}, err
		}
		tuple, err := choice.NewSelectedTuple(s.record, selected, fields)
		if err != nil {
			return choice.RulingDraftInput{}, apiChoiceError(err)
		}
		if !boundedVisibleText(input.CustomReviewer, 512) {
			return choice.RulingDraftInput{}, &APIError{Code: "STUDIO_CUSTOM_REVIEW_REFUSED", Detail: "custom expectation requires one bounded local reviewer label"}
		}
		reviewBytes, _ := json.Marshal(struct {
			Choicepoint string             `json:"choicepoint_digest"`
			Reviewer    string             `json:"reviewer"`
			Fields      []CustomValueInput `json:"fields"`
		}{s.record.Digest().String(), input.CustomReviewer, input.CustomValues})
		digest, err := canon.DigestBytes("StudioCustomExpectationReview", reviewBytes)
		if err != nil {
			return choice.RulingDraftInput{}, &APIError{Code: "STUDIO_CUSTOM_REVIEW_REFUSED", Detail: "custom review evidence could not be derived"}
		}
		evidence, err := domain.ParseDigest(digest.String())
		if err != nil {
			return choice.RulingDraftInput{}, &APIError{Code: "STUDIO_CUSTOM_REVIEW_REFUSED", Detail: "custom review evidence could not be retained"}
		}
		result.CustomExpectation = &tuple
		result.CustomReviewer = input.CustomReviewer
		result.CustomReviewEvidence = evidence
	} else if len(input.CustomValues) != 0 || input.CustomReviewer != "" {
		return choice.RulingDraftInput{}, &APIError{Code: "STUDIO_UNEXPECTED_CUSTOM_EXPECTATION", Detail: "only CUSTOM_EXPECTATION may carry authored values or review attribution"}
	}
	return result, nil
}

func validateAcknowledgements(differing []string, acknowledgements []FieldAcknowledgement, action choice.Action) ([]string, error) {
	if len(acknowledgements) != len(differing) {
		return nil, &APIError{Code: "STUDIO_FIELD_ACKNOWLEDGEMENT_REFUSED", Detail: "every differing field must be acknowledged exactly once"}
	}
	want := make(map[string]struct{}, len(differing))
	for _, field := range differing {
		want[field] = struct{}{}
	}
	seen := make(map[string]struct{}, len(acknowledgements))
	selected := make([]string, 0, len(acknowledgements))
	for _, acknowledgement := range acknowledgements {
		if _, present := want[acknowledgement.FieldID]; !present {
			return nil, &APIError{Code: "STUDIO_FIELD_ACKNOWLEDGEMENT_REFUSED", Detail: "an acknowledgement names a field outside the package-issued differing roster"}
		}
		if _, duplicate := seen[acknowledgement.FieldID]; duplicate {
			return nil, &APIError{Code: "STUDIO_FIELD_ACKNOWLEDGEMENT_REFUSED", Detail: "a differing field was acknowledged more than once"}
		}
		seen[acknowledgement.FieldID] = struct{}{}
		switch acknowledgement.Disposition {
		case "ASSERT":
			selected = append(selected, acknowledgement.FieldID)
		case "CONTEXT":
		default:
			return nil, &APIError{Code: "STUDIO_FIELD_ACKNOWLEDGEMENT_REFUSED", Detail: "field disposition must be ASSERT or CONTEXT"}
		}
	}
	if action == choice.ActionRejectAll || action == choice.ActionDefer || action == choice.ActionRefine {
		if len(selected) != 0 {
			return nil, &APIError{Code: "STUDIO_NONCOMPILABLE_PREDICATE_REFUSED", Detail: "reject, defer, and refine cannot carry asserted fields"}
		}
		return []string{}, nil
	}
	return selected, nil
}

func (s *studioState) customFields(selected []string, inputs []CustomValueInput) ([]choice.FieldValue, error) {
	if len(inputs) != len(selected) {
		return nil, &APIError{Code: "STUDIO_CUSTOM_EXPECTATION_REFUSED", Detail: "custom expectation must provide exactly one value for every asserted field"}
	}
	byID := make(map[string]CustomValueInput, len(inputs))
	for _, input := range inputs {
		if _, duplicate := byID[input.FieldID]; duplicate {
			return nil, &APIError{Code: "STUDIO_CUSTOM_EXPECTATION_REFUSED", Detail: "custom expectation repeats one field"}
		}
		byID[input.FieldID] = input
	}
	result := make([]choice.FieldValue, len(selected))
	for index, field := range selected {
		input, present := byID[field]
		if !present {
			return nil, &APIError{Code: "STUDIO_CUSTOM_EXPECTATION_REFUSED", Detail: "custom expectation omitted an asserted field"}
		}
		value, err := exactValue(input)
		if err != nil {
			return nil, err
		}
		result[index] = choice.FieldValue{FieldID: field, Value: value}
	}
	return result, nil
}

func exactValue(input CustomValueInput) (choice.ExactValue, error) {
	switch choice.ValueTag(input.Tag) {
	case choice.ValueMissing:
		return choice.MissingValue(), nil
	case choice.ValueNull:
		return choice.NullValue(), nil
	case choice.ValueString:
		value, err := choice.StringValue(input.Text)
		return value, apiChoiceError(err)
	case choice.ValueInteger:
		value, err := choice.IntegerValue(input.Text)
		return value, apiChoiceError(err)
	case choice.ValueBoolean:
		return choice.BooleanValue(input.Boolean), nil
	case choice.ValueBytes:
		body, err := base64.StdEncoding.Strict().DecodeString(input.BytesBase64)
		if err != nil || base64.StdEncoding.EncodeToString(body) != input.BytesBase64 {
			return choice.ExactValue{}, &APIError{Code: "STUDIO_CUSTOM_EXPECTATION_REFUSED", Detail: "custom bytes must use canonical base64"}
		}
		value, err := choice.BytesValue(body)
		return value, apiChoiceError(err)
	case choice.ValueOrderedStringList:
		value, err := choice.OrderedStringListValue(input.OrderedStrings)
		return value, apiChoiceError(err)
	case choice.ValueCanonicalJSON:
		value, err := choice.CanonicalJSONBytes(input.CanonicalJSON)
		return value, apiChoiceError(err)
	default:
		return choice.ExactValue{}, &APIError{Code: "STUDIO_CUSTOM_EXPECTATION_REFUSED", Detail: "custom value uses an unknown exact tag"}
	}
}

func apiChoiceError(err error) error {
	if err == nil {
		return nil
	}
	var refusal *choice.RefusalError
	if errors.As(err, &refusal) {
		return &APIError{Code: "CHOICE_" + string(refusal.Code), Detail: "the package-owned choice validator refused this transition"}
	}
	return &APIError{Code: "STUDIO_CHOICE_REFUSED", Detail: "the package-owned choice transition was refused"}
}

func cloneAcknowledgements(input []FieldAcknowledgement) []FieldAcknowledgement {
	if input == nil {
		return nil
	}
	result := make([]FieldAcknowledgement, len(input))
	copy(result, input)
	sort.Slice(result, func(i, j int) bool { return result[i].FieldID < result[j].FieldID })
	return result
}

func cloneDraft(input DraftInput) DraftInput {
	result := input
	result.AllowedAliases = make([]string, len(input.AllowedAliases))
	copy(result.AllowedAliases, input.AllowedAliases)
	result.FieldAcknowledgements = cloneAcknowledgements(input.FieldAcknowledgements)
	result.CustomValues = make([]CustomValueInput, len(input.CustomValues))
	copy(result.CustomValues, input.CustomValues)
	for index := range result.CustomValues {
		result.CustomValues[index].OrderedStrings = append([]string(nil), input.CustomValues[index].OrderedStrings...)
		result.CustomValues[index].CanonicalJSON = append(json.RawMessage(nil), input.CustomValues[index].CanonicalJSON...)
	}
	return result
}

func boundedVisibleText(value string, limit int) bool {
	return value != "" && boundedVisibleTextAllowEmpty(value, limit)
}

func boundedVisibleTextAllowEmpty(value string, limit int) bool {
	if len(value) > limit || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return strings.TrimSpace(value) == value
}
