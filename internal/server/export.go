package server

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/nelsonwerd/countershape/internal/canon"
)

// ExportSnapshot is an immutable, package-issued projection of the same seed
// artifacts used by the local studio. It is deliberately opaque: consumers may
// present these exact bytes, but cannot mint or mutate server authority.
type ExportSnapshot struct {
	bench       []byte
	choicepoint []byte
	reveal      []byte
	decision    []byte
}

// BuildSeedExportSnapshot returns one deterministic resolved local fixture for
// report/package exercises. It does not execute a study or create receipt
// authority.
func BuildSeedExportSnapshot() (ExportSnapshot, error) {
	state, err := newStudioState(StateResolved, bytes.NewReader(bytes.Repeat([]byte{0x9d}, 32)))
	if err != nil {
		return ExportSnapshot{}, err
	}
	rawBench, err := json.Marshal(state.benchResponse())
	if err != nil {
		return ExportSnapshot{}, err
	}
	bench, err := canon.Canonicalize(rawBench)
	if err != nil || !state.record.Valid() || !state.decision.Valid() || len(state.reveal) == 0 {
		return ExportSnapshot{}, errors.New("studio export snapshot could not be closed")
	}
	return ExportSnapshot{
		bench:       append([]byte(nil), bench...),
		choicepoint: state.record.CanonicalBytes(),
		reveal:      append([]byte(nil), state.reveal...),
		decision:    state.decision.CanonicalBytes(),
	}, nil
}

func (s ExportSnapshot) BenchBytes() []byte       { return append([]byte(nil), s.bench...) }
func (s ExportSnapshot) ChoicepointBytes() []byte { return append([]byte(nil), s.choicepoint...) }
func (s ExportSnapshot) RevealBytes() []byte      { return append([]byte(nil), s.reveal...) }
func (s ExportSnapshot) DecisionBytes() []byte    { return append([]byte(nil), s.decision...) }

// Valid reopens every retained strict artifact and checks the exact server
// projection. It is intentionally defensive because ExportSnapshot is a Go
// value and callers may copy it.
func (s ExportSnapshot) Valid() bool {
	fresh, err := BuildSeedExportSnapshot()
	return err == nil && bytes.Equal(s.bench, fresh.bench) &&
		bytes.Equal(s.choicepoint, fresh.choicepoint) &&
		bytes.Equal(s.reveal, fresh.reveal) && bytes.Equal(s.decision, fresh.decision)
}
