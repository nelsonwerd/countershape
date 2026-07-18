package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	choicepromotionauthority "github.com/nelsonwerd/countershape/internal/choice/promotion/authority"
	"github.com/nelsonwerd/countershape/internal/compare"
	confirmationauthority "github.com/nelsonwerd/countershape/internal/confirmation/authority"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeauthority "github.com/nelsonwerd/countershape/internal/emit/node/authority"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

const (
	studyHeadFilename = "head.json"
	studyLockFilename = ".head.lock"
	headTempPrefix    = ".head-"

	codeInvalidStudyID      = "INVALID_STUDY_ID"
	codeStudyAlreadyExists  = "STUDY_ALREADY_EXISTS"
	codeStudyAbsent         = "STUDY_ABSENT"
	codeHeadCorrupt         = "HEAD_CORRUPT"
	codeCASConflict         = "CAS_CONFLICT"
	codeIllegalLineage      = "ILLEGAL_LINEAGE"
	codeHeadUpdateFailed    = "HEAD_UPDATE_FAILED"
	codeHeadUpdateAmbiguous = "HEAD_UPDATE_AMBIGUOUS"
)

// StudyID is a domain-derived identifier. Raw labels are never joined into a
// filesystem path.
type StudyID struct {
	text   string
	digest domain.Digest
}

func NewStudyID(label string) (StudyID, error) {
	if label == "" || len(label) > 512 || !utf8.ValidString(label) || strings.TrimSpace(label) == "" {
		return StudyID{}, refuse(codeInvalidStudyID, "study label is empty or outside the bounded UTF-8 profile", nil)
	}
	for _, character := range label {
		if unicode.IsControl(character) {
			return StudyID{}, refuse(codeInvalidStudyID, "study label contains control text", nil)
		}
	}
	digest, _, err := canon.DigestTyped("StudyID", struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Label         string `json:"label"`
	}{domain.SchemaVersion, "StudyID", label})
	if err != nil {
		return StudyID{}, refuse(codeInvalidStudyID, "study identity cannot be derived", err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return StudyID{}, refuse(codeInvalidStudyID, "study digest is invalid", err)
	}
	hex := strings.TrimPrefix(parsed.String(), "sha256:")
	return StudyID{text: "study:" + hex, digest: parsed}, nil
}

func (id StudyID) Valid() bool {
	return id.digest.Valid() && id.text == "study:"+strings.TrimPrefix(id.digest.String(), "sha256:")
}
func (id StudyID) String() string        { return id.text }
func (id StudyID) Digest() domain.Digest { return id.digest }

type LineageStage string

const (
	StageSourcePlan       LineageStage = "SOURCE_PLAN"
	StageBaseline         LineageStage = "BASELINE"
	StageDivergence       LineageStage = "DIVERGENCE"
	StageReduction        LineageStage = "REDUCTION"
	StageConfirmation     LineageStage = "CONFIRMATION"
	StageChoicepointReady LineageStage = "CHOICEPOINT_READY"
	StageRuling           LineageStage = "RULING"
	StageResidue          LineageStage = "RESIDUE"
	StagePartial          LineageStage = "PARTIAL"
	StageCancelled        LineageStage = "CANCELLED"
)

func (stage LineageStage) valid() bool {
	switch stage {
	case StageSourcePlan, StageBaseline, StageDivergence, StageReduction, StageConfirmation,
		StageChoicepointReady, StageRuling, StageResidue:
		return true
	default:
		return false
	}
}

type headIdentity struct {
	SchemaVersion        string `json:"schema_version"`
	Kind                 string `json:"kind"`
	StudyID              string `json:"study_id"`
	Revision             int64  `json:"revision"`
	Stage                string `json:"stage"`
	CurrentKind          string `json:"current_kind"`
	CurrentDigest        string `json:"current_digest"`
	LineageRootDigest    string `json:"lineage_root_digest"`
	PreviousHeadDigest   string `json:"previous_head_digest"`
	PreviousObjectDigest string `json:"previous_object_digest"`
}

type storedHead struct {
	identity  headIdentity
	digest    domain.Digest
	canonical []byte
}

// HeadToken is the opaque exact expected value for one compare-and-swap. It is
// intentionally not serializable. A restarted process reopens a fresh token.
type HeadToken struct {
	storeInstance  *objectStoreInstance
	study          StudyID
	revision       int64
	stage          LineageStage
	currentKind    string
	currentDigest  domain.Digest
	previousHead   domain.Digest
	previousObject domain.Digest
	lineageRoot    domain.Digest
	headDigest     domain.Digest
}

func (h HeadToken) StudyID() StudyID             { return h.study }
func (h HeadToken) Revision() int64              { return h.revision }
func (h HeadToken) Stage() LineageStage          { return h.stage }
func (h HeadToken) CurrentKind() string          { return h.currentKind }
func (h HeadToken) CurrentDigest() domain.Digest { return h.currentDigest }
func (h HeadToken) PreviousHeadDigest() (domain.Digest, bool) {
	return h.previousHead, h.revision > 1 && h.previousHead.Valid()
}
func (h HeadToken) PreviousObjectDigest() (domain.Digest, bool) {
	return h.previousObject, h.revision > 1 && h.previousObject.Valid()
}
func (h HeadToken) LineageRootDigest() domain.Digest { return h.lineageRoot }
func (h HeadToken) HeadDigest() domain.Digest        { return h.headDigest }

func (h HeadToken) validFor(s *ObjectStore) bool {
	return s != nil && s.instance != nil && h.storeInstance == s.instance && h.study.Valid() && h.revision >= 1 &&
		h.stage.valid() && h.currentKind != "" && h.currentDigest.Valid() && h.lineageRoot.Valid() && h.headDigest.Valid() &&
		((h.revision == 1 && !h.previousHead.Valid() && !h.previousObject.Valid()) ||
			(h.revision > 1 && h.previousHead.Valid() && h.previousObject.Valid()))
}

// CreateStudy admits only a live, exactly reconstructible WorldPlan. The
// generic SemanticObject carrier is intentionally not an exported head API.
func (s *ObjectStore) CreateStudy(
	ctx context.Context,
	study StudyID,
	plan domain.WorldPlan,
) (HeadToken, error) {
	exact := plan.CanonicalBytes()
	parsed, err := domain.ParseWorldPlan(exact, plan.ProjectionDefinitionBinding())
	if err != nil || parsed.Digest() != plan.Digest() || !bytes.Equal(parsed.CanonicalBytes(), exact) {
		return HeadToken{}, refuse(codeIllegalLineage, "initial study requires a live exact WorldPlan", err)
	}
	object, err := NewSemanticObject("WorldPlan", plan.Digest(), exact)
	if err != nil {
		return HeadToken{}, err
	}
	return s.createStudyObject(ctx, study, object)
}

func (s *ObjectStore) createStudyObject(
	ctx context.Context,
	study StudyID,
	initial SemanticObject,
) (HeadToken, error) {
	if s == nil || s.instance == nil {
		return HeadToken{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeHeadUpdateFailed); err != nil {
		return HeadToken{}, err
	}
	if !study.Valid() || !initial.Valid() || !kindAllowedForStage(StageSourcePlan, initial.kind) {
		return HeadToken{}, refuse(codeIllegalLineage, "initial study object must be a typed source plan", nil)
	}
	if err := s.assertReady(); err != nil {
		return HeadToken{}, err
	}
	studyPath, err := s.ensureStudyPath(study)
	if err != nil {
		return HeadToken{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(studyPath, studyLockFilename), true)
	if err != nil {
		return HeadToken{}, refuse(codeHeadUpdateFailed, "study lock could not be acquired", err)
	}
	defer lock.release()
	headPath := filepath.Join(studyPath, studyHeadFilename)
	if _, err := os.Lstat(headPath); err == nil {
		return HeadToken{}, refuse(codeStudyAlreadyExists, "study already has a head", nil)
	} else if !errors.Is(err, os.ErrNotExist) {
		return HeadToken{}, refuse(codeHeadCorrupt, "study head path cannot be classified", err)
	}
	if err := storeContextRefusal(ctx, codeHeadUpdateFailed); err != nil {
		return HeadToken{}, err
	}
	objectAuthority, err := s.publishLocked(ctx, initial)
	if err != nil {
		return HeadToken{}, err
	}
	if objectAuthority.storeInstance != s.instance {
		return HeadToken{}, refuse(codeHeadUpdateFailed, "initial object authority is foreign", nil)
	}
	head, err := newStoredHead(headIdentity{
		StudyID: study.String(), Revision: 1, Stage: string(StageSourcePlan), CurrentKind: initial.kind,
		CurrentDigest: initial.digest.String(), LineageRootDigest: initial.digest.String(),
		PreviousHeadDigest: "", PreviousObjectDigest: "",
	})
	if err != nil {
		return HeadToken{}, err
	}
	if err := replaceHead(studyPath, headPath, head, false); err != nil {
		return HeadToken{}, err
	}
	verified, err := s.reopenHeadAndObject(study, studyPath)
	if err != nil || verified.digest != head.digest {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "created head did not reopen exactly", err)
	}
	return s.token(study, verified), nil
}

func (s *ObjectStore) OpenHead(ctx context.Context, study StudyID) (HeadToken, error) {
	if s == nil || s.instance == nil {
		return HeadToken{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeHeadCorrupt); err != nil {
		return HeadToken{}, err
	}
	if err := s.assertReady(); err != nil {
		return HeadToken{}, err
	}
	studyPath, err := s.existingStudyPath(study)
	if err != nil {
		return HeadToken{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(studyPath, studyLockFilename), false)
	if err != nil {
		return HeadToken{}, refuse(codeHeadCorrupt, "study lock could not be acquired", err)
	}
	defer lock.release()
	head, err := s.reopenHeadAndObject(study, studyPath)
	if err != nil {
		return HeadToken{}, err
	}
	return s.token(study, head), nil
}

// AdvanceBaseline admits the exact strict observed map rather than a generic
// kind-labeled object.
func (s *ObjectStore) AdvanceBaseline(
	ctx context.Context,
	expected HeadToken,
	outcome compare.CandidateOutcomeMap,
) (HeadToken, error) {
	exact := outcome.CanonicalBytes()
	parsed, err := compare.ParseCandidateOutcomeMap(exact)
	if err != nil || expected.Stage() != StageSourcePlan || outcome.PlanDigest() != expected.LineageRootDigest() ||
		parsed.ArtifactDigest() != outcome.ArtifactDigest() || !bytes.Equal(parsed.CanonicalBytes(), exact) {
		return HeadToken{}, refuse(codeIllegalLineage, "baseline requires an exact CandidateOutcomeMap", err)
	}
	object, err := NewSemanticObject("CandidateOutcomeMap", outcome.ArtifactDigest().DomainDigest(), exact)
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageBaseline, object)
}

// AdvanceDivergence requires the opaque divergent-baseline authority, so a
// merely serialized or one-fingerprint map cannot claim this head stage.
func (s *ObjectStore) AdvanceDivergence(
	ctx context.Context,
	expected HeadToken,
	baseline compare.DivergentBaseline,
) (HeadToken, error) {
	if !baseline.Valid() {
		return HeadToken{}, refuse(codeIllegalLineage, "divergence requires an exact divergent baseline", nil)
	}
	outcome := baseline.OutcomeMap()
	if expected.Stage() != StageBaseline || outcome.PlanDigest() != expected.LineageRootDigest() {
		return HeadToken{}, refuse(codeIllegalLineage, "divergence does not bind the current plan lineage", nil)
	}
	currentObject, currentAuthority, err := s.Read(ctx, "CandidateOutcomeMap", expected.CurrentDigest())
	if err != nil {
		return HeadToken{}, err
	}
	if err := s.Validate(ctx, currentObject, currentAuthority); err != nil {
		return HeadToken{}, err
	}
	currentMap, err := compare.ParseCandidateOutcomeMap(currentObject.CanonicalBytes())
	if err != nil || compare.AssessPreservation(currentMap, outcome).Relation() != compare.PreservationEqual {
		return HeadToken{}, refuse(codeIllegalLineage, "divergence map is not exact-map comparable with the current baseline", err)
	}
	object, err := NewSemanticObject("CandidateOutcomeMap", outcome.ArtifactDigest().DomainDigest(), outcome.CanonicalBytes())
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageDivergence, object)
}

// AdvanceReduction accepts a live ReductionRun. Strict restart records remain
// inspectable but cannot advance a study by themselves.
func (s *ObjectStore) AdvanceReduction(
	ctx context.Context,
	expected HeadToken,
	run reduce.ReductionRun,
) (HeadToken, error) {
	if !run.Valid() || expected.Stage() != StageDivergence ||
		expected.LineageRootDigest() != run.BaselinePlanDigest() ||
		expected.CurrentDigest() != run.Baseline().OutcomeMap().ArtifactDigest().DomainDigest() {
		return HeadToken{}, refuse(codeIllegalLineage, "reduction requires a live exact ReductionRun", nil)
	}
	object, err := NewSemanticObject("ReductionRun", run.Digest(), run.CanonicalBytes())
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageReduction, object)
}

// AdvanceConfirmation accepts only the publication capability issued with a
// live confirmation.Draft and binds it to the exact current reduction object.
func (s *ObjectStore) AdvanceConfirmation(
	ctx context.Context,
	expected HeadToken,
	authority confirmationauthority.Publication,
) (HeadToken, error) {
	if !authority.Valid() || expected.Stage() != StageReduction || expected.CurrentDigest() != authority.Predecessor() {
		return HeadToken{}, refuse(codeIllegalLineage, "confirmation publication authority does not bind the current reduction", nil)
	}
	object, err := NewSemanticObject("FreshConfirmation", authority.Digest(), authority.CanonicalBytes())
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageConfirmation, object)
}

// AdvanceChoicepoint accepts only a capability issued by the store-bound
// promotion service after it validates the current StoredConfirmation.
func (s *ObjectStore) AdvanceChoicepoint(
	ctx context.Context,
	expected HeadToken,
	authority choicepromotionauthority.Choicepoint,
) (HeadToken, error) {
	if !authority.Valid() || expected.Stage() != StageConfirmation || expected.CurrentDigest() != authority.Predecessor() {
		return HeadToken{}, refuse(codeIllegalLineage, "Choicepoint publication authority does not bind the current confirmation", nil)
	}
	object, err := NewSemanticObject("Choicepoint", authority.Digest(), authority.CanonicalBytes())
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageChoicepointReady, object)
}

// AdvanceRuling accepts only a promotion-issued capability. REFINE has no U6
// successor-study authority and is refused here as well as by promotion.
func (s *ObjectStore) AdvanceRuling(
	ctx context.Context,
	expected HeadToken,
	authority choicepromotionauthority.Ruling,
) (HeadToken, error) {
	if !authority.Valid() || expected.Stage() != StageChoicepointReady || expected.CurrentDigest() != authority.Predecessor() {
		return HeadToken{}, refuse(codeIllegalLineage, "ruling publication authority does not bind the current Choicepoint", nil)
	}
	switch authority.Action() {
	case "ALLOW_OBSERVED", "CUSTOM_EXPECTATION", "REJECT_ALL", "DEFER":
	default:
		return HeadToken{}, refuse(codeIllegalLineage, "ruling action has no U6 durable transition authority", nil)
	}
	object, err := NewSemanticObject("DecisionRecord", authority.Digest(), authority.CanonicalBytes())
	if err != nil {
		return HeadToken{}, err
	}
	return s.advanceHead(ctx, expected, StageRuling, object)
}

// AdvanceResidue accepts only a node-emitter-issued capability over one exact
// prepared ContractBundle. The shared raw owner compares every expected-head
// field under transition exclusion before it creates a successor object or
// temporary file.
func (s *ObjectStore) AdvanceResidue(
	ctx context.Context,
	expected HeadToken,
	authority nodeauthority.Publication,
) (HeadToken, error) {
	choicepoint, hasChoicepoint := expected.PreviousObjectDigest()
	if !authority.Valid() || !expected.validFor(s) || expected.Stage() != StageRuling ||
		expected.CurrentKind() != "DecisionRecord" || expected.StudyID().String() != authority.StudyID() ||
		expected.HeadDigest() != authority.ExpectedHeadDigest() || expected.CurrentDigest() != authority.Predecessor() ||
		!hasChoicepoint || choicepoint != authority.Choicepoint() ||
		expected.LineageRootDigest() != authority.LineageRoot() {
		return HeadToken{}, refuse(codeIllegalLineage, "residue publication authority does not bind the current ruling", nil)
	}
	object, err := NewSemanticObject("ContractBundle", authority.Digest(), authority.CanonicalBytes())
	if err != nil {
		return HeadToken{}, refuse(codeIllegalLineage, "residue publication authority does not contain one exact ContractBundle", err)
	}
	return s.advanceHead(ctx, expected, StageResidue, object)
}

// ConfirmResiduePublication turns a visible exact terminal successor into a
// durable local fact only after taking the study exclusion, syncing its head
// directory, and reopening the exact head and referenced object. It performs
// no mutation and is the sole reconciliation path for post-head ambiguity.
func (s *ObjectStore) ConfirmResiduePublication(
	ctx context.Context,
	expected HeadToken,
	authority nodeauthority.Publication,
) (HeadToken, error) {
	previousHead, hasPreviousHead := expected.PreviousHeadDigest()
	previousObject, hasPreviousObject := expected.PreviousObjectDigest()
	if s == nil || s.instance == nil || !expected.validFor(s) || !authority.Valid() ||
		expected.Stage() != StageResidue || expected.CurrentKind() != "ContractBundle" ||
		expected.StudyID().String() != authority.StudyID() || expected.CurrentDigest() != authority.Digest() ||
		!hasPreviousHead || previousHead != authority.ExpectedHeadDigest() ||
		!hasPreviousObject || previousObject != authority.Predecessor() ||
		expected.LineageRootDigest() != authority.LineageRoot() {
		return HeadToken{}, refuse(codeIllegalLineage, "terminal residue does not bind the exact publication authority", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeHeadUpdateAmbiguous); err != nil {
		return HeadToken{}, err
	}
	if err := s.assertReady(); err != nil {
		return HeadToken{}, refuse(
			codeHeadUpdateAmbiguous, "object-store identity changed before residue reconciliation", err,
		)
	}
	studyPath, err := s.existingStudyPath(expected.study)
	if err != nil {
		return HeadToken{}, refuse(
			codeHeadUpdateAmbiguous, "study directory changed before residue reconciliation", err,
		)
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(studyPath, studyLockFilename), false)
	if err != nil {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "study lock could not be acquired for residue reconciliation", err)
	}
	defer lock.release()
	current, err := s.reopenHeadAndObject(expected.study, studyPath)
	if err != nil || !sameHeadToken(s.token(expected.study, current), expected) ||
		current.identity.PreviousHeadDigest != authority.ExpectedHeadDigest().String() {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "visible residue differs before durability reconciliation", err)
	}
	if err := syncDirectory(studyPath); err != nil {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "terminal residue study-directory sync failed", err)
	}
	verified, err := s.reopenHeadAndObject(expected.study, studyPath)
	if err != nil {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "terminal residue did not reopen after durability sync", err)
	}
	confirmed := s.token(expected.study, verified)
	if !sameHeadToken(confirmed, expected) || verified.identity.PreviousHeadDigest != authority.ExpectedHeadDigest().String() {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "terminal residue changed during durability reconciliation", nil)
	}
	return confirmed, nil
}

func (s *ObjectStore) advanceHead(
	ctx context.Context,
	expected HeadToken,
	nextStage LineageStage,
	next SemanticObject,
) (HeadToken, error) {
	if s == nil || s.instance == nil {
		return HeadToken{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeHeadUpdateFailed); err != nil {
		return HeadToken{}, err
	}
	if err := s.assertReady(); err != nil {
		return HeadToken{}, err
	}
	if !expected.validFor(s) || !next.Valid() || !nextStage.valid() {
		return HeadToken{}, refuse(codeIllegalLineage, "advance authority or next object is invalid", nil)
	}
	studyPath, err := s.existingStudyPath(expected.study)
	if err != nil {
		return HeadToken{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(studyPath, studyLockFilename), false)
	if err != nil {
		return HeadToken{}, refuse(codeHeadUpdateFailed, "study lock could not be acquired", err)
	}
	defer lock.release()
	current, err := s.reopenHeadAndObject(expected.study, studyPath)
	if err != nil {
		return HeadToken{}, err
	}
	currentToken := s.token(expected.study, current)
	// MUTANT_U6_CAS_COMPARE_AFTER_PUBLISH: this full comparison must precede
	// every successor object or temporary-file creation.
	if !sameHeadToken(currentToken, expected) {
		return HeadToken{}, refuse(codeCASConflict, "expected revision/digest no longer names the current head", nil)
	}
	if current.identity.Revision >= canon.MaxSafeInteger {
		return HeadToken{}, refuse(codeIllegalLineage, "head revision would exceed the exact integer range", nil)
	}
	if !permittedTransition(LineageStage(current.identity.Stage), nextStage) ||
		!kindAllowedForStage(nextStage, next.kind) || next.digest == currentToken.currentDigest {
		return HeadToken{}, refuse(codeIllegalLineage, "next stage, kind, or object is not a legal successor", nil)
	}
	if err := storeContextRefusal(ctx, codeHeadUpdateFailed); err != nil {
		return HeadToken{}, err
	}
	if _, err := s.publishLocked(ctx, next); err != nil {
		return HeadToken{}, err
	}
	head, err := newStoredHead(headIdentity{
		StudyID: expected.study.String(), Revision: current.identity.Revision + 1, Stage: string(nextStage),
		CurrentKind: next.kind, CurrentDigest: next.digest.String(), LineageRootDigest: current.identity.LineageRootDigest,
		PreviousHeadDigest: current.digest.String(), PreviousObjectDigest: current.identity.CurrentDigest,
	})
	if err != nil {
		return HeadToken{}, err
	}
	headPath := filepath.Join(studyPath, studyHeadFilename)
	if err := replaceHead(studyPath, headPath, head, true); err != nil {
		return HeadToken{}, err
	}
	verified, err := s.reopenHeadAndObject(expected.study, studyPath)
	if err != nil || verified.digest != head.digest {
		return HeadToken{}, refuse(codeHeadUpdateAmbiguous, "advanced head did not reopen exactly", err)
	}
	return s.token(expected.study, verified), nil
}

func (s *ObjectStore) token(study StudyID, head storedHead) HeadToken {
	current, _ := domain.ParseDigest(head.identity.CurrentDigest)
	root, _ := domain.ParseDigest(head.identity.LineageRootDigest)
	previousHead, _ := domain.ParseDigest(head.identity.PreviousHeadDigest)
	previous, _ := domain.ParseDigest(head.identity.PreviousObjectDigest)
	return HeadToken{
		storeInstance: s.instance, study: study, revision: head.identity.Revision,
		stage: LineageStage(head.identity.Stage), currentKind: head.identity.CurrentKind,
		currentDigest: current, previousHead: previousHead, previousObject: previous,
		lineageRoot: root, headDigest: head.digest,
	}
}

func sameHeadToken(left, right HeadToken) bool {
	return left.study.text == right.study.text && left.revision == right.revision && left.stage == right.stage &&
		left.currentKind == right.currentKind && left.currentDigest == right.currentDigest &&
		left.previousHead == right.previousHead && left.previousObject == right.previousObject &&
		left.lineageRoot == right.lineageRoot && left.headDigest == right.headDigest
}

func permittedTransition(current, next LineageStage) bool {
	switch current {
	case StageSourcePlan:
		return next == StageBaseline
	case StageBaseline:
		return next == StageDivergence
	case StageDivergence:
		return next == StageReduction
	case StageReduction:
		return next == StageConfirmation
	case StageConfirmation:
		return next == StageChoicepointReady
	case StageChoicepointReady:
		return next == StageRuling
	case StageRuling:
		return next == StageResidue
	default:
		return false
	}
}

func stageForRevision(revision int64) (LineageStage, bool) {
	stages := [...]LineageStage{
		StageSourcePlan,
		StageBaseline,
		StageDivergence,
		StageReduction,
		StageConfirmation,
		StageChoicepointReady,
		StageRuling,
		StageResidue,
	}
	if revision < 1 || revision > int64(len(stages)) {
		return "", false
	}
	return stages[revision-1], true
}

func kindAllowedForStage(stage LineageStage, kind string) bool {
	switch stage {
	case StageSourcePlan:
		return kind == "WorldPlan"
	case StageBaseline, StageDivergence:
		return kind == "CandidateOutcomeMap"
	case StageReduction:
		return kind == "ReductionRun" || kind == "ReductionGrade"
	case StageConfirmation:
		return kind == "FreshConfirmation"
	case StageChoicepointReady:
		return kind == "Choicepoint"
	case StageRuling:
		return kind == "DecisionRecord"
	case StageResidue:
		return kind == "ContractBundle"
	case StagePartial, StageCancelled:
		return kind == "PartialAttempt" || kind == "CancelledAttempt"
	default:
		return false
	}
}

func newStoredHead(input headIdentity) (storedHead, error) {
	input.SchemaVersion = domain.SchemaVersion
	input.Kind = "StudyHead"
	if err := validateHeadIdentity(input); err != nil {
		return storedHead{}, err
	}
	digest, canonicalBytes, err := canon.DigestTyped("StudyHead", input)
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head cannot be canonicalized", err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head digest is invalid", err)
	}
	return storedHead{identity: input, digest: parsed, canonical: canonicalBytes}, nil
}

func parseStoredHead(exact []byte) (storedHead, error) {
	if len(exact) == 0 || len(exact) > canon.MaxInputBytes {
		return storedHead{}, refuse(codeHeadCorrupt, "head is empty or oversized", nil)
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head is not strict canonical JSON", err)
	}
	canonicalBytes, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonicalBytes, exact) {
		return storedHead{}, refuse(codeHeadCorrupt, "head bytes are not canonical", err)
	}
	members, object := value.Members()
	if !object || len(members) != 10 {
		return storedHead{}, refuse(codeHeadCorrupt, "head has an unknown or missing member", nil)
	}
	var identity headIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head shape cannot be decoded", err)
	}
	if err := validateHeadIdentity(identity); err != nil {
		return storedHead{}, err
	}
	rebuilt, err := newStoredHead(identity)
	if err != nil || !bytes.Equal(rebuilt.canonical, exact) {
		return storedHead{}, refuse(codeHeadCorrupt, "head failed exact reconstruction", err)
	}
	return rebuilt, nil
}

func validateHeadIdentity(identity headIdentity) error {
	expectedStage, expectedRevision := stageForRevision(identity.Revision)
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "StudyHead" || identity.Revision < 1 ||
		identity.Revision > canon.MaxSafeInteger || !expectedRevision || identity.Stage != string(expectedStage) ||
		!LineageStage(identity.Stage).valid() || identity.CurrentKind == "" {
		return refuse(codeHeadCorrupt, "head scalar identity is invalid", nil)
	}
	study, err := parseStudyID(identity.StudyID)
	if err != nil || !study.Valid() {
		return refuse(codeHeadCorrupt, "head study identity is invalid", err)
	}
	current, currentErr := domain.ParseDigest(identity.CurrentDigest)
	root, rootErr := domain.ParseDigest(identity.LineageRootDigest)
	if currentErr != nil || rootErr != nil || !current.Valid() || !root.Valid() ||
		!kindAllowedForStage(LineageStage(identity.Stage), identity.CurrentKind) {
		return refuse(codeHeadCorrupt, "head object reference is invalid", errors.Join(currentErr, rootErr))
	}
	if identity.Revision == 1 {
		if identity.Stage != string(StageSourcePlan) || identity.CurrentDigest != identity.LineageRootDigest ||
			identity.PreviousHeadDigest != "" || identity.PreviousObjectDigest != "" {
			return refuse(codeHeadCorrupt, "initial head lineage is invalid", nil)
		}
	} else {
		// Previous digests are exact identity commitments used by the live CAS.
		// Replaced head files are not archived by the generic store, so restart
		// validation cannot claim dereferenceable historical ancestry.
		previousHead, headErr := domain.ParseDigest(identity.PreviousHeadDigest)
		previousObject, objectErr := domain.ParseDigest(identity.PreviousObjectDigest)
		if headErr != nil || objectErr != nil || !previousHead.Valid() || !previousObject.Valid() || previousObject == current {
			return refuse(codeHeadCorrupt, "successor predecessor identity is invalid", errors.Join(headErr, objectErr))
		}
	}
	return nil
}

func parseStudyID(raw string) (StudyID, error) {
	if len(raw) != len("study:")+64 || !strings.HasPrefix(raw, "study:") {
		return StudyID{}, refuse(codeInvalidStudyID, "study id is not canonical", nil)
	}
	digest, err := domain.ParseDigest("sha256:" + strings.TrimPrefix(raw, "study:"))
	if err != nil {
		return StudyID{}, err
	}
	return StudyID{text: raw, digest: digest}, nil
}

func (s *ObjectStore) ensureStudyPath(study StudyID) (string, error) {
	if !study.Valid() {
		return "", refuse(codeInvalidStudyID, "study identity is invalid", nil)
	}
	hex := strings.TrimPrefix(study.String(), "study:")
	if err := rejectCaseAlias(s.studies, hex); err != nil {
		return "", refuse(codeHeadCorrupt, "study namespace has a case alias", err)
	}
	path := filepath.Join(s.studies, hex)
	info, created, err := ensurePrivateDirectory(path)
	if err != nil {
		return "", refuse(codeHeadUpdateFailed, "study directory is invalid", err)
	}
	if created {
		if err := syncDirectory(s.studies); err != nil {
			return "", refuse(codeHeadUpdateFailed, "study-root sync failed", err)
		}
	}
	if retained, present := s.studyInfos[path]; present && !os.SameFile(retained, info) {
		return "", refuse(codeHeadCorrupt, "study directory changed identity", nil)
	}
	s.studyInfos[path] = info
	return path, nil
}

func (s *ObjectStore) existingStudyPath(study StudyID) (string, error) {
	if !study.Valid() {
		return "", refuse(codeInvalidStudyID, "study identity is invalid", nil)
	}
	hex := strings.TrimPrefix(study.String(), "study:")
	if err := rejectCaseAlias(s.studies, hex); err != nil {
		return "", refuse(codeHeadCorrupt, "study namespace has a case alias", err)
	}
	path := filepath.Join(s.studies, hex)
	info, err := exactPrivateDirectoryInfo(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", refuse(codeStudyAbsent, "study has no durable directory", err)
	}
	if err != nil {
		return "", refuse(codeHeadCorrupt, "study directory is invalid", err)
	}
	if retained, present := s.studyInfos[path]; present && !os.SameFile(retained, info) {
		return "", refuse(codeHeadCorrupt, "study directory changed identity", nil)
	}
	s.studyInfos[path] = info
	return path, nil
}

func replaceHead(studyPath, headPath string, head storedHead, replacing bool) error {
	if replacing {
		if _, err := exactObjectInfo(headPath, fileSize(headPath)); err != nil {
			return refuse(codeHeadCorrupt, "current head is not a real private file", err)
		}
	} else if _, err := os.Lstat(headPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return refuse(codeHeadCorrupt, "initial head destination is not absent", err)
	}
	temporary, err := writeSyncedTemporary(studyPath, headTempPrefix, head.canonical)
	if err != nil {
		return refuse(codeHeadUpdateFailed, "head temporary write failed", err)
	}
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	verified, err := readHeadFile(temporary)
	if err != nil || verified.digest != head.digest {
		return refuse(codeHeadUpdateFailed, "head temporary verification failed", err)
	}
	if err := renameHeadNoFollow(temporary, headPath); err != nil {
		return refuse(codeHeadUpdateFailed, "atomic head replacement failed", err)
	}
	removeTemporary = false
	if err := syncDirectory(studyPath); err != nil {
		return refuse(codeHeadUpdateAmbiguous, "head renamed but directory sync failed", err)
	}
	reopened, err := readHeadFile(headPath)
	if err != nil || reopened.digest != head.digest {
		return refuse(codeHeadUpdateAmbiguous, "head renamed but exact reopen failed", err)
	}
	return nil
}

func fileSize(path string) int64 {
	info, err := os.Lstat(path)
	if err != nil {
		return -1
	}
	return info.Size()
}

func readHeadFile(path string) (storedHead, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() < 1 || info.Size() > int64(canon.MaxInputBytes) {
		return storedHead{}, refuse(codeHeadCorrupt, "head path facts are invalid", err)
	}
	handle, err := os.Open(path)
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head open failed", err)
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !os.SameFile(info, opened) || opened.Size() != info.Size() {
		_ = handle.Close()
		return storedHead{}, refuse(codeHeadCorrupt, "opened head disagrees with path facts", statErr)
	}
	body, readErr := io.ReadAll(io.LimitReader(handle, int64(canon.MaxInputBytes)+1))
	closeErr := handle.Close()
	after, afterErr := os.Lstat(path)
	if readErr != nil || closeErr != nil || afterErr != nil || !os.SameFile(opened, after) || int64(len(body)) != info.Size() {
		return storedHead{}, refuse(codeHeadCorrupt, "head changed during bounded read", errors.Join(readErr, closeErr, afterErr))
	}
	return parseStoredHead(body)
}

func (s *ObjectStore) reopenHeadAndObject(study StudyID, studyPath string) (storedHead, error) {
	if err := s.assertReady(); err != nil {
		return storedHead{}, err
	}
	head, err := readHeadFile(filepath.Join(studyPath, studyHeadFilename))
	if err != nil {
		return storedHead{}, err
	}
	if head.identity.StudyID != study.String() {
		return storedHead{}, refuse(codeHeadCorrupt, "head names a different study", nil)
	}
	digest, err := domain.ParseDigest(head.identity.CurrentDigest)
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head current digest is invalid", err)
	}
	rootDigest, err := domain.ParseDigest(head.identity.LineageRootDigest)
	if err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head lineage-root digest is invalid", err)
	}
	if _, err := s.readObjectReference("WorldPlan", rootDigest); err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head lineage-root WorldPlan is absent or corrupt", err)
	}
	if _, err := s.readObjectReference(head.identity.CurrentKind, digest); err != nil {
		return storedHead{}, refuse(codeHeadCorrupt, "head target object is absent or corrupt", err)
	}
	if err := s.assertReady(); err != nil {
		return storedHead{}, err
	}
	return head, nil
}

func (s *ObjectStore) readObjectReference(kind string, digest domain.Digest) (SemanticObject, error) {
	path, _, err := s.existingObjectPath(digest)
	if err != nil {
		return SemanticObject{}, err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() < 1 || info.Size() > int64(canon.MaxInputBytes) {
		return SemanticObject{}, refuse(codeObjectRefused, "referenced object path facts are invalid", err)
	}
	handle, err := os.Open(path)
	if err != nil {
		return SemanticObject{}, refuse(codeObjectRefused, "referenced object open failed", err)
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !os.SameFile(info, opened) {
		_ = handle.Close()
		return SemanticObject{}, refuse(codeObjectRefused, "referenced object changed during open", statErr)
	}
	body, readErr := io.ReadAll(io.LimitReader(handle, int64(canon.MaxInputBytes)+1))
	closeErr := handle.Close()
	after, afterErr := os.Lstat(path)
	if readErr != nil || closeErr != nil || afterErr != nil || !os.SameFile(opened, after) ||
		int64(len(body)) != info.Size() || after.Size() != info.Size() {
		return SemanticObject{}, refuse(codeObjectRefused, "referenced object bounded read failed", errors.Join(readErr, closeErr, afterErr))
	}
	object, err := NewSemanticObject(kind, digest, body)
	if err != nil {
		return SemanticObject{}, err
	}
	return object, nil
}

func (h storedHead) String() string {
	return fmt.Sprintf("%s@%s", h.identity.StudyID, strconv.FormatInt(h.identity.Revision, 10))
}
