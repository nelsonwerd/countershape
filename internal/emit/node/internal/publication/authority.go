// Package publication owns the store-transition capability for one exact
// prepared ContractBundle. Its Go internal path limits issuance to the node
// emitter subtree; parsed bundles cannot reconstruct the private seal.
package publication

import (
	"bytes"
	"errors"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type seal struct{ marker byte }

// Authority binds one exact ContractBundle to the DecisionRecord that the
// sensitive store transition must still name as its current predecessor.
type Authority struct {
	study        string
	expectedHead domain.Digest
	digest       domain.Digest
	predecessor  domain.Digest
	choicepoint  domain.Digest
	lineageRoot  domain.Digest
	canonical    []byte
	seal         *seal
}

// Issue accepts only a strictly reconstructible in-memory bundle. The caller
// is still responsible for proving that it retains the matching live ruling
// preparation before issuance.
func Issue(
	bundle model.ContractBundle,
	study string,
	expectedHead domain.Digest,
	predecessor domain.Digest,
	lineageRoot domain.Digest,
) (Authority, error) {
	if err := validateBody(
		study, expectedHead, bundle.Digest(), predecessor, bundle.ChoicepointDigest(),
		lineageRoot, bundle.CanonicalBytes(),
	); err != nil {
		return Authority{}, errors.New("invalid ContractBundle publication authority")
	}
	return Authority{
		study: study, expectedHead: expectedHead, digest: bundle.Digest(), predecessor: predecessor,
		choicepoint: bundle.ChoicepointDigest(), lineageRoot: lineageRoot,
		canonical: bundle.CanonicalBytes(), seal: &seal{marker: 1},
	}, nil
}

func (a Authority) Valid() bool {
	return a.seal != nil && a.seal.marker == 1 &&
		validateBody(
			a.study, a.expectedHead, a.digest, a.predecessor, a.choicepoint,
			a.lineageRoot, a.canonical,
		) == nil
}

func (a Authority) StudyID() string                   { return a.study }
func (a Authority) ExpectedHeadDigest() domain.Digest { return a.expectedHead }
func (a Authority) Digest() domain.Digest             { return a.digest }
func (a Authority) Predecessor() domain.Digest        { return a.predecessor }
func (a Authority) Choicepoint() domain.Digest        { return a.choicepoint }
func (a Authority) LineageRoot() domain.Digest        { return a.lineageRoot }
func (a Authority) CanonicalBytes() []byte            { return append([]byte(nil), a.canonical...) }

func (a Authority) Equal(other Authority) bool {
	return a.Valid() && other.Valid() && a.study == other.study && a.expectedHead == other.expectedHead &&
		a.digest == other.digest && a.predecessor == other.predecessor &&
		a.choicepoint == other.choicepoint && a.lineageRoot == other.lineageRoot &&
		bytes.Equal(a.canonical, other.canonical)
}

func validateBody(
	study string,
	expectedHead domain.Digest,
	digest domain.Digest,
	predecessor domain.Digest,
	choicepoint domain.Digest,
	lineageRoot domain.Digest,
	canonical []byte,
) error {
	if len(study) != len("study:")+64 || study[:len("study:")] != "study:" ||
		!expectedHead.Valid() || !digest.Valid() || !predecessor.Valid() ||
		!choicepoint.Valid() || !lineageRoot.Valid() || len(canonical) == 0 {
		return errors.New("ContractBundle publication identity is invalid")
	}
	bundle, err := model.ParseContractBundle(canonical, digest)
	if err != nil || !bundle.Valid() || bundle.Digest() != digest ||
		bundle.DecisionRecordDigest() != predecessor ||
		bundle.ChoicepointDigest() != choicepoint ||
		bundle.PortableSource().Plan().Digest() != lineageRoot ||
		!bytes.Equal(bundle.CanonicalBytes(), canonical) {
		return errors.New("ContractBundle body, predecessor, and digest disagree")
	}
	return nil
}
