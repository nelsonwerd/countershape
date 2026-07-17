// Package model owns the pure, closed values consumed by the Node contract
// emitter. These values carry no store, candidate, receipt, or runtime
// authority.
package model

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	PredicateKindV1  = "one-of-exact/v1"
	PredicateScopeV1 = "EXACT_WITNESSED_STIMULUS"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type exactTagWire struct {
	Tag string `json:"tag"`
}

type exactStringWire struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

type exactIntegerWire struct {
	Tag       string `json:"tag"`
	Canonical string `json:"canonical"`
}

type exactBooleanWire struct {
	Tag   string `json:"tag"`
	Value bool   `json:"value"`
}

type exactBytesWire struct {
	Tag    string `json:"tag"`
	Base64 string `json:"base64"`
}

type exactListWire struct {
	Tag    string   `json:"tag"`
	Values []string `json:"values"`
}

type exactJSONWire struct {
	Tag             string `json:"tag"`
	CanonicalBase64 string `json:"canonical_base64"`
}

// ExactValue is one immutable value in the closed portable algebra. Its wire
// is the exact common.schema.json representation, not Choice's compatibility
// wire and not a caller-paired digest.
type ExactValue struct {
	value     portablevalue.Value
	canonical []byte
}

// NewExactValue defensively reconstructs through the tag-specific
// portablevalue constructors so that the value algebra has one owner.
func NewExactValue(input portablevalue.Value) (ExactValue, error) {
	rebuilt, err := rebuildPortableValue(input)
	if err != nil {
		return ExactValue{}, refuse("INVALID_EXACT_VALUE", "portable value could not be reconstructed", err)
	}
	wire, err := exactValueWire(rebuilt)
	if err != nil {
		return ExactValue{}, err
	}
	canonical, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return ExactValue{}, refuse("INVALID_EXACT_VALUE", "exact value wire could not be canonicalized", err)
	}
	return ExactValue{value: rebuilt, canonical: canonical}, nil
}

func rebuildPortableValue(input portablevalue.Value) (portablevalue.Value, error) {
	if !input.Valid() {
		return portablevalue.Value{}, refuse("INVALID_EXACT_VALUE", "portable value is invalid", nil)
	}
	switch input.Tag() {
	case portablevalue.TagMissing:
		return portablevalue.Missing(), nil
	case portablevalue.TagNull:
		return portablevalue.Null(), nil
	case portablevalue.TagBoolean:
		value, ok := input.BooleanValue()
		if !ok {
			break
		}
		return portablevalue.Boolean(value), nil
	case portablevalue.TagInteger:
		value, ok := input.IntegerText()
		if !ok {
			break
		}
		return portablevalue.Integer(value)
	case portablevalue.TagString:
		value, ok := input.StringText()
		if !ok {
			break
		}
		return portablevalue.String(value)
	case portablevalue.TagBytes:
		value, ok := input.BytesValue()
		if !ok {
			break
		}
		return portablevalue.Bytes(value)
	case portablevalue.TagOrderedStringList:
		value, ok := input.OrderedStrings()
		if !ok {
			break
		}
		return portablevalue.OrderedStringList(value)
	case portablevalue.TagCanonicalJSON:
		value, ok := input.CanonicalJSONBytes()
		if !ok {
			break
		}
		return portablevalue.CanonicalJSON(value)
	}
	return portablevalue.Value{}, refuse("INVALID_EXACT_VALUE", "portable value tag and payload disagree", nil)
}

func exactValueWire(value portablevalue.Value) (any, error) {
	switch value.Tag() {
	case portablevalue.TagMissing, portablevalue.TagNull:
		return exactTagWire{Tag: string(value.Tag())}, nil
	case portablevalue.TagString:
		text, ok := value.StringText()
		if ok {
			return exactStringWire{Tag: string(value.Tag()), Value: text}, nil
		}
	case portablevalue.TagInteger:
		integer, ok := value.IntegerText()
		if ok {
			return exactIntegerWire{Tag: string(value.Tag()), Canonical: integer}, nil
		}
	case portablevalue.TagBoolean:
		boolean, ok := value.BooleanValue()
		if ok {
			return exactBooleanWire{Tag: string(value.Tag()), Value: boolean}, nil
		}
	case portablevalue.TagBytes:
		opaque, ok := value.BytesValue()
		if ok {
			return exactBytesWire{Tag: string(value.Tag()), Base64: base64.StdEncoding.EncodeToString(opaque)}, nil
		}
	case portablevalue.TagOrderedStringList:
		values, ok := value.OrderedStrings()
		if ok {
			return exactListWire{Tag: string(value.Tag()), Values: append([]string(nil), values...)}, nil
		}
	case portablevalue.TagCanonicalJSON:
		exact, ok := value.CanonicalJSONBytes()
		if ok {
			return exactJSONWire{Tag: string(value.Tag()), CanonicalBase64: base64.StdEncoding.EncodeToString(exact)}, nil
		}
	}
	return nil, refuse("INVALID_EXACT_VALUE", "portable value has no exact wire", nil)
}

func (v ExactValue) Valid() bool {
	rebuilt, err := NewExactValue(v.value)
	return err == nil && bytes.Equal(rebuilt.canonical, v.canonical)
}

func (v ExactValue) Tag() portablevalue.Tag { return v.value.Tag() }
func (v ExactValue) CanonicalBytes() []byte { return append([]byte(nil), v.canonical...) }
func (v ExactValue) PortableValue() portablevalue.Value {
	rebuilt, _ := rebuildPortableValue(v.value)
	return rebuilt
}

// ExactField is one field/value pair. Field order is supplied only by the
// profile-aware tuple and predicate constructors.
type ExactField struct {
	id    string
	value ExactValue
}

func NewExactField(fieldID string, value ExactValue) (ExactField, error) {
	if !validFieldID(fieldID) || !value.Valid() {
		return ExactField{}, refuse("INVALID_EXACT_FIELD", "field ID or value is outside the closed v1 model", nil)
	}
	return ExactField{id: fieldID, value: value}, nil
}

func (f ExactField) FieldID() string { return f.id }
func (f ExactField) Value() ExactValue {
	value, _ := NewExactValue(f.value.value)
	return value
}

type exactFieldWire struct {
	FieldID string `json:"field_id"`
	Value   any    `json:"value"`
}

type exactTupleWire struct {
	Fields []exactFieldWire `json:"fields"`
}

// ExactTuple is a complete selected-field tuple. It has no tuple digest and
// preserves the caller's explicitly validated profile order.
type ExactTuple struct {
	fields    []ExactField
	canonical []byte
}

func NewExactTuple(fields []ExactField) (ExactTuple, error) {
	if len(fields) == 0 || len(fields) > portablevalue.MaxTupleFields {
		return ExactTuple{}, refuse("INVALID_EXACT_TUPLE", "tuple field count is outside the closed ceiling", nil)
	}
	copyFields := make([]ExactField, len(fields))
	values := make([]portablevalue.Value, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		if !validFieldID(field.id) || !field.value.Valid() {
			return ExactTuple{}, refuse("INVALID_EXACT_TUPLE", fmt.Sprintf("tuple field %d is invalid", index), nil)
		}
		if _, duplicate := seen[field.id]; duplicate {
			return ExactTuple{}, refuse("INVALID_EXACT_TUPLE", "tuple repeats a field ID", nil)
		}
		seen[field.id] = struct{}{}
		value, _ := NewExactValue(field.value.value)
		copyFields[index] = ExactField{id: field.id, value: value}
		values[index] = value.value
	}
	if err := portablevalue.ValidateTuple(values); err != nil {
		return ExactTuple{}, refuse("INVALID_EXACT_TUPLE", "tuple exceeds the portable aggregate profile", err)
	}
	wire, err := tupleWire(copyFields)
	if err != nil {
		return ExactTuple{}, err
	}
	canonical, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return ExactTuple{}, refuse("INVALID_EXACT_TUPLE", "tuple could not be canonicalized", err)
	}
	return ExactTuple{fields: copyFields, canonical: canonical}, nil
}

func tupleWire(fields []ExactField) (exactTupleWire, error) {
	wire := exactTupleWire{Fields: make([]exactFieldWire, len(fields))}
	for index, field := range fields {
		value, err := exactValueWire(field.value.value)
		if err != nil {
			return exactTupleWire{}, err
		}
		wire.Fields[index] = exactFieldWire{FieldID: field.id, Value: value}
	}
	return wire, nil
}

func (t ExactTuple) Valid() bool {
	rebuilt, err := NewExactTuple(t.fields)
	return err == nil && bytes.Equal(rebuilt.canonical, t.canonical)
}

func (t ExactTuple) Fields() []ExactField {
	result := make([]ExactField, len(t.fields))
	for index, field := range t.fields {
		result[index], _ = NewExactField(field.id, field.value)
	}
	return result
}

func (t ExactTuple) CanonicalBytes() []byte { return append([]byte(nil), t.canonical...) }

type predicateWire struct {
	Kind                  string           `json:"kind"`
	Scope                 string           `json:"scope"`
	StimulusDigest        string           `json:"stimulus_digest"`
	PortableProfileDigest string           `json:"portable_profile_digest"`
	SelectedFields        []string         `json:"selected_fields"`
	AllowedTuples         []exactTupleWire `json:"allowed_tuples"`
}

// Predicate is a canonical set of complete correlated tuples. It cannot
// represent independent per-field allowed values or a Cartesian product.
type Predicate struct {
	stimulusDigest domain.Digest
	profile        projectionprofile.Profile
	selected       []string
	allowed        []ExactTuple
	canonical      []byte
}

func NewPredicate(
	stimulusDigest domain.Digest,
	profile projectionprofile.Profile,
	selectedFields []string,
	allowedTuples []ExactTuple,
) (Predicate, error) {
	if !stimulusDigest.Valid() || !profile.Valid() || len(selectedFields) == 0 || len(selectedFields) > 64 ||
		len(allowedTuples) == 0 || len(allowedTuples) > 4 {
		return Predicate{}, refuse("INVALID_PREDICATE", "predicate identity or bounded roster is incomplete", nil)
	}
	selected := append([]string(nil), selectedFields...)
	descriptors := profile.Fields()
	positions := make(map[string]int, len(descriptors))
	byID := make(map[string]projectionprofile.Descriptor, len(descriptors))
	for index, descriptor := range descriptors {
		positions[descriptor.FieldID] = index
		byID[descriptor.FieldID] = descriptor
	}
	last := -1
	seenSelected := make(map[string]struct{}, len(selected))
	for _, fieldID := range selected {
		position, exists := positions[fieldID]
		if !exists || position <= last {
			return Predicate{}, refuse("INVALID_PREDICATE", "selected fields differ from exact profile order", nil)
		}
		if _, duplicate := seenSelected[fieldID]; duplicate {
			return Predicate{}, refuse("INVALID_PREDICATE", "selected fields repeat", nil)
		}
		seenSelected[fieldID] = struct{}{}
		last = position
	}
	unique := make(map[string]ExactTuple, len(allowedTuples))
	for tupleIndex, tuple := range allowedTuples {
		if err := validateTupleForSelection(tuple, selected, byID); err != nil {
			return Predicate{}, refuse("INVALID_PREDICATE", fmt.Sprintf("allowed tuple %d is incompatible", tupleIndex), err)
		}
		unique[string(tuple.canonical)] = tuple
	}
	allowed := make([]ExactTuple, 0, len(unique))
	for _, tuple := range unique {
		allowed = append(allowed, tuple)
	}
	sort.Slice(allowed, func(i, j int) bool { return bytes.Compare(allowed[i].canonical, allowed[j].canonical) < 0 })
	wire := predicateWire{
		Kind: PredicateKindV1, Scope: PredicateScopeV1, StimulusDigest: stimulusDigest.String(),
		PortableProfileDigest: profile.Digest().String(), SelectedFields: append([]string(nil), selected...),
		AllowedTuples: make([]exactTupleWire, len(allowed)),
	}
	for index, tuple := range allowed {
		tupleIdentity, err := tupleWire(tuple.fields)
		if err != nil {
			return Predicate{}, err
		}
		wire.AllowedTuples[index] = tupleIdentity
	}
	canonical, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return Predicate{}, refuse("INVALID_PREDICATE", "predicate could not be canonicalized", err)
	}
	return Predicate{
		stimulusDigest: stimulusDigest, profile: profile, selected: selected,
		allowed: cloneTuples(allowed), canonical: canonical,
	}, nil
}

func validateTupleForSelection(
	tuple ExactTuple,
	selected []string,
	descriptors map[string]projectionprofile.Descriptor,
) error {
	if !tuple.Valid() || len(tuple.fields) != len(selected) {
		return refuse("INVALID_EXACT_TUPLE", "tuple does not cover the selected roster", nil)
	}
	for index, field := range tuple.fields {
		if field.id != selected[index] {
			return refuse("INVALID_EXACT_TUPLE", "tuple field order differs from selected profile order", nil)
		}
		descriptor := descriptors[field.id]
		tag := field.value.Tag()
		switch tag {
		case portablevalue.TagMissing:
			if !descriptor.AllowMissing {
				return refuse("INVALID_EXACT_TUPLE", "missing value is not allowed by its descriptor", nil)
			}
		case portablevalue.TagNull:
			if !descriptor.AllowNull {
				return refuse("INVALID_EXACT_TUPLE", "null value is not allowed by its descriptor", nil)
			}
		default:
			if tag != descriptor.PortableTag {
				return refuse("INVALID_EXACT_TUPLE", "present value tag differs from its descriptor", nil)
			}
		}
	}
	return nil
}

func (p Predicate) Valid() bool {
	rebuilt, err := NewPredicate(p.stimulusDigest, p.profile, p.selected, p.allowed)
	return err == nil && bytes.Equal(rebuilt.canonical, p.canonical)
}

func (p Predicate) ValidFor(profile projectionprofile.Profile, stimulus domain.Digest) bool {
	return p.Valid() && profile.Valid() && stimulus.Valid() && p.profile.Digest() == profile.Digest() &&
		bytes.Equal(p.profile.CanonicalBytes(), profile.CanonicalBytes()) && p.stimulusDigest == stimulus
}

func (p Predicate) StimulusDigest() domain.Digest        { return p.stimulusDigest }
func (p Predicate) PortableProfileDigest() domain.Digest { return p.profile.Digest() }
func (p Predicate) SelectedFields() []string             { return append([]string(nil), p.selected...) }
func (p Predicate) AllowedTuples() []ExactTuple          { return cloneTuples(p.allowed) }
func (p Predicate) CanonicalBytes() []byte               { return append([]byte(nil), p.canonical...) }

func cloneTuples(input []ExactTuple) []ExactTuple {
	result := make([]ExactTuple, len(input))
	for index, tuple := range input {
		result[index], _ = NewExactTuple(tuple.Fields())
	}
	return result
}

func validFieldID(fieldID string) bool {
	switch fieldID {
	case "http.status", "http.header.content-type", "http.body.kind", "http.body.metadata",
		"cli.completion.kind", "cli.exit.code", "cli.exit.signal", "cli.stdout.bytes",
		"cli.stdout.json.mode", "cli.stdout.json.source", "cli.stderr.text":
		return true
	default:
		return false
	}
}
