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
	decisionMemberCount          = 40
	maxDecisionActorBytes        = 512
	maxDecisionAnnotationBytes   = 32 * 1024
	maxPortableDecisionLateBytes = 256 * 1024
	maxPortableDecisionBaseBytes = canon.MaxInputBytes - maxPortableDecisionLateBytes
	decisionScope                = "EXACT_WITNESSED_STIMULUS_V1"
	presentationFactScope        = "PRESENTED_NOT_COMPREHENDED_OR_DEBIASED"
	compileEligible              = "COMPILE_ELIGIBLE_EXACT_PREDICATE"
	compileIneligible            = "NONCOMPILABLE_ACTION"
	receiptInterpretationOpaque  = "OPAQUE_VERBATIM_REFERENCE_NOT_CONFORMANCE"
	actorAttributionCaller       = "LOCAL_CALLER_ASSERTED_OPERATOR"
	actorAuthenticityUnproven    = "AUTHENTICITY_NOT_ESTABLISHED_IN_U6"
)

type decisionSeparationIdentity struct {
	Status                 string `json:"status"`
	AllowedTupleCount      int    `json:"allowed_tuple_count"`
	DisallowedTupleCount   int    `json:"disallowed_tuple_count"`
	AllowedOutcomeCount    int    `json:"allowed_outcome_count"`
	DisallowedOutcomeCount int    `json:"disallowed_outcome_count"`
	ComparedPairCount      int    `json:"compared_pair_count"`
}

type decisionIdentity struct {
	SchemaVersion                 string                     `json:"schema_version"`
	Kind                          string                     `json:"kind"`
	ChoicepointDigest             string                     `json:"choicepoint_digest"`
	BlindPayloadDigest            string                     `json:"blind_payload_digest"`
	OriginalStimulusDigest        string                     `json:"original_stimulus_digest"`
	MinimizedStimulusDigest       string                     `json:"minimized_stimulus_digest"`
	Scope                         string                     `json:"scope"`
	GeneralizationClaimed         bool                       `json:"generalization_claimed"`
	ProvisionalRulingBase64       string                     `json:"provisional_ruling_base64"`
	ProvisionalRulingDigest       string                     `json:"provisional_ruling_digest"`
	ProvisionalAction             string                     `json:"provisional_action"`
	FinalRulingBase64             string                     `json:"final_ruling_base64"`
	FinalRulingDigest             string                     `json:"final_ruling_digest"`
	RevealBase64                  string                     `json:"reveal_base64"`
	RevealDigest                  string                     `json:"reveal_digest"`
	EarlyReveal                   bool                       `json:"early_reveal"`
	ChangedAfterReveal            bool                       `json:"changed_after_reveal"`
	PostRevealChangeRationale     string                     `json:"post_reveal_change_rationale"`
	VisitedSurfaces               []string                   `json:"visited_surfaces"`
	PreRevealVisitedSurfaces      []string                   `json:"pre_reveal_visited_surfaces"`
	PostRevealVisitedSurfaces     []string                   `json:"post_reveal_visited_surfaces"`
	PresentationFactScope         string                     `json:"presentation_fact_scope"`
	ActorID                       string                     `json:"actor_id"`
	ActorAttribution              string                     `json:"actor_attribution"`
	ActorAuthenticity             string                     `json:"actor_authenticity"`
	HumanAnnotation               string                     `json:"human_annotation"`
	Action                        string                     `json:"action"`
	SelectedFields                []string                   `json:"selected_fields"`
	NonassertedFields             []string                   `json:"nonasserted_fields"`
	AllowedAliases                []string                   `json:"allowed_aliases"`
	ConfirmedAllowedOutcomeIDs    []string                   `json:"confirmed_allowed_outcome_ids"`
	ConfirmedDisallowedOutcomeIDs []string                   `json:"confirmed_disallowed_outcome_ids"`
	AllowedCompleteTuples         []tupleWire                `json:"allowed_complete_tuples"`
	DisallowedCompleteTuples      []tupleWire                `json:"disallowed_complete_tuples"`
	SeparationCheck               decisionSeparationIdentity `json:"separation_check"`
	CompileEligibility            string                     `json:"compile_eligibility"`
	Compilable                    bool                       `json:"compilable"`
	DidrunReceipts                []domain.ReceiptWire       `json:"didrun_receipts"`
	ReceiptStatusWhenEmpty        string                     `json:"receipt_status_when_empty"`
	ReceiptInterpretation         string                     `json:"receipt_interpretation"`
}

// DecisionRecord is an immutable, strict human record against one exact
// Choicepoint. Its compile eligibility is reconstructed by ValidateRuling and
// retained as a sealed sum; no serialized boolean can authorize an emitter.
type DecisionRecord struct {
	digest         domain.Digest
	canonicalBytes []byte
	choicepoint    ChoicepointRecord
	identity       decisionIdentity
	validated      ValidatedRuling
	receipts       []domain.ReceiptReference
}

func newDecisionRecord(
	session Session,
	actor string,
	annotation string,
	receipts []domain.ReceiptReference,
	expected []byte,
) (DecisionRecord, error) {
	return buildDecisionRecord(session, actor, annotation, receipts, expected, true)
}

func buildDecisionRecord(
	session Session,
	actor string,
	annotation string,
	receipts []domain.ReceiptReference,
	expected []byte,
	enforcePortableBudget bool,
) (DecisionRecord, error) {
	if !session.record.Valid() || session.state != SessionPostRevealRecorded || !session.revealed || session.final == nil {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "decision construction requires a complete post-reveal session")
	}
	view, err := NewBlindView(session.record)
	if err != nil || !session.view.same(view) {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "decision session does not bind the exact blind Choicepoint view")
	}
	for _, surface := range requiredReviewSurfaces {
		if _, visited := session.visits[surface]; !visited {
			return DecisionRecord{}, refusal(CodeRequiredSurfaceNotVisited, string(surface))
		}
	}
	if actor == "" || len(actor) > maxDecisionActorBytes || !utf8.ValidString(actor) ||
		strings.TrimSpace(actor) == "" || containsControl(actor) || len(annotation) > maxDecisionAnnotationBytes ||
		!utf8.ValidString(annotation) || strings.ContainsRune(annotation, '\x00') {
		return DecisionRecord{}, refusal(CodeInvalidDecisionAttribution, "actor or annotation is outside the bounded human-attribution profile")
	}

	earlyReveal := session.provisional == nil
	for surface := range session.preRevealVisits {
		if surface == SurfaceProvenance {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "provenance cannot be a pre-reveal presentation fact")
		}
		if _, visited := session.visits[surface]; !visited {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "pre-reveal visit is absent from the complete visit set")
		}
	}
	if !earlyReveal {
		for _, surface := range requiredReviewSurfaces {
			if surface == SurfaceProvenance {
				continue
			}
			if _, visited := session.preRevealVisits[surface]; !visited {
				return DecisionRecord{}, refusal(CodeRequiredSurfaceNotVisited, "standard blind-first ruling lacks pre-reveal "+string(surface))
			}
		}
	}
	changed := !earlyReveal && session.provisional.digest != session.final.digest
	if changed && (session.rationale == "" || strings.TrimSpace(session.rationale) == "" ||
		len(session.rationale) > 8*1024 || !utf8.ValidString(session.rationale) || containsControl(session.rationale)) {
		return DecisionRecord{}, refusal(CodeChangeRationaleRequired, "changed post-reveal ruling requires an exact rationale")
	}
	if !changed && session.rationale != "" {
		return DecisionRecord{}, refusal(CodeUnnecessaryChangeRationale, "unchanged or early-reveal ruling carries a rationale")
	}

	reveal, err := buildRevealDTO(session.record, view)
	if err != nil {
		return DecisionRecord{}, err
	}
	projection, err := decisionProjection(session.record, session.final.validated)
	if err != nil {
		return DecisionRecord{}, err
	}
	receiptCopies, err := normalizeDecisionReceipts(receipts)
	if err != nil {
		return DecisionRecord{}, err
	}
	visited := make([]string, len(requiredReviewSurfaces))
	preRevealVisited := []string{}
	postRevealVisited := []string{}
	for index, surface := range requiredReviewSurfaces {
		visited[index] = string(surface)
		if _, beforeReveal := session.preRevealVisits[surface]; beforeReveal {
			preRevealVisited = append(preRevealVisited, string(surface))
		} else {
			postRevealVisited = append(postRevealVisited, string(surface))
		}
	}
	identity := decisionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "DecisionRecord",
		ChoicepointDigest: session.record.digest.String(), BlindPayloadDigest: view.dto.digest.String(),
		OriginalStimulusDigest:  session.record.original.digest.String(),
		MinimizedStimulusDigest: session.record.minimized.digest.String(), Scope: decisionScope,
		GeneralizationClaimed: false, FinalRulingBase64: base64.StdEncoding.EncodeToString(session.final.canonicalBytes),
		FinalRulingDigest: session.final.digest.String(), RevealBase64: base64.StdEncoding.EncodeToString(reveal.canonical),
		RevealDigest: reveal.digest.String(), EarlyReveal: earlyReveal, ChangedAfterReveal: changed,
		PostRevealChangeRationale: session.rationale, VisitedSurfaces: visited,
		PreRevealVisitedSurfaces: preRevealVisited, PostRevealVisitedSurfaces: postRevealVisited,
		PresentationFactScope: presentationFactScope,
		ActorID:               actor, ActorAttribution: actorAttributionCaller,
		ActorAuthenticity: actorAuthenticityUnproven, HumanAnnotation: annotation,
		Action: string(session.final.input.Action), SelectedFields: cloneExplicitStrings(session.final.input.SelectedFields),
		NonassertedFields: projection.nonassertedFields, AllowedAliases: cloneExplicitStrings(session.final.input.AllowedAliases),
		ConfirmedAllowedOutcomeIDs:    projection.allowedOutcomeIDs,
		ConfirmedDisallowedOutcomeIDs: projection.disallowedOutcomeIDs,
		AllowedCompleteTuples:         projection.allowedTuples, DisallowedCompleteTuples: projection.disallowedTuples,
		SeparationCheck: projection.separation, CompileEligibility: projection.eligibility,
		Compilable:     projection.eligibility == compileEligible,
		DidrunReceipts: make([]domain.ReceiptWire, len(receiptCopies)), ReceiptStatusWhenEmpty: domain.Unreceipted(),
		ReceiptInterpretation: receiptInterpretationOpaque,
	}
	if session.provisional != nil {
		identity.ProvisionalRulingBase64 = base64.StdEncoding.EncodeToString(session.provisional.canonicalBytes)
		identity.ProvisionalRulingDigest = session.provisional.digest.String()
		identity.ProvisionalAction = string(session.provisional.input.Action)
	}
	for index, receipt := range receiptCopies {
		identity.DidrunReceipts[index] = receipt.Wire()
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("DecisionRecord", identity)
	if err != nil || len(canonicalBytes) > canon.MaxInputBytes {
		return DecisionRecord{}, refusal(CodeInputLimitExceeded, "DecisionRecord exceeds the canonical resource profile")
	}
	if enforcePortableBudget && session.record.mode == choicepointPortable {
		base, baseErr := buildDecisionRecord(session, "x", "", []domain.ReceiptReference{}, nil, false)
		if baseErr != nil || len(base.canonicalBytes) > maxPortableDecisionBaseBytes {
			return DecisionRecord{}, refusal(CodeInputLimitExceeded, "portable DecisionRecord structural body exceeds its reserved durable budget")
		}
		lateBytes := len(canonicalBytes) - len(base.canonicalBytes)
		if lateBytes < 0 || lateBytes > maxPortableDecisionLateBytes {
			return DecisionRecord{}, refusal(CodeInputLimitExceeded, "portable DecisionRecord attribution and receipts exceed their reserved durable budget")
		}
	}
	if expected != nil && !bytes.Equal(expected, canonicalBytes) {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord wire did not reconstruct exactly")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return DecisionRecord{}, err
	}
	return DecisionRecord{
		digest: digest, canonicalBytes: canonicalBytes, choicepoint: session.record,
		identity: identity, validated: session.final.validated, receipts: receiptCopies,
	}, nil
}

type projectedDecision struct {
	nonassertedFields    []string
	allowedOutcomeIDs    []string
	disallowedOutcomeIDs []string
	allowedTuples        []tupleWire
	disallowedTuples     []tupleWire
	separation           decisionSeparationIdentity
	eligibility          string
}

func decisionProjection(record ChoicepointRecord, ruling ValidatedRuling) (projectedDecision, error) {
	selected := map[string]struct{}{}
	result := projectedDecision{
		nonassertedFields: []string{}, allowedOutcomeIDs: []string{}, disallowedOutcomeIDs: []string{},
		allowedTuples: []tupleWire{}, disallowedTuples: []tupleWire{}, eligibility: compileIneligible,
		separation: decisionSeparationIdentity{Status: "NOT_APPLICABLE"},
	}
	for _, field := range record.confirmed.registry.Definitions() {
		selected[field.ID] = struct{}{}
	}
	compilable, ok := ruling.eligibility.(CompilableRuling)
	if ok {
		result.eligibility = compileEligible
		result.separation = decisionSeparation(ruling.separation, "PASSED")
		selected = make(map[string]struct{}, len(compilable.SelectedFields()))
		for _, field := range compilable.SelectedFields() {
			selected[field.String()] = struct{}{}
		}
		result.allowedOutcomeIDs = outcomeIDs(compilable.AllowedOutcomes())
		result.disallowedOutcomeIDs = outcomeIDs(compilable.DisallowedOutcomes())
		var err error
		result.allowedTuples, err = tuplesToWire(record.confirmed.registry, compilable.AllowedTuples())
		if err != nil {
			return projectedDecision{}, err
		}
		result.disallowedTuples, err = tuplesToWire(record.confirmed.registry, compilable.DisallowedTuples())
		if err != nil {
			return projectedDecision{}, err
		}
	} else {
		if _, sealed := ruling.eligibility.(NoncompilableRuling); !sealed {
			return projectedDecision{}, refusal(CodeInvalidSessionState, "decision ruling has no sealed compile-eligibility variant")
		}
		selected = map[string]struct{}{}
	}
	for _, field := range record.confirmed.registry.Definitions() {
		if _, asserted := selected[field.ID]; !asserted {
			result.nonassertedFields = append(result.nonassertedFields, field.ID)
		}
	}
	return result, nil
}

func decisionSeparation(check SeparationCheck, status string) decisionSeparationIdentity {
	return decisionSeparationIdentity{
		Status: status, AllowedTupleCount: check.AllowedTupleCount,
		DisallowedTupleCount: check.DisallowedTupleCount, AllowedOutcomeCount: check.AllowedOutcomeCount,
		DisallowedOutcomeCount: check.DisallowedOutcomeCount, ComparedPairCount: check.ComparedPairCount,
	}
}

func outcomeIDs(refs []ConfirmedOutcomeRef) []string {
	result := make([]string, len(refs))
	for index, ref := range refs {
		result[index] = ref.ID().String()
	}
	sort.Strings(result)
	return result
}

func tuplesToWire(registry FieldRegistry, tuples []CompleteTuple) ([]tupleWire, error) {
	ordered := cloneTuples(tuples)
	sort.Slice(ordered, func(i, j int) bool {
		return tupleIdentityKey(registry, ordered[i].Fields) < tupleIdentityKey(registry, ordered[j].Fields)
	})
	result := make([]tupleWire, len(ordered))
	for index, tuple := range ordered {
		wire, err := tupleToWire(registry, tuple)
		if err != nil {
			return nil, err
		}
		result[index] = wire
	}
	return result, nil
}

func normalizeDecisionReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {
	if len(receipts) > canon.MaxContainerMembers {
		return nil, refusal(CodeInputLimitExceeded, "DecisionRecord receipt count exceeds the canonical container profile")
	}
	remaining := canon.MaxInputBytes
	for _, receipt := range receipts {
		wire := receipt.Wire()
		if !consumeReceiptWireBudget(wire, &remaining) {
			return nil, refusal(CodeInputLimitExceeded, "DecisionRecord receipt payload exceeds the canonical input profile")
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
			return nil, refusal(CodeInvalidSessionState, "DecisionRecord receipt reference is invalid")
		}
		keyed[index] = keyedReceipt{receipt: receipt, key: receiptIdentityKey(receipt)}
	}
	sort.Slice(keyed, func(i, j int) bool { return keyed[i].key < keyed[j].key })
	result := make([]domain.ReceiptReference, len(keyed))
	for index, entry := range keyed {
		if index > 0 && entry.key == keyed[index-1].key {
			return nil, refusal(CodeInvalidSessionState, "DecisionRecord receipt reference is duplicated")
		}
		result[index] = entry.receipt
	}
	return result, nil
}

// ParseDecisionRecord strictly reconstructs all derived decision facts from the
// exact Choicepoint and final ruling draft. Serializable bytes alone never
// acquire current store authority.
func ParseDecisionRecord(exact []byte, record ChoicepointRecord) (DecisionRecord, error) {
	if !record.Valid() {
		return DecisionRecord{}, refusal(CodeInvalidConfirmedOutcomeSet, "DecisionRecord requires an exact Choicepoint")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord wire is outside the canonical profile")
	}
	canonicalBytes, canonicalErr := value.CanonicalChecked()
	members, object := value.Members()
	if canonicalErr != nil || !bytes.Equal(canonicalBytes, exact) || !object || len(members) != decisionMemberCount {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord wire is nonexact or has unknown/missing members")
	}
	var identity decisionIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord typed wire decode failed")
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "DecisionRecord" ||
		identity.ChoicepointDigest != record.digest.String() || identity.BlindPayloadDigest == "" ||
		identity.OriginalStimulusDigest != record.original.digest.String() ||
		identity.MinimizedStimulusDigest != record.minimized.digest.String() || identity.Scope != decisionScope ||
		identity.GeneralizationClaimed || identity.PresentationFactScope != presentationFactScope ||
		identity.ActorAttribution != actorAttributionCaller || identity.ActorAuthenticity != actorAuthenticityUnproven ||
		identity.ReceiptStatusWhenEmpty != domain.Unreceipted() ||
		identity.ReceiptInterpretation != receiptInterpretationOpaque {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord closed wire facts disagree")
	}
	if identity.VisitedSurfaces == nil || identity.PreRevealVisitedSurfaces == nil || identity.PostRevealVisitedSurfaces == nil ||
		identity.SelectedFields == nil || identity.NonassertedFields == nil ||
		identity.AllowedAliases == nil || identity.ConfirmedAllowedOutcomeIDs == nil ||
		identity.ConfirmedDisallowedOutcomeIDs == nil || identity.AllowedCompleteTuples == nil ||
		identity.DisallowedCompleteTuples == nil || identity.DidrunReceipts == nil {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord explicit arrays cannot be null")
	}
	view, err := NewBlindView(record)
	if err != nil {
		return DecisionRecord{}, err
	}
	if identity.BlindPayloadDigest != view.dto.digest.String() {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord blind payload differs")
	}
	finalBytes, err := decodeDecisionBase64(identity.FinalRulingBase64)
	if err != nil {
		return DecisionRecord{}, err
	}
	final, err := parseRulingDraft(finalBytes, record, view)
	if err != nil || final.digest.String() != identity.FinalRulingDigest || string(final.input.Action) != identity.Action {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord final ruling differs")
	}
	revealBytes, err := decodeDecisionBase64(identity.RevealBase64)
	if err != nil {
		return DecisionRecord{}, err
	}
	reveal, err := buildRevealDTO(record, view)
	if err != nil || !bytes.Equal(reveal.canonical, revealBytes) || reveal.digest.String() != identity.RevealDigest {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord reveal differs")
	}
	var provisional *rulingDraft
	if identity.EarlyReveal {
		if identity.ProvisionalRulingBase64 != "" || identity.ProvisionalRulingDigest != "" || identity.ProvisionalAction != "" {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "early reveal cannot claim a provisional ruling")
		}
	} else {
		provisionalBytes, decodeErr := decodeDecisionBase64(identity.ProvisionalRulingBase64)
		if decodeErr != nil {
			return DecisionRecord{}, decodeErr
		}
		parsed, parseErr := parseRulingDraft(provisionalBytes, record, view)
		if parseErr != nil || parsed.digest.String() != identity.ProvisionalRulingDigest || string(parsed.input.Action) != identity.ProvisionalAction {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord provisional ruling differs")
		}
		provisional = &parsed
	}
	visits := make(map[ReviewSurface]struct{}, len(requiredReviewSurfaces))
	preRevealVisits := make(map[ReviewSurface]struct{}, len(identity.PreRevealVisitedSurfaces))
	if len(identity.VisitedSurfaces) != len(requiredReviewSurfaces) {
		return DecisionRecord{}, refusal(CodeRequiredSurfaceNotVisited, "DecisionRecord visit roster differs")
	}
	for index, surface := range requiredReviewSurfaces {
		if identity.VisitedSurfaces[index] != string(surface) {
			return DecisionRecord{}, refusal(CodeRequiredSurfaceNotVisited, "DecisionRecord visit order differs")
		}
		visits[surface] = struct{}{}
	}
	preIndex := 0
	postIndex := 0
	for _, surface := range requiredReviewSurfaces {
		if preIndex < len(identity.PreRevealVisitedSurfaces) && identity.PreRevealVisitedSurfaces[preIndex] == string(surface) {
			preRevealVisits[surface] = struct{}{}
			preIndex++
			continue
		}
		if postIndex >= len(identity.PostRevealVisitedSurfaces) || identity.PostRevealVisitedSurfaces[postIndex] != string(surface) {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord pre/post reveal visit partition differs")
		}
		postIndex++
	}
	if preIndex != len(identity.PreRevealVisitedSurfaces) || postIndex != len(identity.PostRevealVisitedSurfaces) {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord pre/post reveal visit roster differs")
	}
	if _, preRevealProvenance := preRevealVisits[SurfaceProvenance]; preRevealProvenance {
		return DecisionRecord{}, refusal(CodeInvalidSessionState, "provenance cannot be visited before reveal")
	}
	receipts := make([]domain.ReceiptReference, len(identity.DidrunReceipts))
	for index, wire := range identity.DidrunReceipts {
		receipts[index], err = domain.ReceiptFromWire(wire)
		if err != nil || (index > 0 && receiptIdentityKey(receipts[index]) <= receiptIdentityKey(receipts[index-1])) {
			return DecisionRecord{}, refusal(CodeInvalidSessionState, "DecisionRecord receipt order differs")
		}
	}
	session := Session{
		record: record, view: view, state: SessionPostRevealRecorded, visits: visits, preRevealVisits: preRevealVisits,
		provisional: provisional, final: &final, revealed: true, rationale: identity.PostRevealChangeRationale,
	}
	rebuilt, err := newDecisionRecord(session, identity.ActorID, identity.HumanAnnotation, receipts, exact)
	if err != nil {
		return DecisionRecord{}, err
	}
	return rebuilt, nil
}

func decodeDecisionBase64(raw string) ([]byte, error) {
	body, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || len(body) == 0 || base64.StdEncoding.EncodeToString(body) != raw {
		return nil, refusal(CodeInvalidSessionState, "DecisionRecord nested bytes are noncanonical base64")
	}
	return body, nil
}

func (r DecisionRecord) Valid() bool {
	parsed, err := ParseDecisionRecord(r.canonicalBytes, r.choicepoint)
	return err == nil && parsed.digest == r.digest && bytes.Equal(parsed.canonicalBytes, r.canonicalBytes)
}

func (r DecisionRecord) Digest() domain.Digest            { return r.digest }
func (r DecisionRecord) CanonicalBytes() []byte           { return append([]byte(nil), r.canonicalBytes...) }
func (r DecisionRecord) ChoicepointDigest() domain.Digest { return r.choicepoint.digest }
func (r DecisionRecord) Action() Action                   { return r.validated.Action() }
func (r DecisionRecord) ActorID() string                  { return r.identity.ActorID }
func (r DecisionRecord) HumanAnnotation() string          { return r.identity.HumanAnnotation }
func (r DecisionRecord) EarlyReveal() bool                { return r.identity.EarlyReveal }
func (r DecisionRecord) ChangedAfterReveal() bool         { return r.identity.ChangedAfterReveal }
func (r DecisionRecord) PostRevealChangeRationale() string {
	return r.identity.PostRevealChangeRationale
}
func (r DecisionRecord) ReceiptStatusWhenEmpty() string { return r.identity.ReceiptStatusWhenEmpty }
func (r DecisionRecord) CompileEligibility() CompileEligibility {
	return r.validated.CompileEligibility()
}

func (r DecisionRecord) CompilableRuling() (CompilableRuling, bool) {
	value, ok := r.validated.CompileEligibility().(CompilableRuling)
	return value, ok
}

func (r DecisionRecord) SelectedFields() []string {
	return append([]string(nil), r.identity.SelectedFields...)
}

func (r DecisionRecord) NonassertedFields() []string {
	return append([]string(nil), r.identity.NonassertedFields...)
}

func (r DecisionRecord) VisitedSurfaces() []ReviewSurface {
	result := make([]ReviewSurface, len(r.identity.VisitedSurfaces))
	for index, surface := range r.identity.VisitedSurfaces {
		result[index] = ReviewSurface(surface)
	}
	return result
}

func (r DecisionRecord) ConfirmedAllowedOutcomeIDs() []string {
	return append([]string(nil), r.identity.ConfirmedAllowedOutcomeIDs...)
}

func (r DecisionRecord) ConfirmedDisallowedOutcomeIDs() []string {
	return append([]string(nil), r.identity.ConfirmedDisallowedOutcomeIDs...)
}

func (r DecisionRecord) Receipts() []domain.ReceiptReference {
	return append([]domain.ReceiptReference(nil), r.receipts...)
}
