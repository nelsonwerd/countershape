// Package publication owns store-transition capabilities issued only after the
// promotion service has validated its live predecessor capabilities. Its Go
// internal path is importable only from the promotion subtree.
package publication

import (
	"bytes"
	"errors"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type seal struct{ marker byte }

type Choicepoint struct {
	digest      domain.Digest
	predecessor domain.Digest
	canonical   []byte
	seal        *seal
}

func IssueChoicepoint(digest, predecessor domain.Digest, canonical []byte) (Choicepoint, error) {
	if err := validateBody("Choicepoint", "fresh_confirmation_digest", digest, predecessor, "", canonical); err != nil {
		return Choicepoint{}, errors.New("invalid Choicepoint publication authority")
	}
	return Choicepoint{
		digest: digest, predecessor: predecessor, canonical: append([]byte(nil), canonical...),
		seal: &seal{marker: 1},
	}, nil
}

func (a Choicepoint) Valid() bool {
	if a.seal == nil || a.seal.marker != 1 || !a.digest.Valid() || !a.predecessor.Valid() || len(a.canonical) == 0 {
		return false
	}
	return validateBody("Choicepoint", "fresh_confirmation_digest", a.digest, a.predecessor, "", a.canonical) == nil
}

func (a Choicepoint) Digest() domain.Digest      { return a.digest }
func (a Choicepoint) Predecessor() domain.Digest { return a.predecessor }
func (a Choicepoint) CanonicalBytes() []byte     { return append([]byte(nil), a.canonical...) }

type Ruling struct {
	digest      domain.Digest
	predecessor domain.Digest
	action      string
	canonical   []byte
	seal        *seal
}

func IssueRuling(digest, predecessor domain.Digest, action string, canonical []byte) (Ruling, error) {
	if err := validateBody("DecisionRecord", "choicepoint_digest", digest, predecessor, action, canonical); err != nil {
		return Ruling{}, errors.New("invalid DecisionRecord publication authority")
	}
	return Ruling{
		digest: digest, predecessor: predecessor, action: action,
		canonical: append([]byte(nil), canonical...), seal: &seal{marker: 1},
	}, nil
}

func (a Ruling) Valid() bool {
	if a.seal == nil || a.seal.marker != 1 || !a.digest.Valid() || !a.predecessor.Valid() || a.action == "" || len(a.canonical) == 0 {
		return false
	}
	return validateBody("DecisionRecord", "choicepoint_digest", a.digest, a.predecessor, a.action, a.canonical) == nil
}

func (a Ruling) Digest() domain.Digest      { return a.digest }
func (a Ruling) Predecessor() domain.Digest { return a.predecessor }
func (a Ruling) Action() string             { return a.action }
func (a Ruling) CanonicalBytes() []byte     { return append([]byte(nil), a.canonical...) }

func validateBody(kind, predecessorMember string, digest, predecessor domain.Digest, action string, canonical []byte) error {
	value, err := canon.Parse(canonical)
	if err != nil {
		return err
	}
	rebuilt, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(rebuilt, canonical) {
		return errors.New("publication body is not exact canonical JSON")
	}
	kindValue, hasKind := value.LookupMember("kind")
	bodyKind, kindIsString := kindValue.Text()
	predecessorValue, hasPredecessor := value.LookupMember(predecessorMember)
	predecessorText, predecessorIsString := predecessorValue.Text()
	parsedPredecessor, parseErr := domain.ParseDigest(predecessorText)
	computed, digestErr := canon.DigestBytes(kind, canonical)
	if !digest.Valid() || !predecessor.Valid() || !hasKind || !kindIsString || bodyKind != kind ||
		!hasPredecessor || !predecessorIsString || parseErr != nil || parsedPredecessor != predecessor ||
		digestErr != nil || computed.String() != digest.String() {
		return errors.New("publication body, predecessor, and digest disagree")
	}
	if kind == "DecisionRecord" {
		actionValue, hasAction := value.LookupMember("action")
		bodyAction, actionIsString := actionValue.Text()
		if !hasAction || !actionIsString || bodyAction != action || !durableRulingAction(action) {
			return errors.New("DecisionRecord body and durable action disagree")
		}
	} else if action != "" {
		return errors.New("non-ruling publication carried an action")
	}
	return nil
}

func durableRulingAction(action string) bool {
	switch action {
	case "ALLOW_OBSERVED", "CUSTOM_EXPECTATION", "REJECT_ALL", "DEFER":
		return true
	default:
		return false
	}
}
