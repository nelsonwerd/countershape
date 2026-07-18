package promotion

import (
	"bytes"
	"context"

	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/store"
)

type residuePredecessorSeal struct{}

var sealedResiduePredecessor = &residuePredecessorSeal{}

// ResiduePredecessorSnapshot is a sealed inert reconstruction of the exact
// DecisionRecord -> Choicepoint -> FreshConfirmation chain immediately behind
// one terminal residue. It carries no current-ruling or head-transition
// authority.
type ResiduePredecessorSnapshot struct {
	study        store.StudyID
	decision     choice.DecisionRecord
	choicepoint  choice.ChoicepointRecord
	confirmation confirmation.Record
	seal         *residuePredecessorSeal
}

// StudyID is inert identity only. It lets the node application owner reopen
// the exact study retained by a sealed preparation without exposing its
// private HeadToken or Ruling.
func (p PortableRulingPreparation) StudyID() store.StudyID {
	if !p.Valid() {
		return store.StudyID{}
	}
	return p.ruling.StudyID()
}

// OpenResiduePredecessor reconstructs every durable semantic object behind a
// terminal ContractBundle head. Caller-supplied digests are accepted only as
// checked joins and cannot manufacture a snapshot.
func OpenResiduePredecessor(
	ctx context.Context,
	objectStore *store.ObjectStore,
	study store.StudyID,
	decisionDigest domain.Digest,
	choicepointDigest domain.Digest,
) (ResiduePredecessorSnapshot, error) {
	if ctx == nil || objectStore == nil || !study.Valid() || !decisionDigest.Valid() || !choicepointDigest.Valid() {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "study, store, and exact predecessor digests are required", nil,
		)
	}
	head, err := objectStore.OpenHead(ctx, study)
	if err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	if head.Stage() != store.StageResidue || head.CurrentKind() != "ContractBundle" {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "current study head is not one terminal residue", nil,
		)
	}
	previous, hasPrevious := head.PreviousObjectDigest()
	if !hasPrevious || previous != decisionDigest {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "terminal head does not name the expected DecisionRecord predecessor", nil,
		)
	}

	decisionObject, decisionAuthority, err := objectStore.Read(ctx, "DecisionRecord", decisionDigest)
	if err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	if err := objectStore.Validate(ctx, decisionObject, decisionAuthority); err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	choicepointObject, choicepointAuthority, err := objectStore.Read(ctx, "Choicepoint", choicepointDigest)
	if err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	if err := objectStore.Validate(ctx, choicepointObject, choicepointAuthority); err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	choicepoint, err := choice.ParseChoicepointRecord(choicepointObject.CanonicalBytes())
	if err != nil || choicepoint.Digest() != choicepointDigest || choicepoint.Digest() != choicepointObject.Digest() {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "residue Choicepoint predecessor is not the exact strict object", err,
		)
	}
	decision, err := choice.ParseDecisionRecord(decisionObject.CanonicalBytes(), choicepoint)
	if err != nil || decision.Digest() != decisionDigest || decision.Digest() != decisionObject.Digest() ||
		decision.ChoicepointDigest() != choicepointDigest ||
		(decision.Action() != choice.ActionAllowObserved && decision.Action() != choice.ActionCustomExpectation) {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "residue DecisionRecord and Choicepoint do not rejoin exactly", err,
		)
	}
	confirmationObject, confirmationAuthority, err := objectStore.Read(
		ctx, "FreshConfirmation", choicepoint.ConfirmationDigest(),
	)
	if err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	if err := objectStore.Validate(ctx, confirmationObject, confirmationAuthority); err != nil {
		return ResiduePredecessorSnapshot{}, err
	}
	confirmationRecord, err := confirmation.ParseRecord(confirmationObject.CanonicalBytes())
	if err != nil || !confirmationRecord.Valid() || confirmationRecord.Digest() != choicepoint.ConfirmationDigest() ||
		!bytes.Equal(confirmationRecord.CanonicalBytes(), choicepoint.ConfirmationRecord().CanonicalBytes()) ||
		choicepoint.WorldPlan().Digest() != head.LineageRootDigest() ||
		confirmationRecord.PlanDigest() != head.LineageRootDigest() {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "residue Choicepoint and durable FreshConfirmation disagree", err,
		)
	}
	snapshot := ResiduePredecessorSnapshot{
		study: study, decision: decision, choicepoint: choicepoint,
		confirmation: confirmationRecord, seal: sealedResiduePredecessor,
	}
	if !snapshot.Valid() {
		return ResiduePredecessorSnapshot{}, refuse(
			"INVALID_RESIDUE_PREDECESSOR", "residue predecessor snapshot failed sealed reconstruction", nil,
		)
	}
	return snapshot, nil
}

func (s ResiduePredecessorSnapshot) Valid() bool {
	return s.seal == sealedResiduePredecessor && s.study.Valid() && s.decision.Digest().Valid() &&
		s.choicepoint.Digest().Valid() && s.confirmation.Digest().Valid() &&
		s.decision.ChoicepointDigest() == s.choicepoint.Digest() &&
		s.choicepoint.ConfirmationDigest() == s.confirmation.Digest()
}

func (s ResiduePredecessorSnapshot) StudyID() store.StudyID {
	if !s.Valid() {
		return store.StudyID{}
	}
	return s.study
}

func (s ResiduePredecessorSnapshot) DecisionDigest() domain.Digest {
	if !s.Valid() {
		return domain.Digest("")
	}
	return s.decision.Digest()
}

func (s ResiduePredecessorSnapshot) ChoicepointDigest() domain.Digest {
	if !s.Valid() {
		return domain.Digest("")
	}
	return s.choicepoint.Digest()
}

func (s ResiduePredecessorSnapshot) ConfirmationDigest() domain.Digest {
	if !s.Valid() {
		return domain.Digest("")
	}
	return s.confirmation.Digest()
}
