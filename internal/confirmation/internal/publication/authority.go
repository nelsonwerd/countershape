// Package publication owns the unforgeable-in-Go publication capability for
// one live FreshConfirmation draft. The Go internal import boundary permits
// issuance only from the confirmation subtree; serialized records cannot call
// Issue or reconstruct the private seal.
package publication

import (
	"bytes"
	"errors"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type seal struct{ marker byte }

// Authority binds exact FreshConfirmation bytes to the reduction object that
// the sensitive store transition must still name as its current predecessor.
type Authority struct {
	digest      domain.Digest
	predecessor domain.Digest
	canonical   []byte
	seal        *seal
}

func Issue(digest, predecessor domain.Digest, canonical []byte) (Authority, error) {
	if err := validateBody(digest, predecessor, canonical); err != nil {
		return Authority{}, errors.New("invalid FreshConfirmation publication authority")
	}
	return Authority{
		digest: digest, predecessor: predecessor, canonical: append([]byte(nil), canonical...),
		seal: &seal{marker: 1},
	}, nil
}

func (a Authority) Valid() bool {
	if a.seal == nil || a.seal.marker != 1 || !a.digest.Valid() || !a.predecessor.Valid() || len(a.canonical) == 0 {
		return false
	}
	return validateBody(a.digest, a.predecessor, a.canonical) == nil
}

func (a Authority) Digest() domain.Digest      { return a.digest }
func (a Authority) Predecessor() domain.Digest { return a.predecessor }
func (a Authority) CanonicalBytes() []byte     { return append([]byte(nil), a.canonical...) }

func (a Authority) Equal(other Authority) bool {
	return a.Valid() && other.Valid() && a.digest == other.digest && a.predecessor == other.predecessor &&
		bytes.Equal(a.canonical, other.canonical)
}

func validateBody(digest, predecessor domain.Digest, canonical []byte) error {
	value, err := canon.Parse(canonical)
	if err != nil {
		return err
	}
	rebuilt, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(rebuilt, canonical) {
		return errors.New("FreshConfirmation body is not exact canonical JSON")
	}
	kindValue, hasKind := value.LookupMember("kind")
	kind, kindIsString := kindValue.Text()
	predecessorValue, hasPredecessor := value.LookupMember("reduction_run_digest")
	predecessorText, predecessorIsString := predecessorValue.Text()
	parsedPredecessor, parseErr := domain.ParseDigest(predecessorText)
	computed, digestErr := canon.DigestBytes("FreshConfirmation", canonical)
	if !digest.Valid() || !predecessor.Valid() || !hasKind || !kindIsString || kind != "FreshConfirmation" ||
		!hasPredecessor || !predecessorIsString || parseErr != nil || parsedPredecessor != predecessor ||
		digestErr != nil || computed.String() != digest.String() {
		return errors.New("FreshConfirmation body, predecessor, and digest disagree")
	}
	return nil
}
