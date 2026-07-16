// Package promotion composes immutable choice semantics with current durable
// store authority. Parent package choice remains a pure semantic kernel.
package promotion

import (
	"bytes"
	"context"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/choice"
	choicepublication "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/store"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

const CodeRefineRequiresSuccessorStudy = "REFINE_REQUIRES_SUCCESSOR_STUDY"

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type storedConfirmationSeal struct{ marker byte }

// StoredConfirmation proves that exact strict FreshConfirmation bytes are the
// current CONFIRMATION head in this store instance.
type StoredConfirmation struct {
	head      store.HeadToken
	object    store.SemanticObject
	authority store.ObjectAuthority
	record    confirmation.Record
	seal      *storedConfirmationSeal
}

func PersistConfirmation(
	ctx context.Context,
	objectStore *store.ObjectStore,
	expectedReduction store.HeadToken,
	draft confirmation.Draft,
) (StoredConfirmation, error) {
	if ctx == nil || objectStore == nil || !draft.Valid() || expectedReduction.Stage() != store.StageReduction {
		return StoredConfirmation{}, refuse("INVALID_CONFIRMATION_PROMOTION", "live reduction head and confirmation draft are required", nil)
	}
	record := draft.Record()
	if !reductionHeadBindsRecord(expectedReduction, record) {
		return StoredConfirmation{}, refuse("CONFIRMATION_REDUCTION_HEAD_MISMATCH", "current reduction object does not bind the confirmation record", nil)
	}
	head, err := objectStore.AdvanceConfirmation(ctx, expectedReduction, draft.PublicationAuthority())
	if err != nil {
		return StoredConfirmation{}, err
	}
	return openConfirmationAtHead(ctx, objectStore, head)
}

func OpenConfirmation(
	ctx context.Context,
	objectStore *store.ObjectStore,
	study store.StudyID,
) (StoredConfirmation, error) {
	if ctx == nil || objectStore == nil || !study.Valid() {
		return StoredConfirmation{}, refuse("INVALID_CONFIRMATION_PROMOTION", "store and study are required", nil)
	}
	head, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return StoredConfirmation{}, err
	}
	return openConfirmationAtHead(ctx, objectStore, head)
}

func openConfirmationAtHead(
	ctx context.Context,
	objectStore *store.ObjectStore,
	head store.HeadToken,
) (StoredConfirmation, error) {
	if head.Stage() != store.StageConfirmation || head.CurrentKind() != "FreshConfirmation" {
		return StoredConfirmation{}, refuse("CONFIRMATION_NOT_CURRENT", "study head is not an exact FreshConfirmation", nil)
	}
	object, authority, err := objectStore.Read(ctx, head.CurrentKind(), head.CurrentDigest())
	if err != nil {
		return StoredConfirmation{}, err
	}
	if err := objectStore.Validate(ctx, object, authority); err != nil {
		return StoredConfirmation{}, err
	}
	record, err := confirmation.ParseRecord(object.CanonicalBytes())
	if err != nil || record.Digest() != head.CurrentDigest() || record.Digest() != object.Digest() {
		return StoredConfirmation{}, refuse("INVALID_STORED_CONFIRMATION", "head, object, and strict record disagree", err)
	}
	stored := StoredConfirmation{
		head: head, object: object, authority: authority, record: record,
		seal: &storedConfirmationSeal{marker: 1},
	}
	if err := validateStoredConfirmation(ctx, objectStore, stored); err != nil {
		return StoredConfirmation{}, err
	}
	return stored, nil
}

func reductionHeadBindsRecord(head store.HeadToken, record confirmation.Record) bool {
	if !record.Valid() || head.Stage() != store.StageReduction {
		return false
	}
	previous, hasPrevious := head.PreviousObjectDigest()
	if !hasPrevious || head.LineageRootDigest() != record.PlanDigest() ||
		previous != record.OriginalBaselineMap().ArtifactDigest().DomainDigest() {
		return false
	}
	switch head.CurrentKind() {
	case "ReductionRun":
		return head.CurrentDigest() == record.ReductionRunDigest()
	case "ReductionGrade":
		return head.CurrentDigest() == record.ReductionGradeDigest()
	default:
		return false
	}
}

func validateStoredConfirmation(
	ctx context.Context,
	objectStore *store.ObjectStore,
	stored StoredConfirmation,
) error {
	if stored.seal == nil || stored.seal.marker != 1 || !stored.record.Valid() ||
		stored.object.Kind() != "FreshConfirmation" || stored.object.Digest() != stored.record.Digest() ||
		!bytes.Equal(stored.object.CanonicalBytes(), stored.record.CanonicalBytes()) {
		return refuse("INVALID_STORED_CONFIRMATION", "stored confirmation capability is incomplete", nil)
	}
	if err := objectStore.Validate(ctx, stored.object, stored.authority); err != nil {
		return err
	}
	current, err := objectStore.OpenHead(ctx, stored.head.StudyID())
	if err != nil {
		return err
	}
	if !sameHead(current, stored.head) || current.Stage() != store.StageConfirmation ||
		current.CurrentKind() != "FreshConfirmation" || current.CurrentDigest() != stored.record.Digest() {
		return refuse("CONFIRMATION_NOT_CURRENT", "stored confirmation was superseded", nil)
	}
	return nil
}

func (s StoredConfirmation) Record() confirmation.Record {
	parsed, _ := confirmation.ParseRecord(s.record.CanonicalBytes())
	return parsed
}

func (s StoredConfirmation) StudyID() store.StudyID { return s.head.StudyID() }

type ChoicepointRequest struct {
	Scenario          string
	Plan              domain.WorldPlan
	Envelope          domain.ComparisonEnvelope
	CandidateBindings []domain.CandidateExecutionBinding
	OriginalStimulus  choice.CanonicalArtifact
	MinimizedStimulus choice.CanonicalArtifact
	CandidateReveals  []choice.CandidateReveal
	EvidenceReceipts  []domain.ReceiptReference
}

type readySeal struct{ marker byte }

// Ready is the only U6 capability that authorizes durable publication from the
// exact current CHOICEPOINT_READY head. ChoicepointRecord remains an inert,
// inspectable semantic record and can drive a pure local preview session, but
// it cannot authorize persistence.
type Ready struct {
	head                  store.HeadToken
	object                store.SemanticObject
	authority             store.ObjectAuthority
	confirmationObject    store.SemanticObject
	confirmationAuthority store.ObjectAuthority
	record                choice.ChoicepointRecord
	seal                  *readySeal
}

func Promote(
	ctx context.Context,
	objectStore *store.ObjectStore,
	stored StoredConfirmation,
	request ChoicepointRequest,
) (Ready, error) {
	if ctx == nil || objectStore == nil {
		return Ready{}, refuse("INVALID_CHOICEPOINT_PROMOTION", "store and context are required", nil)
	}
	if err := validateStoredConfirmation(ctx, objectStore, stored); err != nil {
		return Ready{}, err
	}
	record, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: request.Scenario, Plan: request.Plan, Envelope: request.Envelope,
		CandidateBindings: request.CandidateBindings, OriginalStimulus: request.OriginalStimulus,
		MinimizedStimulus: request.MinimizedStimulus, Confirmation: stored.record,
		CandidateReveals: request.CandidateReveals, EvidenceReceipts: request.EvidenceReceipts,
	})
	if err != nil {
		return Ready{}, err
	}
	publicationAuthority, err := choicepublication.IssueChoicepoint(
		record.Digest(), stored.record.Digest(), record.CanonicalBytes(),
	)
	if err != nil {
		return Ready{}, refuse("INVALID_CHOICEPOINT_OBJECT", "Choicepoint publication authority could not be issued", err)
	}
	head, err := objectStore.AdvanceChoicepoint(ctx, stored.head, publicationAuthority)
	if err != nil {
		return Ready{}, err
	}
	return openReadyAtHead(ctx, objectStore, head)
}

func OpenReady(ctx context.Context, objectStore *store.ObjectStore, study store.StudyID) (Ready, error) {
	if ctx == nil || objectStore == nil || !study.Valid() {
		return Ready{}, refuse("INVALID_CHOICEPOINT_PROMOTION", "store and study are required", nil)
	}
	head, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return Ready{}, err
	}
	return openReadyAtHead(ctx, objectStore, head)
}

func openReadyAtHead(ctx context.Context, objectStore *store.ObjectStore, head store.HeadToken) (Ready, error) {
	if head.Stage() != store.StageChoicepointReady || head.CurrentKind() != "Choicepoint" {
		return Ready{}, refuse("CHOICEPOINT_NOT_CURRENT", "study head is not CHOICEPOINT_READY", nil)
	}
	object, authority, err := objectStore.Read(ctx, head.CurrentKind(), head.CurrentDigest())
	if err != nil {
		return Ready{}, err
	}
	if err := objectStore.Validate(ctx, object, authority); err != nil {
		return Ready{}, err
	}
	record, err := choice.ParseChoicepointRecord(object.CanonicalBytes())
	if err != nil || record.Digest() != object.Digest() || record.Digest() != head.CurrentDigest() {
		return Ready{}, refuse("INVALID_STORED_CHOICEPOINT", "head, object, and strict Choicepoint disagree", err)
	}
	previous, hasPrevious := head.PreviousObjectDigest()
	if !hasPrevious || previous != record.ConfirmationDigest() {
		return Ready{}, refuse("INVALID_STORED_CHOICEPOINT", "Choicepoint head does not descend from its exact confirmation", nil)
	}
	confirmationObject, confirmationAuthority, err := objectStore.Read(ctx, "FreshConfirmation", record.ConfirmationDigest())
	if err != nil {
		return Ready{}, err
	}
	if err := objectStore.Validate(ctx, confirmationObject, confirmationAuthority); err != nil {
		return Ready{}, err
	}
	confirmationRecord := record.ConfirmationRecord()
	if !confirmationRecord.Valid() || !bytes.Equal(confirmationObject.CanonicalBytes(), confirmationRecord.CanonicalBytes()) {
		return Ready{}, refuse("INVALID_STORED_CHOICEPOINT", "embedded and durable confirmation bytes disagree", nil)
	}
	ready := Ready{
		head: head, object: object, authority: authority, confirmationObject: confirmationObject,
		confirmationAuthority: confirmationAuthority, record: record, seal: &readySeal{marker: 1},
	}
	if err := validateReady(ctx, objectStore, ready); err != nil {
		return Ready{}, err
	}
	return ready, nil
}

func validateReady(ctx context.Context, objectStore *store.ObjectStore, ready Ready) error {
	if ready.seal == nil || ready.seal.marker != 1 || !ready.record.Valid() ||
		ready.object.Kind() != "Choicepoint" || ready.object.Digest() != ready.record.Digest() {
		return refuse("INVALID_STORED_CHOICEPOINT", "ready capability is incomplete", nil)
	}
	if err := objectStore.Validate(ctx, ready.object, ready.authority); err != nil {
		return err
	}
	if err := objectStore.Validate(ctx, ready.confirmationObject, ready.confirmationAuthority); err != nil {
		return err
	}
	current, err := objectStore.OpenHead(ctx, ready.head.StudyID())
	if err != nil {
		return err
	}
	if !sameHead(current, ready.head) || current.Stage() != store.StageChoicepointReady ||
		current.CurrentDigest() != ready.record.Digest() {
		return refuse("CHOICEPOINT_NOT_CURRENT", "ready capability was superseded", nil)
	}
	return nil
}

func (r Ready) Record() choice.ChoicepointRecord {
	parsed, _ := choice.ParseChoicepointRecord(r.record.CanonicalBytes())
	return parsed
}

func (r Ready) StudyID() store.StudyID { return r.head.StudyID() }

type rulingSeal struct{ marker byte }

type portableRulingPreparationSeal struct{ marker byte }

// Ruling is current durable authority for one strict DecisionRecord and its
// exact Choicepoint predecessor. The DecisionRecord bytes remain inert without
// this store/head-bound capability.
type Ruling struct {
	head                  store.HeadToken
	object                store.SemanticObject
	authority             store.ObjectAuthority
	choicepointObject     store.SemanticObject
	choicepointAuthority  store.ObjectAuthority
	confirmationObject    store.SemanticObject
	confirmationAuthority store.ObjectAuthority
	record                choice.DecisionRecord
	choicepoint           choice.ChoicepointRecord
	seal                  *rulingSeal
}

// Finalize advances only a still-current CHOICEPOINT_READY head. The semantic
// DecisionRecord is fully constructed before this call, but it is not published
// until the typed store transition has won the exact compare-and-swap.
func Finalize(
	ctx context.Context,
	objectStore *store.ObjectStore,
	ready Ready,
	decision choice.DecisionRecord,
) (Ruling, error) {
	if ctx == nil || objectStore == nil || !decision.Valid() {
		return Ruling{}, refuse("INVALID_RULING_PROMOTION", "store, context, and strict DecisionRecord are required", nil)
	}
	if err := validateReady(ctx, objectStore, ready); err != nil {
		return Ruling{}, err
	}
	if decision.ChoicepointDigest() != ready.record.Digest() {
		return Ruling{}, refuse("RULING_CHOICEPOINT_MISMATCH", "DecisionRecord does not bind the current Choicepoint", nil)
	}
	// REFINE is a valid semantic decision, but its durable transition is a new
	// successor study rather than an in-place RULING on this lineage. U6 has no
	// construction-safe successor-study authority, so fail before constructing
	// or publishing a semantic object instead of overstating the transition.
	if decision.Action() == choice.ActionRefine {
		return Ruling{}, refuse(
			CodeRefineRequiresSuccessorStudy,
			"durable REFINE requires successor-study creation; current Choicepoint and head remain unchanged",
			nil,
		)
	}
	publicationAuthority, err := choicepublication.IssueRuling(
		decision.Digest(), ready.record.Digest(), string(decision.Action()), decision.CanonicalBytes(),
	)
	if err != nil {
		return Ruling{}, refuse("INVALID_RULING_OBJECT", "DecisionRecord publication authority could not be issued", err)
	}
	head, err := objectStore.AdvanceRuling(ctx, ready.head, publicationAuthority)
	if err != nil {
		return Ruling{}, err
	}
	return openRulingAtHead(ctx, objectStore, head)
}

func OpenRuling(ctx context.Context, objectStore *store.ObjectStore, study store.StudyID) (Ruling, error) {
	if ctx == nil || objectStore == nil || !study.Valid() {
		return Ruling{}, refuse("INVALID_RULING_PROMOTION", "store and study are required", nil)
	}
	head, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return Ruling{}, err
	}
	return openRulingAtHead(ctx, objectStore, head)
}

func openRulingAtHead(ctx context.Context, objectStore *store.ObjectStore, head store.HeadToken) (Ruling, error) {
	if head.Stage() != store.StageRuling || head.CurrentKind() != "DecisionRecord" {
		return Ruling{}, refuse("RULING_NOT_CURRENT", "study head is not an exact DecisionRecord ruling", nil)
	}
	object, authority, err := objectStore.Read(ctx, head.CurrentKind(), head.CurrentDigest())
	if err != nil {
		return Ruling{}, err
	}
	if err := objectStore.Validate(ctx, object, authority); err != nil {
		return Ruling{}, err
	}
	previous, hasPrevious := head.PreviousObjectDigest()
	if !hasPrevious {
		return Ruling{}, refuse("INVALID_STORED_RULING", "ruling head lacks its exact Choicepoint predecessor", nil)
	}
	choicepointObject, choicepointAuthority, err := objectStore.Read(ctx, "Choicepoint", previous)
	if err != nil {
		return Ruling{}, err
	}
	if err := objectStore.Validate(ctx, choicepointObject, choicepointAuthority); err != nil {
		return Ruling{}, err
	}
	choicepoint, err := choice.ParseChoicepointRecord(choicepointObject.CanonicalBytes())
	if err != nil || choicepoint.Digest() != previous || choicepoint.Digest() != choicepointObject.Digest() {
		return Ruling{}, refuse("INVALID_STORED_RULING", "ruling predecessor is not the exact strict Choicepoint", err)
	}
	record, err := choice.ParseDecisionRecord(object.CanonicalBytes(), choicepoint)
	if err != nil || record.Digest() != object.Digest() || record.Digest() != head.CurrentDigest() {
		return Ruling{}, refuse("INVALID_STORED_RULING", "head, object, and strict DecisionRecord disagree", err)
	}
	if record.Action() == choice.ActionRefine {
		return Ruling{}, refuse(
			CodeRefineRequiresSuccessorStudy,
			"stored REFINE cannot mint current ruling authority without successor-study creation",
			nil,
		)
	}
	confirmationObject, confirmationAuthority, err := objectStore.Read(ctx, "FreshConfirmation", choicepoint.ConfirmationDigest())
	if err != nil {
		return Ruling{}, err
	}
	if err := objectStore.Validate(ctx, confirmationObject, confirmationAuthority); err != nil {
		return Ruling{}, err
	}
	confirmationRecord := choicepoint.ConfirmationRecord()
	if !confirmationRecord.Valid() || !bytes.Equal(confirmationObject.CanonicalBytes(), confirmationRecord.CanonicalBytes()) {
		return Ruling{}, refuse("INVALID_STORED_RULING", "Choicepoint embedded and durable confirmation bytes disagree", nil)
	}
	ruling := Ruling{
		head: head, object: object, authority: authority,
		choicepointObject: choicepointObject, choicepointAuthority: choicepointAuthority,
		confirmationObject: confirmationObject, confirmationAuthority: confirmationAuthority,
		record: record, choicepoint: choicepoint, seal: &rulingSeal{marker: 1},
	}
	if err := validateRuling(ctx, objectStore, ruling); err != nil {
		return Ruling{}, err
	}
	return ruling, nil
}

func validateRuling(ctx context.Context, objectStore *store.ObjectStore, ruling Ruling) error {
	if ruling.seal == nil || ruling.seal.marker != 1 || !ruling.record.Valid() || !ruling.choicepoint.Valid() ||
		ruling.object.Kind() != "DecisionRecord" || ruling.object.Digest() != ruling.record.Digest() ||
		ruling.choicepointObject.Kind() != "Choicepoint" || ruling.choicepointObject.Digest() != ruling.choicepoint.Digest() ||
		ruling.record.ChoicepointDigest() != ruling.choicepoint.Digest() {
		return refuse("INVALID_STORED_RULING", "ruling capability is incomplete", nil)
	}
	if err := objectStore.Validate(ctx, ruling.object, ruling.authority); err != nil {
		return err
	}
	if err := objectStore.Validate(ctx, ruling.choicepointObject, ruling.choicepointAuthority); err != nil {
		return err
	}
	if err := objectStore.Validate(ctx, ruling.confirmationObject, ruling.confirmationAuthority); err != nil {
		return err
	}
	current, err := objectStore.OpenHead(ctx, ruling.head.StudyID())
	if err != nil {
		return err
	}
	if !sameHead(current, ruling.head) || current.Stage() != store.StageRuling ||
		current.CurrentKind() != "DecisionRecord" || current.CurrentDigest() != ruling.record.Digest() {
		return refuse("RULING_NOT_CURRENT", "ruling capability was superseded", nil)
	}
	return nil
}

func (r Ruling) Record() choice.DecisionRecord {
	parsed, _ := choice.ParseDecisionRecord(r.record.CanonicalBytes(), r.choicepoint)
	return parsed
}

func (r Ruling) StudyID() store.StudyID { return r.head.StudyID() }

// PortableRulingPreparation is the current-head-bound seam P07B must consume.
// It contains no source bytes and cannot be reconstructed from a serialized
// DecisionRecord or from Choice's non-authorizing semantic inspection.
type PortableRulingPreparation struct {
	ruling     Ruling
	inspection choice.PortableRulingInspection
	seal       *portableRulingPreparationSeal
}

// PreparePortableRuling revalidates the exact current RULING head before
// issuing portable downstream preparation authority. Legacy current rulings
// retain their exact Choice refusal code through InspectPortableRuling.
func PreparePortableRuling(
	ctx context.Context,
	objectStore *store.ObjectStore,
	ruling Ruling,
) (PortableRulingPreparation, error) {
	if ctx == nil || objectStore == nil {
		return PortableRulingPreparation{}, refuse("INVALID_PORTABLE_RULING_PREPARATION", "store and context are required", nil)
	}
	if err := validateRuling(ctx, objectStore, ruling); err != nil {
		return PortableRulingPreparation{}, err
	}
	inspection, err := choice.InspectPortableRuling(ruling.record)
	if err != nil {
		return PortableRulingPreparation{}, err
	}
	return PortableRulingPreparation{
		ruling: ruling, inspection: inspection, seal: &portableRulingPreparationSeal{marker: 1},
	}, nil
}

// Valid reports sealed construction integrity, not durable store currentness.
func (p PortableRulingPreparation) Valid() bool {
	return p.seal != nil && p.seal.marker == 1 && p.inspection.Valid() &&
		p.ruling.record.Valid() && p.ruling.record.Digest() == p.inspection.DecisionDigest()
}

func (p PortableRulingPreparation) DecisionDigest() domain.Digest {
	return p.inspection.DecisionDigest()
}

func (p PortableRulingPreparation) ProfileDigest() domain.Digest {
	return p.inspection.ProfileDigest()
}

func (p PortableRulingPreparation) SelectedFields() []string {
	return p.inspection.SelectedFields()
}

// ValidatePortableRulingPreparation proves currentness only at this instant.
// Valid reports sealed construction integrity, and the getters expose inert
// identity; neither is durable current-head authority. P07B must revalidate
// inside its owner-controlled store transition/CAS and materialize files only
// after that publication succeeds, so no standalone check becomes a TOCTOU
// claim.
func ValidatePortableRulingPreparation(
	ctx context.Context,
	objectStore *store.ObjectStore,
	preparation PortableRulingPreparation,
) error {
	if ctx == nil || objectStore == nil || !preparation.Valid() {
		return refuse("INVALID_PORTABLE_RULING_PREPARATION", "sealed portable preparation is required", nil)
	}
	if err := validateRuling(ctx, objectStore, preparation.ruling); err != nil {
		return err
	}
	inspection, err := choice.InspectPortableRuling(preparation.ruling.record)
	if err != nil || inspection.DecisionDigest() != preparation.inspection.DecisionDigest() ||
		inspection.ProfileDigest() != preparation.inspection.ProfileDigest() ||
		!equalStrings(inspection.SelectedFields(), preparation.inspection.SelectedFields()) {
		return refuse("INVALID_PORTABLE_RULING_PREPARATION", "portable preparation no longer matches its exact ruling", err)
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameHead(left, right store.HeadToken) bool {
	leftPrevious, leftHasPrevious := left.PreviousObjectDigest()
	rightPrevious, rightHasPrevious := right.PreviousObjectDigest()
	return left.StudyID().String() == right.StudyID().String() && left.Revision() == right.Revision() &&
		left.Stage() == right.Stage() && left.CurrentKind() == right.CurrentKind() &&
		left.CurrentDigest() == right.CurrentDigest() && left.LineageRootDigest() == right.LineageRootDigest() &&
		left.HeadDigest() == right.HeadDigest() && leftHasPrevious == rightHasPrevious && leftPrevious == rightPrevious
}
