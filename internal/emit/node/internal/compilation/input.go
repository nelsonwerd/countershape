// Package compilation owns the sealed, authority-narrowed input accepted by
// the pure Node compiler. Go's internal visibility confines construction and
// inspection to the internal/emit/node subtree.
package compilation

import (
	"bytes"
	"encoding/base64"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type DecisionAction string

const (
	ActionAllowObserved     DecisionAction = "ALLOW_OBSERVED"
	ActionCustomExpectation DecisionAction = "CUSTOM_EXPECTATION"
)

type inputSeal struct{}

var sealedInput = &inputSeal{}

type inputWire struct {
	SchemaVersion        string `json:"schema_version"`
	Kind                 string `json:"kind"`
	DecisionRecordDigest string `json:"decision_record_digest"`
	ChoicepointDigest    string `json:"choicepoint_digest"`
	DecisionAction       string `json:"decision_action"`
	PortableSourceDigest string `json:"portable_source_digest"`
	PortableSourceBase64 string `json:"portable_source_base64"`
	SourceProfileDigest  string `json:"source_profile_digest"`
	SourceProfileBase64  string `json:"source_profile_base64"`
	PredicateBase64      string `json:"predicate_base64"`
}

// Input contains no DecisionRecord, Choicepoint, confirmation, proof,
// candidate, receipt, head, target, runtime measurement, or publication
// authority. The exact PortableSource may itself retain authorized source and
// opaque lineage content; this is authority narrowing, not redaction.
type Input struct {
	decision    domain.Digest
	choicepoint domain.Digest
	action      DecisionAction
	source      contractsource.PortableSource
	profile     model.SourceProfile
	predicate   model.Predicate
	digest      domain.Digest
	canonical   []byte
	seal        *inputSeal
}

// New is the only sanitized-input constructor. The A2 architecture gate
// freezes its sole production call site in the parent application service.
func New(
	decision, choicepoint domain.Digest,
	action DecisionAction,
	source contractsource.PortableSource,
	profile model.SourceProfile,
	predicate model.Predicate,
) (Input, error) {
	if !decision.Valid() || !choicepoint.Valid() ||
		(action != ActionAllowObserved && action != ActionCustomExpectation) {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "decision identity or action is invalid"}
	}
	reparsed, err := contractsource.Parse(source.CanonicalBytes())
	if err != nil || reparsed.Digest() != source.Digest() || !bytes.Equal(reparsed.CanonicalBytes(), source.CanonicalBytes()) {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "PortableSource is not exact", Cause: err}
	}
	if !profile.ValidFor(reparsed) || !predicate.ValidFor(reparsed.Profile(), reparsed.StimulusDigest()) {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "source profile or predicate does not bind the exact source"}
	}
	if action == ActionCustomExpectation && len(predicate.AllowedTuples()) != 1 {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "CUSTOM_EXPECTATION requires exactly one tuple"}
	}
	wire := inputWire{
		SchemaVersion: domain.SchemaVersion, Kind: "NodeCompilationInput",
		DecisionRecordDigest: decision.String(), ChoicepointDigest: choicepoint.String(),
		DecisionAction: string(action), PortableSourceDigest: reparsed.Digest().String(),
		PortableSourceBase64: base64.StdEncoding.EncodeToString(reparsed.CanonicalBytes()),
		SourceProfileDigest:  profile.Digest().String(),
		SourceProfileBase64:  base64.StdEncoding.EncodeToString(profile.CanonicalBytes()),
		PredicateBase64:      base64.StdEncoding.EncodeToString(predicate.CanonicalBytes()),
	}
	canonical, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "private input could not be canonicalized", Cause: err}
	}
	digestRaw, err := canon.DigestBytes("NodeCompilationInput", canonical)
	if err != nil {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "private input digest could not be derived", Cause: err}
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return Input{}, &model.Error{Code: "INVALID_COMPILATION_INPUT", Detail: "private input digest is invalid", Cause: err}
	}
	return Input{
		decision: decision, choicepoint: choicepoint, action: action, source: reparsed,
		profile: profile, predicate: predicate, digest: digest, canonical: canonical, seal: sealedInput,
	}, nil
}

func (i Input) Valid() bool {
	if i.seal != sealedInput || !i.digest.Valid() || len(i.canonical) == 0 {
		return false
	}
	rebuilt, err := New(i.decision, i.choicepoint, i.action, i.source, i.profile, i.predicate)
	return err == nil && rebuilt.digest == i.digest && bytes.Equal(rebuilt.canonical, i.canonical)
}

func (i Input) Digest() domain.Digest               { return i.digest }
func (i Input) CanonicalBytes() []byte              { return append([]byte(nil), i.canonical...) }
func (i Input) DecisionRecordDigest() domain.Digest { return i.decision }
func (i Input) ChoicepointDigest() domain.Digest    { return i.choicepoint }
func (i Input) Action() DecisionAction              { return i.action }
func (i Input) Source() contractsource.PortableSource {
	reparsed, _ := contractsource.Parse(i.source.CanonicalBytes())
	return reparsed
}
func (i Input) SourceProfile() model.SourceProfile { return i.profile }
func (i Input) Predicate() model.Predicate         { return i.predicate }
