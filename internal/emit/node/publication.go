package node

import (
	"bytes"
	"context"
	"errors"

	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/domain"
	internalpublication "github.com/nelsonwerd/countershape/internal/emit/node/internal/publication"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidResiduePublication   = "INVALID_RESIDUE_PUBLICATION"
	CodeResidueConflict             = "RESIDUE_CONFLICT"
	CodeResiduePublicationFailed    = "RESIDUE_PUBLICATION_FAILED"
	CodeResiduePublicationAmbiguous = "RESIDUE_PUBLICATION_AMBIGUOUS"
	CodeInvalidResidue              = "INVALID_RESIDUE"
)

type PublicationDisposition string

const (
	PublicationCreated        PublicationDisposition = "PUBLISHED"
	PublicationAlreadyCurrent PublicationDisposition = "ALREADY_CURRENT"
)

type residueSeal struct{}

var sealedResidue = &residueSeal{}

// Residue is a store-instance-bound, fully reopened terminal capability. It
// retains the exact terminal head, bundle object authority, and reconstructed
// predecessor chain privately. Valid proves construction integrity only.
type Residue struct {
	objectStore *store.ObjectStore
	head        store.HeadToken
	object      store.SemanticObject
	authority   store.ObjectAuthority
	bundle      model.ContractBundle
	predecessor promotion.ResiduePredecessorSnapshot
	seal        *residueSeal
}

type PublicationResult struct {
	Residue     Residue
	Disposition PublicationDisposition
}

// PublishPrepared revalidates the exact preparation privately retained by one
// PreparedBundle, then asks the store for the sole typed terminal transition.
// It performs no filesystem output materialization.
func PublishPrepared(
	ctx context.Context,
	objectStore *store.ObjectStore,
	prepared PreparedBundle,
) (PublicationResult, error) {
	if ctx == nil || objectStore == nil || !prepared.Valid() {
		return PublicationResult{}, refuse(CodeInvalidResiduePublication, "context, store, and sealed PreparedBundle are required", nil)
	}
	retained := prepared.prepared.preparation
	study := retained.StudyID()
	bundle := prepared.Bundle()
	if !study.Valid() || !bundle.Valid() || retained.DecisionDigest() != bundle.DecisionRecordDigest() ||
		retained.ProfileDigest() != bundle.PortableSource().Profile().Digest() {
		return PublicationResult{}, refuse(CodeInvalidResiduePublication, "prepared publication authority does not rejoin its exact bundle", nil)
	}
	if err := promotion.ValidatePortableRulingPredecessor(ctx, objectStore, retained); err != nil {
		return PublicationResult{}, err
	}

	current, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return PublicationResult{}, err
	}
	if current.Stage() == store.StageResidue {
		return reopenPublishedResult(ctx, objectStore, study, bundle, PublicationAlreadyCurrent, nil)
	}
	if current.Stage() != store.StageRuling || current.CurrentKind() != "DecisionRecord" ||
		current.CurrentDigest() != retained.DecisionDigest() {
		return PublicationResult{}, refuse(CodeResidueConflict, "study head is not the exact retained ruling", nil)
	}
	if err := promotion.ValidatePortableRulingPreparation(ctx, objectStore, retained); err != nil {
		return reopenPublishedResult(ctx, objectStore, study, bundle, PublicationAlreadyCurrent, err)
	}
	expected, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return PublicationResult{}, err
	}
	if expected.Stage() != store.StageRuling || expected.CurrentKind() != "DecisionRecord" ||
		expected.CurrentDigest() != retained.DecisionDigest() {
		return reopenPublishedResult(ctx, objectStore, study, bundle, PublicationAlreadyCurrent,
			refuse(CodeResiduePublicationFailed, "ruling changed before terminal publication", nil))
	}
	publicationAuthority, err := internalpublication.Issue(
		bundle, study.String(), expected.HeadDigest(), retained.DecisionDigest(), expected.LineageRootDigest(),
	)
	if err != nil || !publicationAuthority.Valid() {
		return PublicationResult{}, refuse(CodeInvalidResiduePublication, "prepared bundle could not issue terminal publication authority", err)
	}
	advanced, advanceErr := objectStore.AdvanceResidue(ctx, expected, publicationAuthority)
	if advanceErr != nil {
		return reopenPublishedResult(ctx, objectStore, study, bundle, PublicationAlreadyCurrent, advanceErr)
	}
	result, err := reopenPublishedResult(ctx, objectStore, study, bundle, PublicationCreated, nil)
	if err != nil {
		return PublicationResult{}, err
	}
	if !samePublicHead(advanced, result.Residue.head) {
		return PublicationResult{}, refuse(
			CodeResiduePublicationAmbiguous, "advanced terminal head differs from exact reopened residue", nil,
		)
	}
	return result, nil
}

func reopenPublishedResult(
	ctx context.Context,
	objectStore *store.ObjectStore,
	study store.StudyID,
	bundle model.ContractBundle,
	disposition PublicationDisposition,
	publicationErr error,
) (PublicationResult, error) {
	residue, openErr := OpenResidue(ctx, objectStore, study)
	if openErr == nil {
		if !bundle.Valid() || residue.BundleDigest() != bundle.Digest() ||
			residue.DecisionRecordDigest() != bundle.DecisionRecordDigest() ||
			residue.ChoicepointDigest() != bundle.ChoicepointDigest() ||
			!bytes.Equal(residue.bundle.CanonicalBytes(), bundle.CanonicalBytes()) {
			return PublicationResult{}, refuse(CodeResidueConflict, "terminal residue differs from the prepared bundle", publicationErr)
		}
		return PublicationResult{Residue: residue, Disposition: disposition}, nil
	}
	if publicationErr == nil {
		return PublicationResult{}, refuse(
			CodeResiduePublicationAmbiguous, "terminal transition returned but exact residue could not be reopened", openErr,
		)
	}
	current, headErr := objectStore.OpenHead(ctx, study)
	if headErr == nil && current.Stage() == store.StageRuling && current.CurrentKind() == "DecisionRecord" &&
		current.CurrentDigest() == bundle.DecisionRecordDigest() {
		return PublicationResult{}, refuse(
			CodeResiduePublicationFailed, "terminal residue was not published and the exact ruling remains current", publicationErr,
		)
	}
	return PublicationResult{}, refuse(
		CodeResiduePublicationAmbiguous,
		"publication failure could not be reconciled to an exact ruling or terminal residue",
		errors.Join(publicationErr, openErr, headErr),
	)
}

// OpenResidue strictly reconstructs one exact terminal bundle and its complete
// durable predecessor chain. It needs no compiler input or prepared authority.
func OpenResidue(
	ctx context.Context,
	objectStore *store.ObjectStore,
	study store.StudyID,
) (Residue, error) {
	if ctx == nil || objectStore == nil || !study.Valid() {
		return Residue{}, refuse(CodeInvalidResidue, "context, store, and study are required", nil)
	}
	head, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return Residue{}, err
	}
	if head.Stage() != store.StageResidue || head.CurrentKind() != "ContractBundle" {
		return Residue{}, refuse(CodeInvalidResidue, "study head is not one terminal ContractBundle residue", nil)
	}
	object, authority, err := objectStore.Read(ctx, "ContractBundle", head.CurrentDigest())
	if err != nil {
		return Residue{}, err
	}
	if err := objectStore.Validate(ctx, object, authority); err != nil {
		return Residue{}, err
	}
	bundle, err := model.ParseContractBundle(object.CanonicalBytes(), head.CurrentDigest())
	if err != nil || !bundle.Valid() || object.Digest() != bundle.Digest() ||
		!bytes.Equal(object.CanonicalBytes(), bundle.CanonicalBytes()) {
		return Residue{}, refuse(CodeInvalidResidue, "terminal head and exact ContractBundle object disagree", err)
	}
	previousHead, hasPreviousHead := head.PreviousHeadDigest()
	if !hasPreviousHead || bundle.PortableSource().Plan().Digest() != head.LineageRootDigest() {
		return Residue{}, refuse(CodeInvalidResidue, "terminal residue head does not bind its exact prior head and source plan", nil)
	}
	publicationAuthority, err := internalpublication.Issue(
		bundle, study.String(), previousHead, bundle.DecisionRecordDigest(), head.LineageRootDigest(),
	)
	if err != nil || !publicationAuthority.Valid() {
		return Residue{}, refuse(CodeInvalidResidue, "terminal residue could not reconstruct its durability authority", err)
	}
	confirmedHead, err := objectStore.ConfirmResiduePublication(ctx, head, publicationAuthority)
	if err != nil || !samePublicHead(confirmedHead, head) {
		return Residue{}, refuse(CodeResiduePublicationAmbiguous, "terminal residue durability could not be confirmed", err)
	}
	predecessor, err := promotion.OpenResiduePredecessor(
		ctx, objectStore, study, bundle.DecisionRecordDigest(), bundle.ChoicepointDigest(),
	)
	if err != nil {
		return Residue{}, err
	}
	residue := Residue{
		objectStore: objectStore, head: head, object: object, authority: authority,
		bundle: bundle, predecessor: predecessor, seal: sealedResidue,
	}
	if !residue.Valid() {
		return Residue{}, refuse(CodeInvalidResidue, "terminal residue failed sealed reconstruction", nil)
	}
	return residue, nil
}

// ReopenResidue requires the same store instance and exact terminal authority,
// then returns a freshly reopened capability for a downstream physical edge.
func ReopenResidue(
	ctx context.Context,
	objectStore *store.ObjectStore,
	residue Residue,
) (Residue, error) {
	if ctx == nil || objectStore == nil || !residue.Valid() || residue.objectStore != objectStore {
		return Residue{}, refuse(CodeInvalidResidue, "residue is not bound to the supplied store instance", nil)
	}
	fresh, err := OpenResidue(ctx, objectStore, residue.StudyID())
	if err != nil {
		return Residue{}, err
	}
	if !samePublicHead(fresh.head, residue.head) || fresh.BundleDigest() != residue.BundleDigest() ||
		fresh.DecisionRecordDigest() != residue.DecisionRecordDigest() ||
		!bytes.Equal(fresh.bundle.CanonicalBytes(), residue.bundle.CanonicalBytes()) {
		return Residue{}, refuse(CodeInvalidResidue, "reopened terminal residue differs from supplied authority", nil)
	}
	return fresh, nil
}

func (r Residue) Valid() bool {
	previous, hasPrevious := r.head.PreviousObjectDigest()
	return r.seal == sealedResidue && r.objectStore != nil &&
		r.head.StudyID().Valid() && r.head.Stage() == store.StageResidue && r.head.CurrentKind() == "ContractBundle" &&
		r.bundle.Digest().Valid() && r.object.Digest().Valid() && r.authorityObjectMatches() && r.predecessor.Valid() &&
		r.head.CurrentDigest() == r.bundle.Digest() && r.object.Digest() == r.bundle.Digest() &&
		hasPrevious && previous == r.bundle.DecisionRecordDigest() &&
		r.predecessor.StudyID().String() == r.head.StudyID().String() &&
		r.predecessor.DecisionDigest() == r.bundle.DecisionRecordDigest() &&
		r.predecessor.ChoicepointDigest() == r.bundle.ChoicepointDigest()
}

func (r Residue) authorityObjectMatches() bool {
	return r.object.Kind() == "ContractBundle" &&
		bytes.Equal(r.object.CanonicalBytes(), r.bundle.CanonicalBytes())
}

func (r Residue) StudyID() store.StudyID {
	if !r.Valid() {
		return store.StudyID{}
	}
	return r.head.StudyID()
}

func (r Residue) BundleDigest() domain.Digest {
	if !r.Valid() {
		return domain.Digest("")
	}
	return r.bundle.Digest()
}

func (r Residue) DecisionRecordDigest() domain.Digest {
	if !r.Valid() {
		return domain.Digest("")
	}
	return r.bundle.DecisionRecordDigest()
}

func (r Residue) ChoicepointDigest() domain.Digest {
	if !r.Valid() {
		return domain.Digest("")
	}
	return r.bundle.ChoicepointDigest()
}

func (r Residue) HeadDigest() domain.Digest {
	if !r.Valid() {
		return domain.Digest("")
	}
	return r.head.HeadDigest()
}

func (r Residue) Bundle() model.ContractBundle {
	if !r.Valid() {
		return model.ContractBundle{}
	}
	bundle, err := model.ParseContractBundle(r.bundle.CanonicalBytes(), r.bundle.Digest())
	if err != nil {
		return model.ContractBundle{}
	}
	return bundle
}

func samePublicHead(left, right store.HeadToken) bool {
	leftPreviousHead, leftHasPreviousHead := left.PreviousHeadDigest()
	rightPreviousHead, rightHasPreviousHead := right.PreviousHeadDigest()
	leftPrevious, leftHasPrevious := left.PreviousObjectDigest()
	rightPrevious, rightHasPrevious := right.PreviousObjectDigest()
	return left.StudyID().String() == right.StudyID().String() && left.Revision() == right.Revision() &&
		left.Stage() == right.Stage() && left.CurrentKind() == right.CurrentKind() &&
		left.CurrentDigest() == right.CurrentDigest() && left.LineageRootDigest() == right.LineageRootDigest() &&
		left.HeadDigest() == right.HeadDigest() && leftHasPreviousHead == rightHasPreviousHead &&
		leftPreviousHead == rightPreviousHead && leftHasPrevious == rightHasPrevious && leftPrevious == rightPrevious
}
