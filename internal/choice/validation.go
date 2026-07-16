// Package choice owns the pure authority boundary between confirmed outcomes,
// an exact human ruling, and any later contract emitter. A compilable ruling can
// only be produced from outcomes sealed by ConfirmedOutcomeSet.
package choice

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

// Action is an explicit human action. The empty value is intentionally invalid.
type Action string

const (
	ActionAllowObserved     Action = "ALLOW_OBSERVED"
	ActionCustomExpectation Action = "CUSTOM_EXPECTATION"
	ActionRejectAll         Action = "REJECT_ALL"
	ActionDefer             Action = "DEFER"
	ActionRefine            Action = "REFINE"
)

// ValueTag is the identity-bearing representation of one exact field value.
// Missing, null, and a present empty string are deliberately distinct.
type ValueTag string

const (
	ValueMissing           ValueTag = "MISSING"
	ValueString            ValueTag = "STRING"
	ValueInteger           ValueTag = "INTEGER"
	ValueBoolean           ValueTag = "BOOLEAN"
	ValueNull              ValueTag = "NULL"
	ValueBytes             ValueTag = "BYTES"
	ValueOrderedStringList ValueTag = "ORDERED_STRING_LIST"
	ValueCanonicalJSON     ValueTag = "CANONICAL_JSON"
)

// FieldType is the non-null, non-missing type declared by an adapter registry.
type FieldType string

const (
	FieldString            FieldType = "STRING"
	FieldInteger           FieldType = "INTEGER"
	FieldBoolean           FieldType = "BOOLEAN"
	FieldBytes             FieldType = "BYTES"
	FieldOrderedStringList FieldType = "ORDERED_STRING_LIST"
	FieldCanonicalJSON     FieldType = "CANONICAL_JSON"
)

// RefusalCode is a stable machine-readable validation failure.
type RefusalCode string

const (
	CodeMissingAction                         RefusalCode = "MISSING_ACTION"
	CodeUnknownAction                         RefusalCode = "UNKNOWN_ACTION"
	CodeOmittedSelectedFields                 RefusalCode = "OMITTED_SELECTED_FIELDS"
	CodeOmittedObservedSelections             RefusalCode = "OMITTED_OBSERVED_SELECTIONS"
	CodeEmptySelectedFields                   RefusalCode = "EMPTY_SELECTED_FIELDS"
	CodeEmptyFieldID                          RefusalCode = "EMPTY_FIELD_ID"
	CodeUnknownField                          RefusalCode = "UNKNOWN_FIELD"
	CodeDuplicateSelectedField                RefusalCode = "DUPLICATE_SELECTED_FIELD"
	CodeDuplicateTupleField                   RefusalCode = "DUPLICATE_TUPLE_FIELD"
	CodeInvalidFieldType                      RefusalCode = "INVALID_FIELD_TYPE"
	CodeIncompleteTuple                       RefusalCode = "INCOMPLETE_TUPLE"
	CodeEmptyObservedSelection                RefusalCode = "EMPTY_OBSERVED_SELECTION"
	CodeDuplicateObservedSelection            RefusalCode = "DUPLICATE_OBSERVED_SELECTION"
	CodeUnconfirmedOutcomeSelection           RefusalCode = "UNCONFIRMED_OUTCOME_SELECTION"
	CodeCustomExpectationCardinality          RefusalCode = "CUSTOM_EXPECTATION_CARDINALITY"
	CodeCustomExpectationUnreviewed           RefusalCode = "CUSTOM_EXPECTATION_UNREVIEWED"
	CodeCustomExpectationAlreadySeen          RefusalCode = "CUSTOM_EXPECTATION_ALREADY_OBSERVED"
	CodeCustomExpectationNotAdapterRealizable RefusalCode = "CUSTOM_EXPECTATION_NOT_ADAPTER_REALIZABLE"
	CodeNoncompilablePredicate                RefusalCode = "NONCOMPILABLE_ACTION_HAS_PREDICATE"
	CodeUnexpectedCustomExpectation           RefusalCode = "UNEXPECTED_CUSTOM_EXPECTATION"
	CodeAmbiguousScope                        RefusalCode = "AMBIGUOUS_SCOPE"
	CodeInvalidFieldRegistry                  RefusalCode = "INVALID_FIELD_REGISTRY"
	CodeInvalidUTF8                           RefusalCode = "INVALID_UTF8"
	CodeEmptyConfirmedOutcomes                RefusalCode = "EMPTY_CONFIRMED_OUTCOMES"
	CodeInvalidConfirmedOutcome               RefusalCode = "INVALID_CONFIRMED_OUTCOME"
	CodeDuplicateConfirmedOutcome             RefusalCode = "DUPLICATE_CONFIRMED_OUTCOME"
	CodeDuplicateConfirmedCandidate           RefusalCode = "DUPLICATE_CONFIRMED_CANDIDATE"
	CodeProjectionFingerprintMismatch         RefusalCode = "PROJECTION_FINGERPRINT_MISMATCH"
	CodeConfirmedProjectionRosterMismatch     RefusalCode = "CONFIRMED_PROJECTION_ROSTER_MISMATCH"
	CodeNoncanonicalProjection                RefusalCode = "NONCANONICAL_PROJECTION"
	CodeInvalidConfirmedOutcomeSet            RefusalCode = "INVALID_CONFIRMED_OUTCOME_SET"
	CodeInvalidReviewFact                     RefusalCode = "INVALID_REVIEW_FACT"
	CodeReviewExpectationMismatch             RefusalCode = "REVIEW_EXPECTATION_MISMATCH"
	CodeInputLimitExceeded                    RefusalCode = "INPUT_LIMIT_EXCEEDED"
	CodeLegacyWholeProjectionNotPortable      RefusalCode = "LEGACY_WHOLE_PROJECTION_NOT_PORTABLE"
)

// RefusalError carries a stable code plus bounded structural coordinates.
// Detail is diagnostic only and never participates in semantic identity.
type RefusalError struct {
	Code            RefusalCode
	FieldID         string
	AllowedIndex    int
	DisallowedIndex int
	Detail          string
}

func (e *RefusalError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{string(e.Code)}
	if e.FieldID != "" {
		parts = append(parts, "field="+strconv.Quote(e.FieldID))
	}
	if e.AllowedIndex >= 0 {
		parts = append(parts, fmt.Sprintf("allowed=%d", e.AllowedIndex))
	}
	if e.DisallowedIndex >= 0 {
		parts = append(parts, fmt.Sprintf("disallowed=%d", e.DisallowedIndex))
	}
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	}
	return strings.Join(parts, ": ")
}

// IsRefusal reports whether err is a ruling refusal with the exact code.
func IsRefusal(err error, code RefusalCode) bool {
	var refusal *RefusalError
	return errors.As(err, &refusal) && refusal.Code == code
}

func refusal(code RefusalCode, detail string) *RefusalError {
	return &RefusalError{Code: code, AllowedIndex: -1, DisallowedIndex: -1, Detail: detail}
}

// FieldDefinition is an adapter-supplied member of a closed field registry.
// Path addresses the value inside a canonical projection. An empty Path means
// one top-level member whose name is ID; dots in IDs are not interpreted.
type FieldDefinition struct {
	ID           string
	Path         []string
	Type         FieldType
	AllowMissing bool
	AllowNull    bool
}

const WholeProjectionFieldID = "countershape.exact_projection"

// FieldID can only be obtained through a validated FieldRegistry.
type FieldID struct{ text string }

func (id FieldID) String() string { return id.text }

// FieldRegistry is an immutable closed field namespace with its own typed
// identity. A registry obtained from ProjectionDefinition additionally carries
// the full definition digest that authorizes interpretation of a roster.
type FieldRegistry struct {
	definitions                map[string]FieldDefinition
	orderedIDs                 []string
	sourceKinds                map[string]string
	fieldRegistryDigest        domain.Digest
	projectionDefinitionDigest domain.Digest
	profileDigest              domain.Digest
	expectationDomain          projectiontranslate.ExpectationDomain
	mode                       fieldRegistryMode
}

type fieldRegistryMode uint8

const (
	fieldRegistryDefinition fieldRegistryMode = iota + 1
	fieldRegistryLegacyWhole
	fieldRegistryPortable
)

const (
	maxProjectionFields    = 64
	maxProjectionPathDepth = 8
	maxProjectionNameBytes = 128
	maxExactStringBytes    = 64 * 1024
	maxExactIntegerBytes   = 32
	maxActionBytes         = 64
	maxFieldTypeBytes      = 32
	maxTupleRetainedBytes  = 2 * 1024 * 1024
	maxReviewerBytes       = 512
)

type fieldDefinitionIdentity struct {
	ID           string   `json:"id"`
	Path         []string `json:"path"`
	Type         string   `json:"type"`
	AllowMissing bool     `json:"allow_missing"`
	AllowNull    bool     `json:"allow_null"`
}

func newFieldRegistry(definitions []FieldDefinition, preserveOrder bool) (FieldRegistry, error) {
	if len(definitions) == 0 || len(definitions) > maxProjectionFields {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry must declare at least one field")
	}
	registry := FieldRegistry{definitions: make(map[string]FieldDefinition, len(definitions))}
	seenPaths := make(map[string]string, len(definitions))
	for _, raw := range definitions {
		definition := raw
		if definition.ID == "" || len(definition.ID) > maxProjectionNameBytes {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry field id is empty or exceeds the byte limit")
		}
		if !utf8.ValidString(definition.ID) {
			return FieldRegistry{}, refusal(CodeInvalidUTF8, "registry field id is not valid UTF-8")
		}
		if _, exists := registry.definitions[definition.ID]; exists {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry contains duplicate field "+strconv.Quote(definition.ID))
		}
		if len(definition.Type) > maxFieldTypeBytes || !utf8.ValidString(string(definition.Type)) {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry field type is outside the bounded UTF-8 profile")
		}
		switch definition.Type {
		case FieldString, FieldInteger, FieldBoolean, FieldBytes, FieldOrderedStringList, FieldCanonicalJSON:
		default:
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry field has unknown type "+strconv.Quote(string(definition.Type)))
		}
		if len(definition.Path) == 0 {
			definition.Path = []string{definition.ID}
		} else {
			definition.Path = append([]string(nil), definition.Path...)
		}
		if len(definition.Path) > maxProjectionPathDepth {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry field path exceeds the depth limit")
		}
		for _, segment := range definition.Path {
			if segment == "" || len(segment) > maxProjectionNameBytes || !utf8.ValidString(segment) {
				return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry field path contains an empty or invalid UTF-8 segment")
			}
		}
		pathKey := projectionPathIdentityKey(definition.Path)
		if previous, duplicate := seenPaths[pathKey]; duplicate {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "registry fields "+strconv.Quote(previous)+" and "+strconv.Quote(definition.ID)+" alias one projection path")
		}
		seenPaths[pathKey] = definition.ID
		registry.definitions[definition.ID] = definition
		registry.orderedIDs = append(registry.orderedIDs, definition.ID)
	}
	if !preserveOrder {
		sort.Strings(registry.orderedIDs)
	}
	identity := struct {
		SchemaVersion string                    `json:"schema_version"`
		Kind          string                    `json:"kind"`
		Fields        []fieldDefinitionIdentity `json:"fields"`
	}{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "FieldRegistry",
		Fields:        make([]fieldDefinitionIdentity, 0, len(registry.orderedIDs)),
	}
	for _, id := range registry.orderedIDs {
		definition := registry.definitions[id]
		identity.Fields = append(identity.Fields, fieldDefinitionIdentity{
			ID: definition.ID, Path: append([]string(nil), definition.Path...), Type: string(definition.Type),
			AllowMissing: definition.AllowMissing, AllowNull: definition.AllowNull,
		})
	}
	digest, _, err := canon.DigestTyped("FieldRegistry", identity)
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "field registry identity could not be derived")
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "field registry digest is invalid")
	}
	registry.fieldRegistryDigest = parsed
	registry.mode = fieldRegistryDefinition
	return registry, nil
}

func projectionPathIdentityKey(path []string) string {
	var builder strings.Builder
	for _, segment := range path {
		builder.WriteString(strconv.Itoa(len(segment)))
		builder.WriteByte(':')
		builder.WriteString(segment)
	}
	return builder.String()
}

func (r FieldRegistry) FieldRegistryDigest() domain.Digest { return r.fieldRegistryDigest }

func (r FieldRegistry) ProjectionDefinitionDigest() domain.Digest {
	return r.projectionDefinitionDigest
}

// newWholeProjectionRegistry exists only for exact legacy wire reconstruction.
// Fresh construction cannot select or obtain this registry.
func newWholeProjectionRegistry(projectionDefinitionDigest domain.Digest) (FieldRegistry, error) {
	if !projectionDefinitionDigest.Valid() {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "whole-projection registry requires an exact projection definition")
	}
	definition := FieldDefinition{ID: WholeProjectionFieldID, Path: []string{}, Type: FieldCanonicalJSON}
	digest, _, err := canon.DigestTyped("ChoiceWholeProjectionRegistry", struct {
		SchemaVersion              string `json:"schema_version"`
		Kind                       string `json:"kind"`
		ProjectionDefinitionDigest string `json:"projection_definition_digest"`
		FieldID                    string `json:"field_id"`
	}{domain.SchemaVersion, "ChoiceWholeProjectionRegistry", projectionDefinitionDigest.String(), WholeProjectionFieldID})
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "whole-projection registry identity could not be derived")
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "whole-projection registry digest is invalid")
	}
	return FieldRegistry{
		definitions: map[string]FieldDefinition{WholeProjectionFieldID: definition},
		orderedIDs:  []string{WholeProjectionFieldID}, fieldRegistryDigest: parsed,
		projectionDefinitionDigest: projectionDefinitionDigest, mode: fieldRegistryLegacyWhole,
	}, nil
}

func (r FieldRegistry) Definitions() []FieldDefinition {
	result := make([]FieldDefinition, 0, len(r.orderedIDs))
	for _, id := range r.orderedIDs {
		definition := r.definitions[id]
		definition.Path = append([]string(nil), definition.Path...)
		result = append(result, definition)
	}
	return result
}

// Resolve closes a textual ID against the domain registry.
func (r FieldRegistry) Resolve(text string) (FieldID, error) {
	if text == "" {
		return FieldID{}, refusal(CodeEmptyFieldID, "field ids are never defaulted")
	}
	if len(text) > maxProjectionNameBytes {
		return FieldID{}, refusal(CodeInputLimitExceeded, "field id exceeds the closed registry byte limit")
	}
	if !utf8.ValidString(text) {
		return FieldID{}, refusal(CodeInvalidUTF8, "field id is not valid UTF-8")
	}
	if _, exists := r.definitions[text]; !exists {
		err := refusal(CodeUnknownField, "field is not in the closed registry")
		err.FieldID = text
		return FieldID{}, err
	}
	return FieldID{text: text}, nil
}

func (r FieldRegistry) fieldOrder(text string) (int, bool) {
	for index, fieldID := range r.orderedIDs {
		if fieldID == text {
			return index, true
		}
	}
	return 0, false
}

// ExactValue has private storage so invalid tag/payload combinations cannot be
// assembled by a caller. Canonical payloads retain proof bytes and canonical
// JSON additionally carries a locally computed digest; callers can never inject
// a digest as the value.
type ExactValue struct {
	tag       ValueTag
	text      string
	boolean   bool
	opaque    []byte
	canonical []byte
}

func MissingValue() ExactValue { return ExactValue{tag: ValueMissing} }

func StringValue(value string) (ExactValue, error) {
	if len(value) > maxExactStringBytes {
		return ExactValue{}, refusal(CodeInputLimitExceeded, "string value exceeds the exact-value byte limit")
	}
	if !utf8.ValidString(value) {
		return ExactValue{}, refusal(CodeInvalidUTF8, "string value is not valid UTF-8")
	}
	return ExactValue{tag: ValueString, text: value}, nil
}

func BooleanValue(value bool) ExactValue { return ExactValue{tag: ValueBoolean, boolean: value} }

func NullValue() ExactValue { return ExactValue{tag: ValueNull} }

// BytesValue retains exact opaque bytes. UTF-8 is deliberately irrelevant.
func BytesValue(value []byte) (ExactValue, error) {
	portable, err := portablevalue.Bytes(value)
	if err != nil {
		return ExactValue{}, refusal(CodeInputLimitExceeded, "byte value exceeds the portable exact-value profile")
	}
	body, _ := portable.BytesValue()
	return ExactValue{tag: ValueBytes, opaque: body}, nil
}

// OrderedStringListValue preserves order, duplicates, empty lists, and empty
// members as strict canonical JSON-array bytes.
func OrderedStringListValue(values []string) (ExactValue, error) {
	portable, err := portablevalue.OrderedStringList(values)
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "ordered string list is outside the portable exact-value profile")
	}
	exact, _ := portable.CanonicalStringListBytes()
	return ExactValue{tag: ValueOrderedStringList, canonical: exact}, nil
}

// IntegerValue delegates lexical and safe-range authority to canon.
func IntegerValue(spelling string) (ExactValue, error) {
	if len(spelling) > maxExactIntegerBytes {
		return ExactValue{}, refusal(CodeInputLimitExceeded, "integer spelling exceeds the exact-value byte limit")
	}
	if !utf8.ValidString(spelling) {
		return ExactValue{}, refusal(CodeInvalidUTF8, "integer spelling is not valid UTF-8")
	}
	value, err := canon.IntegerFromString(spelling)
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "integer is outside the canonical safe-integer profile")
	}
	integer, ok := value.Int64()
	if !ok {
		return ExactValue{}, refusal(CodeInvalidFieldType, "canonical integer authority returned a non-integer")
	}
	return ExactValue{tag: ValueInteger, text: strconv.FormatInt(integer, 10)}, nil
}

// CanonicalJSONValue computes identity from an immutable canon.Value.
func CanonicalJSONValue(value canon.Value) (ExactValue, error) {
	if value.Kind() == canon.KindNull {
		return ExactValue{}, refusal(CodeInvalidFieldType, "JSON null must use the explicit NULL value tag")
	}
	canonicalBytes, err := value.CanonicalChecked()
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON value is outside the closed resource profile")
	}
	if len(canonicalBytes) > canon.MaxInputBytes {
		return ExactValue{}, refusal(CodeInputLimitExceeded, "canonical JSON value exceeds the parse-round-trip ceiling")
	}
	digest, err := canon.DigestValue("ChoiceCanonicalJSON", value)
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON digest could not be computed")
	}
	return ExactValue{
		tag:       ValueCanonicalJSON,
		text:      digest.String(),
		canonical: append([]byte(nil), canonicalBytes...),
	}, nil
}

// CanonicalJSONBytes admits exact canonical bytes only, then computes identity.
func CanonicalJSONBytes(canonicalBytes []byte) (ExactValue, error) {
	value, err := canon.Parse(canonicalBytes)
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON bytes do not parse under the closed profile")
	}
	checked, err := value.CanonicalChecked()
	if err != nil {
		return ExactValue{}, refusal(CodeInvalidFieldType, "canonical JSON value is outside the closed resource profile")
	}
	if !bytes.Equal(checked, canonicalBytes) {
		return ExactValue{}, refusal(CodeNoncanonicalProjection, "canonical JSON value bytes were not canonical")
	}
	return CanonicalJSONValue(value)
}

func (v ExactValue) Tag() ValueTag { return v.tag }
func (v ExactValue) Text() string  { return v.text }
func (v ExactValue) Boolean() bool { return v.boolean }

// Bytes returns a defensive copy only for BYTES values.
func (v ExactValue) Bytes() []byte {
	if v.tag != ValueBytes {
		return nil
	}
	return append([]byte(nil), v.opaque...)
}

// OrderedStrings returns a defensive decoded list only for ordered-list values.
func (v ExactValue) OrderedStrings() ([]string, bool) {
	if v.tag != ValueOrderedStringList {
		return nil, false
	}
	portable, err := portablevalue.OrderedStringListFromCanonical(v.canonical)
	if err != nil {
		return nil, false
	}
	return portable.OrderedStrings()
}

// CanonicalBytes returns defensive canonical payload bytes for canonical JSON
// and ordered-string-list values.
func (v ExactValue) CanonicalBytes() []byte {
	return append([]byte(nil), v.canonical...)
}

func (v ExactValue) identityKey(mode fieldRegistryMode) string {
	switch v.tag {
	case ValueMissing:
		return "M"
	case ValueNull:
		return "N"
	case ValueString:
		return "S" + lengthPrefix(v.text)
	case ValueInteger:
		return "I" + lengthPrefix(v.text)
	case ValueBoolean:
		if v.boolean {
			return "B1"
		}
		return "B0"
	case ValueBytes:
		return "X" + lengthPrefix(string(v.opaque))
	case ValueOrderedStringList:
		return "L" + lengthPrefix(string(v.canonical))
	case ValueCanonicalJSON:
		if mode == fieldRegistryPortable {
			return "J" + lengthPrefix(string(v.canonical))
		}
		return "J" + lengthPrefix(v.text)
	default:
		return "?"
	}
}

// FieldValue is one explicit field in a confirmed or custom complete tuple.
type FieldValue struct {
	FieldID string
	Value   ExactValue
}

// CompleteTuple is a complete domain projection. Slice order is display-only.
type CompleteTuple struct{ Fields []FieldValue }

// SelectedTuple is a sealed, registry-ordered custom expectation containing
// exactly the fields a ruling selected and no unselected profile values. It is
// constructible only against one exact Choicepoint/confirmed universe.
type SelectedTuple struct {
	universe    canon.Digest
	selectedKey string
	tupleKey    string
	fields      []FieldValue
	seal        *selectedTupleSeal
}

type selectedTupleSeal struct{}

var selectedTupleAuthority = &selectedTupleSeal{}

// NewSelectedTuple validates caller-authored exact values against the sealed
// profile owned by record. Field input order is ignored; the retained tuple is
// canonicalized exclusively in registry order.
func NewSelectedTuple(record ChoicepointRecord, selectedFields []string, fields []FieldValue) (SelectedTuple, error) {
	if !record.Valid() || !record.confirmed.Valid() {
		return SelectedTuple{}, refusal(CodeInvalidConfirmedOutcomeSet, "selected tuple requires an exact Choicepoint")
	}
	return newSelectedTuple(record.confirmed, selectedFields, fields)
}

func newSelectedTuple(confirmed ConfirmedOutcomeSet, selectedFields []string, fields []FieldValue) (SelectedTuple, error) {
	if !confirmed.Valid() {
		return SelectedTuple{}, refusal(CodeInvalidConfirmedOutcomeSet, "selected tuple requires a sealed confirmed universe")
	}
	selected, err := confirmed.registry.resolveSelected(selectedFields)
	if err != nil {
		return SelectedTuple{}, err
	}
	if len(selected) == 0 {
		return SelectedTuple{}, refusal(CodeEmptySelectedFields, "selected tuple requires a nonempty selected-field set")
	}
	values, err := confirmed.registry.validateSelectedTuple(selected, fields)
	if err != nil {
		return SelectedTuple{}, err
	}
	projected, key, err := projectTuple(confirmed.registry, selected, values)
	if err != nil {
		return SelectedTuple{}, err
	}
	return SelectedTuple{
		universe: confirmed.seal, selectedKey: selectedIdentityKey(selected), tupleKey: key,
		fields: cloneTuple(projected).Fields, seal: selectedTupleAuthority,
	}, nil
}

// Fields returns the exact selected values in profile registry order.
func (t SelectedTuple) Fields() []FieldValue {
	return cloneTuple(CompleteTuple{Fields: t.fields}).Fields
}

func (t SelectedTuple) validFor(confirmed ConfirmedOutcomeSet, selected []FieldID) bool {
	if t.seal != selectedTupleAuthority || t.universe != confirmed.seal ||
		t.selectedKey != selectedIdentityKey(selected) || len(t.fields) != len(selected) {
		return false
	}
	values, err := confirmed.registry.validateSelectedTuple(selected, t.fields)
	if err != nil {
		return false
	}
	projected, key, err := projectTuple(confirmed.registry, selected, values)
	return err == nil && key == t.tupleKey && tupleIdentityKey(confirmed.registry, projected.Fields) == t.tupleKey
}

func cloneSelectedTuplePointer(input *SelectedTuple) *SelectedTuple {
	if input == nil {
		return nil
	}
	copy := *input
	copy.fields = cloneTuple(CompleteTuple{Fields: input.fields}).Fields
	return &copy
}

// ProjectionProofInput supplies the canonical projection bytes needed to
// reconstruct an exact tuple. It is not confirmation authority: candidate and
// fingerprint membership must be verified against compare's opaque,
// map-derived ConfirmedProjectionRoster. That roster proves U1 map binding; it
// deliberately does not claim the physical fresh-confirmation lineage owned by
// the later U6 execution/store boundary.
type ProjectionProofInput struct {
	CandidateExecutionKey domain.CandidateExecutionKey
	CanonicalProjection   []byte
}

// ConfirmedOutcomeID is opaque and is derived from candidate identity plus the
// verified projection fingerprint. There is no public parser or constructor.
type ConfirmedOutcomeID struct{ text string }

func (id ConfirmedOutcomeID) String() string { return id.text }

// ConfirmedOutcomeRef is an opaque capability returned by a particular sealed
// set. Rulings select these capabilities, never raw tuples.
type ConfirmedOutcomeRef struct {
	id          ConfirmedOutcomeID
	candidate   domain.CandidateExecutionKey
	fingerprint domain.ProjectionFingerprint
	universe    canon.Digest
}

func (r ConfirmedOutcomeRef) ID() ConfirmedOutcomeID                              { return r.id }
func (r ConfirmedOutcomeRef) CandidateExecutionKey() domain.CandidateExecutionKey { return r.candidate }
func (r ConfirmedOutcomeRef) ProjectionFingerprint() domain.ProjectionFingerprint {
	return r.fingerprint
}

type confirmedOutcome struct {
	ref                 ConfirmedOutcomeRef
	tuple               CompleteTuple
	canonicalProjection []byte
}

// ConfirmedOutcomeSet is immutable and constructible only through verified
// canonical projections. Its zero value is invalid.
type ConfirmedOutcomeSet struct {
	registry           FieldRegistry
	ordered            []confirmedOutcome
	byID               map[string]confirmedOutcome
	outcomeMapDigest   compare.OutcomeArtifactDigest
	preservationDigest compare.PreservationMapDigest
	seal               canon.Digest
	valid              bool
}

// newLegacyConfirmedOutcomeSet reconstructs the exact historical whole-
// projection universe. It is private so fresh callers cannot author an
// alternate projection interpretation beside the adapter-derived translator.
func newLegacyConfirmedOutcomeSet(
	registry FieldRegistry,
	roster compare.ConfirmedProjectionRoster,
	inputs []ProjectionProofInput,
) (ConfirmedOutcomeSet, error) {
	if len(registry.definitions) == 0 || len(registry.orderedIDs) == 0 || !registry.fieldRegistryDigest.Valid() || !registry.projectionDefinitionDigest.Valid() {
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidFieldRegistry, "confirmed outcomes require a validated field registry")
	}
	if !roster.Valid() {
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "confirmed outcomes require an opaque map-derived confirmation roster")
	}
	if registry.projectionDefinitionDigest != roster.ProjectionDefinitionDigest() { // MUTANT_U1_CHOICE_IGNORE_PROJECTION_DEFINITION
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidFieldRegistry, "field registry does not match the confirmed plan projection definition")
	}
	if len(inputs) == 0 {
		return ConfirmedOutcomeSet{}, refusal(CodeEmptyConfirmedOutcomes, "at least one confirmed outcome is required")
	}
	if len(inputs) != len(roster.Entries()) {
		return ConfirmedOutcomeSet{}, refusal(CodeConfirmedProjectionRosterMismatch, "projection proofs must cover the confirmed map roster exactly")
	}
	set := ConfirmedOutcomeSet{
		registry:           registry.clone(),
		ordered:            make([]confirmedOutcome, 0, len(inputs)),
		byID:               make(map[string]confirmedOutcome, len(inputs)),
		outcomeMapDigest:   roster.OutcomeMapDigest(),
		preservationDigest: roster.PreservationDigest(),
		valid:              true,
	}
	seenCandidates := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		if !input.CandidateExecutionKey.Valid() {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcome, fmt.Sprintf("projection proof %d has invalid candidate identity", index))
		}
		candidateKey := input.CandidateExecutionKey.String()
		if _, duplicate := seenCandidates[candidateKey]; duplicate {
			return ConfirmedOutcomeSet{}, refusal(CodeDuplicateConfirmedCandidate, "one confirmed universe cannot contain two outcomes for the same candidate")
		}
		canonicalValue, err := canon.Parse(input.CanonicalProjection)
		if err != nil {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcome, fmt.Sprintf("confirmation %d projection is outside the canonical profile", index))
		}
		checked, checkedErr := canonicalValue.CanonicalChecked()
		if checkedErr != nil || !bytes.Equal(checked, input.CanonicalProjection) {
			return ConfirmedOutcomeSet{}, refusal(CodeNoncanonicalProjection, fmt.Sprintf("confirmation %d projection bytes are not canonical", index))
		}
		fingerprint, err := roster.Verify(input.CandidateExecutionKey, input.CanonicalProjection)
		if err != nil {
			var domainErr *domain.Error
			if errors.As(err, &domainErr) && domainErr.Code == "CONFIRMED_PROJECTION_FINGERPRINT_MISMATCH" {
				return ConfirmedOutcomeSet{}, refusal(CodeProjectionFingerprintMismatch, fmt.Sprintf("confirmation %d projection differs from the map-confirmed fingerprint", index))
			}
			return ConfirmedOutcomeSet{}, refusal(CodeConfirmedProjectionRosterMismatch, fmt.Sprintf("confirmation %d candidate/projection pair is not in the confirmed map roster", index))
		}
		tuple, err := set.registry.tupleFromProjection(canonicalValue)
		if err != nil {
			return ConfirmedOutcomeSet{}, err
		}
		id, err := makeConfirmedOutcomeID(input.CandidateExecutionKey, fingerprint)
		if err != nil {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcome, "confirmed outcome identity could not be derived")
		}
		if _, duplicate := set.byID[id.text]; duplicate {
			return ConfirmedOutcomeSet{}, refusal(CodeDuplicateConfirmedOutcome, "derived confirmed outcome identity occurs more than once")
		}
		seenCandidates[candidateKey] = struct{}{}
		outcome := confirmedOutcome{
			ref:                 ConfirmedOutcomeRef{id: id, candidate: input.CandidateExecutionKey, fingerprint: fingerprint},
			tuple:               cloneTuple(tuple),
			canonicalProjection: append([]byte(nil), input.CanonicalProjection...),
		}
		set.byID[id.text] = outcome
		set.ordered = append(set.ordered, outcome)
	}
	sort.Slice(set.ordered, func(i, j int) bool { return set.ordered[i].ref.id.text < set.ordered[j].ref.id.text })
	seal, err := makeConfirmedOutcomeSetSeal(set.registry, set.outcomeMapDigest, set.preservationDigest, set.ordered)
	if err != nil {
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "confirmed universe seal could not be derived")
	}
	set.seal = seal
	for index := range set.ordered {
		set.ordered[index].ref.universe = seal
		set.byID[set.ordered[index].ref.id.text] = set.ordered[index]
	}
	return set, nil
}

func makeConfirmedOutcomeID(candidate domain.CandidateExecutionKey, fingerprint domain.ProjectionFingerprint) (ConfirmedOutcomeID, error) {
	identity := struct {
		SchemaVersion         string `json:"schema_version"`
		Kind                  string `json:"kind"`
		CandidateExecutionKey string `json:"candidate_execution_key"`
		ProjectionFingerprint string `json:"projection_fingerprint"`
	}{
		SchemaVersion:         domain.SchemaVersion,
		Kind:                  "ConfirmedOutcomeIdentity",
		CandidateExecutionKey: candidate.String(),
		ProjectionFingerprint: fingerprint.String(),
	}
	digest, _, err := canon.DigestTyped("ConfirmedOutcomeIdentity", identity)
	if err != nil {
		return ConfirmedOutcomeID{}, err
	}
	return ConfirmedOutcomeID{text: "outcome:" + strings.TrimPrefix(digest.String(), "sha256:")}, nil
}

func makeConfirmedOutcomeSetSeal(
	registry FieldRegistry,
	outcomeMapDigest compare.OutcomeArtifactDigest,
	preservationDigest compare.PreservationMapDigest,
	outcomes []confirmedOutcome,
) (canon.Digest, error) {
	type outcomeIdentity struct {
		ConfirmedOutcomeID    string `json:"confirmed_outcome_id"`
		CandidateExecutionKey string `json:"candidate_execution_key"`
		ProjectionFingerprint string `json:"projection_fingerprint"`
	}
	identity := struct {
		SchemaVersion              string                    `json:"schema_version"`
		Kind                       string                    `json:"kind"`
		OutcomeMapDigest           string                    `json:"outcome_map_digest"`
		PreservationDigest         string                    `json:"preservation_map_digest"`
		ProjectionDefinitionDigest string                    `json:"projection_definition_digest"`
		FieldRegistryDigest        string                    `json:"field_registry_digest"`
		PortableProfileDigest      string                    `json:"portable_profile_digest"`
		ExpectationDomainDigest    string                    `json:"expectation_domain_digest"`
		RegistryMode               int                       `json:"registry_mode"`
		Fields                     []fieldDefinitionIdentity `json:"fields"`
		Outcomes                   []outcomeIdentity         `json:"outcomes"`
	}{
		SchemaVersion:              domain.SchemaVersion,
		Kind:                       "ConfirmedOutcomeSetIdentity",
		OutcomeMapDigest:           outcomeMapDigest.String(),
		PreservationDigest:         preservationDigest.String(),
		ProjectionDefinitionDigest: registry.projectionDefinitionDigest.String(),
		FieldRegistryDigest:        registry.fieldRegistryDigest.String(),
		RegistryMode:               int(registry.mode),
		Fields:                     make([]fieldDefinitionIdentity, 0, len(registry.orderedIDs)),
		Outcomes:                   make([]outcomeIdentity, 0, len(outcomes)),
	}
	if registry.profileDigest.Valid() {
		identity.PortableProfileDigest = registry.profileDigest.String()
	}
	if registry.expectationDomain.Valid() {
		identity.ExpectationDomainDigest = registry.expectationDomain.Digest().String()
	}
	for _, id := range registry.orderedIDs {
		definition := registry.definitions[id]
		identity.Fields = append(identity.Fields, fieldDefinitionIdentity{
			ID:           definition.ID,
			Path:         append([]string(nil), definition.Path...),
			Type:         string(definition.Type),
			AllowMissing: definition.AllowMissing,
			AllowNull:    definition.AllowNull,
		})
	}
	for _, outcome := range outcomes {
		identity.Outcomes = append(identity.Outcomes, outcomeIdentity{
			ConfirmedOutcomeID:    outcome.ref.id.text,
			CandidateExecutionKey: outcome.ref.candidate.String(),
			ProjectionFingerprint: outcome.ref.fingerprint.String(),
		})
	}
	digest, _, err := canon.DigestTyped("ConfirmedOutcomeSetIdentity", identity)
	return digest, err
}

// Outcomes returns the complete confirmed capability set in canonical ID order.
func (s ConfirmedOutcomeSet) Outcomes() []ConfirmedOutcomeRef {
	result := make([]ConfirmedOutcomeRef, len(s.ordered))
	for index := range s.ordered {
		result[index] = s.ordered[index].ref
	}
	return result
}

// OutcomeMapDigest identifies the exact confirmed map that authorized this set.
func (s ConfirmedOutcomeSet) OutcomeMapDigest() compare.OutcomeArtifactDigest {
	return s.outcomeMapDigest
}

// PreservationDigest identifies the complete labeled candidate/projection map.
func (s ConfirmedOutcomeSet) PreservationDigest() compare.PreservationMapDigest {
	return s.preservationDigest
}

func (s ConfirmedOutcomeSet) Valid() bool {
	return s.valid && s.seal != (canon.Digest{}) && len(s.ordered) > 0 && len(s.byID) == len(s.ordered)
}

func (s ConfirmedOutcomeSet) Registry() FieldRegistry { return s.registry.clone() }

func (s ConfirmedOutcomeSet) ProjectionProofs() []ProjectionProofInput {
	result := make([]ProjectionProofInput, len(s.ordered))
	for index, outcome := range s.ordered {
		result[index] = ProjectionProofInput{
			CandidateExecutionKey: outcome.ref.candidate,
			CanonicalProjection:   append([]byte(nil), outcome.canonicalProjection...),
		}
	}
	return result
}

func (r FieldRegistry) tupleFromProjection(projection canon.Value) (CompleteTuple, error) {
	tuple := CompleteTuple{Fields: make([]FieldValue, 0, len(r.orderedIDs))}
	retainedBytes := 0
	for _, id := range r.orderedIDs {
		definition := r.definitions[id]
		value, found, err := lookupProjectionPath(projection, definition.Path)
		if err != nil {
			return CompleteTuple{}, err
		}
		var exact ExactValue
		if !found {
			exact = MissingValue()
		} else if definition.Type == FieldCanonicalJSON && value.Kind() != canon.KindNull {
			exact, err = CanonicalJSONValue(value)
			if err != nil {
				return CompleteTuple{}, err
			}
		} else {
			exact, err = exactValueFromCanonical(value)
			if err != nil {
				return CompleteTuple{}, err
			}
		}
		if err := validateValue(definition, exact); err != nil {
			return CompleteTuple{}, err
		}
		retainedBytes += exactValueRetainedBytes(exact)
		if retainedBytes > maxTupleRetainedBytes {
			return CompleteTuple{}, refusal(CodeInputLimitExceeded, "projected tuple exceeds the retained canonical-byte ceiling")
		}
		tuple.Fields = append(tuple.Fields, FieldValue{FieldID: id, Value: exact})
	}
	return tuple, nil
}

func lookupProjectionPath(root canon.Value, path []string) (canon.Value, bool, error) {
	current := root
	for depth, segment := range path {
		if current.Kind() != canon.KindObject {
			return canon.Value{}, false, refusal(CodeInvalidConfirmedOutcome, fmt.Sprintf("projection path %q crosses a non-object at depth %d", strings.Join(path, "."), depth))
		}
		next, found := current.LookupMember(segment)
		if !found {
			return canon.Value{}, false, nil
		}
		current = next
	}
	return current, true, nil
}

func exactValueFromCanonical(value canon.Value) (ExactValue, error) {
	switch value.Kind() {
	case canon.KindNull:
		return NullValue(), nil
	case canon.KindBoolean:
		boolean, _ := value.Boolean()
		return BooleanValue(boolean), nil
	case canon.KindInteger:
		integer, _ := value.Int64()
		return IntegerValue(strconv.FormatInt(integer, 10))
	case canon.KindString:
		text, _ := value.Text()
		return StringValue(text)
	case canon.KindArray, canon.KindObject:
		return CanonicalJSONValue(value)
	default:
		return ExactValue{}, refusal(CodeInvalidFieldType, "projection contains an unknown canonical value kind")
	}
}

// CustomExpectationReview is a sealed, exact-tuple review fact. It is bound to
// selected fields and the projected tuple, so a generic review cannot authorize
// a subsequently changed expectation.
type CustomExpectationReview struct {
	universe       canon.Digest
	selectedKey    string
	expectationKey string
	reviewer       string
	evidence       domain.Digest
	valid          bool
}

func (r CustomExpectationReview) Reviewer() string              { return r.reviewer }
func (r CustomExpectationReview) EvidenceDigest() domain.Digest { return r.evidence }

// NewCustomExpectationReview validates and seals one exact review fact against
// the confirmed universe whose complete complement the ruling will reject.
func NewCustomExpectationReview(confirmed ConfirmedOutcomeSet, selectedFields []string, expectation SelectedTuple, reviewer string, evidence domain.Digest) (CustomExpectationReview, error) {
	if !confirmed.valid || len(confirmed.byID) == 0 {
		return CustomExpectationReview{}, refusal(CodeInvalidConfirmedOutcomeSet, "custom review requires a sealed confirmed universe")
	}
	if reviewer == "" || len(reviewer) > maxReviewerBytes || !utf8.ValidString(reviewer) ||
		strings.TrimSpace(reviewer) == "" || containsControl(reviewer) || !evidence.Valid() {
		return CustomExpectationReview{}, refusal(CodeInvalidReviewFact, "reviewer and review-evidence digest must be explicit and valid")
	}
	selected, err := confirmed.registry.resolveSelected(selectedFields)
	if err != nil {
		return CustomExpectationReview{}, err
	}
	if len(selected) == 0 {
		return CustomExpectationReview{}, refusal(CodeEmptySelectedFields, "a custom review must bind a nonempty selected-field set")
	}
	if !expectation.validFor(confirmed, selected) {
		return CustomExpectationReview{}, refusal(CodeReviewExpectationMismatch, "selected tuple does not bind this exact confirmed universe and field selection")
	}
	return CustomExpectationReview{
		universe:       confirmed.seal,
		selectedKey:    selectedIdentityKey(selected),
		expectationKey: expectation.tupleKey,
		reviewer:       reviewer,
		evidence:       evidence,
		valid:          true,
	}, nil
}

// RulingInput has no semantic defaults. Observed selections are opaque refs.
// There is intentionally no caller-authored disallowed-outcomes field: the
// complete complement is derived from ConfirmedOutcomeSet.
type RulingInput struct {
	Action            Action
	SelectedFields    []string
	AllowedObserved   []ConfirmedOutcomeRef
	CustomExpectation *SelectedTuple
	CustomReview      CustomExpectationReview
}

// CompileEligibility is a sealed sum. There is no bool an emitter can coerce.
type CompileEligibility interface {
	isCompileEligibility()
	Action() Action
}

// NoncompilableRuling is returned for REJECT_ALL, DEFER, and REFINE.
type NoncompilableRuling interface {
	CompileEligibility
	isNoncompilableRuling()
}

type noncompilableRuling struct{ action Action }

func (noncompilableRuling) isCompileEligibility()  {}
func (noncompilableRuling) isNoncompilableRuling() {}
func (r noncompilableRuling) Action() Action       { return r.action }

// CompilableRuling contains only a checked ALLOW_OBSERVED or
// CUSTOM_EXPECTATION predicate. Both sides of its confirmed-outcome partition
// and its complete-tuple predicate are derived, immutable values.
type CompilableRuling interface {
	CompileEligibility
	isCompilableRuling()
	SelectedFields() []FieldID
	AllowedTuples() []CompleteTuple
	DisallowedTuples() []CompleteTuple
	AllowedOutcomes() []ConfirmedOutcomeRef
	DisallowedOutcomes() []ConfirmedOutcomeRef
	CustomReview() (CustomExpectationReview, bool)
	Allows(CompleteTuple) (bool, error)
}

type compilableRuling struct {
	action             Action
	registry           FieldRegistry
	selectedFields     []FieldID
	allowedTuples      []CompleteTuple
	disallowedTuples   []CompleteTuple
	allowedOutcomes    []ConfirmedOutcomeRef
	disallowedOutcomes []ConfirmedOutcomeRef
	allowedTupleKeys   map[string]struct{}
	customReview       CustomExpectationReview
}

func (compilableRuling) isCompileEligibility() {}
func (compilableRuling) isCompilableRuling()   {}
func (r compilableRuling) Action() Action      { return r.action }

func (r compilableRuling) SelectedFields() []FieldID {
	return append([]FieldID(nil), r.selectedFields...)
}

func (r compilableRuling) AllowedTuples() []CompleteTuple { return cloneTuples(r.allowedTuples) }

func (r compilableRuling) DisallowedTuples() []CompleteTuple {
	return cloneTuples(r.disallowedTuples)
}

func (r compilableRuling) AllowedOutcomes() []ConfirmedOutcomeRef {
	return append([]ConfirmedOutcomeRef(nil), r.allowedOutcomes...)
}

func (r compilableRuling) DisallowedOutcomes() []ConfirmedOutcomeRef {
	return append([]ConfirmedOutcomeRef(nil), r.disallowedOutcomes...)
}

func (r compilableRuling) CustomReview() (CustomExpectationReview, bool) {
	return r.customReview, r.action == ActionCustomExpectation && r.customReview.valid
}

// Allows performs exact complete-tuple membership and cannot synthesize a
// Cartesian product from independently observed field values.
func (r compilableRuling) Allows(candidate CompleteTuple) (bool, error) {
	fields, err := r.registry.validateTuple(candidate)
	if err != nil {
		return false, err
	}
	_, key, err := projectTuple(r.registry, r.selectedFields, fields)
	if err != nil {
		return false, err
	}
	_, ok := r.allowedTupleKeys[key] // MUTANT_U1_CHOICE_INDEPENDENT_SETS
	return ok, nil
}

// ValidatedRuling is an action plus its sealed compile-eligibility variant.
type ValidatedRuling struct {
	action      Action
	eligibility CompileEligibility
	separation  SeparationCheck
}

func (r ValidatedRuling) Action() Action                         { return r.action }
func (r ValidatedRuling) CompileEligibility() CompileEligibility { return r.eligibility }
func (r ValidatedRuling) Separation() SeparationCheck            { return r.separation }

// SeparationCheck records the finite, all-pairs obligation actually checked.
type SeparationCheck struct {
	AllowedTupleCount      int
	DisallowedTupleCount   int
	AllowedOutcomeCount    int
	DisallowedOutcomeCount int
	ComparedPairCount      int
}

// ValidateRuling validates one exact-witness action against a sealed confirmed
// universe. The caller selects only allowed outcome refs. The complement is
// always constructed internally, so omission cannot become permission.
func ValidateRuling(confirmed ConfirmedOutcomeSet, input RulingInput) (ValidatedRuling, error) {
	if !confirmed.valid || len(confirmed.byID) == 0 || len(confirmed.ordered) == 0 {
		return ValidatedRuling{}, refusal(CodeInvalidConfirmedOutcomeSet, "ruling requires a nonzero sealed confirmed outcome set")
	}
	if input.Action == "" {
		return ValidatedRuling{}, refusal(CodeMissingAction, "action must be explicit")
	}
	if len(input.Action) > maxActionBytes {
		return ValidatedRuling{}, refusal(CodeInputLimitExceeded, "action exceeds the bounded action profile")
	}
	if !utf8.ValidString(string(input.Action)) {
		return ValidatedRuling{}, refusal(CodeInvalidUTF8, "action is not valid UTF-8")
	}
	if input.SelectedFields == nil {
		return ValidatedRuling{}, refusal(CodeOmittedSelectedFields, "selected_fields must be explicit")
	}
	if input.AllowedObserved == nil {
		return ValidatedRuling{}, refusal(CodeOmittedObservedSelections, "allowed_observed must be explicit")
	}

	switch input.Action {
	case ActionRejectAll, ActionDefer, ActionRefine:
		if len(input.SelectedFields) != 0 || len(input.AllowedObserved) != 0 || input.CustomExpectation != nil || input.CustomReview.valid {
			return ValidatedRuling{}, refusal(CodeNoncompilablePredicate, "noncompilable actions cannot carry a predicate or review")
		}
		noncompilable := noncompilableRuling{action: input.Action}
		return ValidatedRuling{action: input.Action, eligibility: noncompilable}, nil
	case ActionAllowObserved, ActionCustomExpectation:
		return validateCompilable(confirmed, input)
	default:
		return ValidatedRuling{}, refusal(CodeUnknownAction, "unrecognized action "+strconv.Quote(string(input.Action)))
	}
}

func validateCompilable(confirmed ConfirmedOutcomeSet, input RulingInput) (ValidatedRuling, error) {
	if len(input.SelectedFields) == 0 { // MUTANT_U1_CHOICE_ALLOW_EMPTY_FIELDS
		return ValidatedRuling{}, refusal(CodeEmptySelectedFields, "compilable actions require a nonempty selected-field set")
	}
	selected, err := confirmed.registry.resolveSelected(input.SelectedFields)
	if err != nil {
		return ValidatedRuling{}, err
	}

	var allowedOutcomes []confirmedOutcome
	var disallowedOutcomes []confirmedOutcome
	var authoredAllowed []CompleteTuple

	switch input.Action {
	case ActionAllowObserved:
		if len(input.AllowedObserved) == 0 {
			return ValidatedRuling{}, refusal(CodeEmptyObservedSelection, "ALLOW_OBSERVED requires at least one confirmed outcome ref")
		}
		if input.CustomExpectation != nil || input.CustomReview.valid {
			return ValidatedRuling{}, refusal(CodeUnexpectedCustomExpectation, "ALLOW_OBSERVED cannot carry a custom expectation or review")
		}
		allowedOutcomes, disallowedOutcomes, err = confirmed.partition(input.AllowedObserved)
		if err != nil {
			return ValidatedRuling{}, err
		}
	case ActionCustomExpectation:
		if len(input.AllowedObserved) != 0 {
			return ValidatedRuling{}, refusal(CodeCustomExpectationCardinality, "CUSTOM_EXPECTATION cannot also select observed outcomes")
		}
		if input.CustomExpectation == nil {
			return ValidatedRuling{}, refusal(CodeCustomExpectationCardinality, "CUSTOM_EXPECTATION requires exactly one authored tuple")
		}
		if !input.CustomReview.valid {
			return ValidatedRuling{}, refusal(CodeCustomExpectationUnreviewed, "custom expectation requires a typed review fact")
		}
		if !input.CustomExpectation.validFor(confirmed, selected) {
			return ValidatedRuling{}, refusal(CodeReviewExpectationMismatch, "custom selected tuple does not bind this exact universe and field selection")
		}
		projected := CompleteTuple{Fields: input.CustomExpectation.Fields()}
		key := input.CustomExpectation.tupleKey
		if input.CustomReview.universe != confirmed.seal || input.CustomReview.selectedKey != selectedIdentityKey(selected) || input.CustomReview.expectationKey != key {
			return ValidatedRuling{}, refusal(CodeReviewExpectationMismatch, "typed review fact does not bind this exact field selection and expectation")
		}
		authoredAllowed = []CompleteTuple{projected}
		disallowedOutcomes = append([]confirmedOutcome(nil), confirmed.ordered...)
	}

	allowedTuples, allowedKeys, err := confirmed.registry.projectConfirmed(selected, allowedOutcomes)
	if err != nil {
		return ValidatedRuling{}, err
	}
	if len(authoredAllowed) == 1 {
		allowedTuples = authoredAllowed
		allowedKeys = map[string]struct{}{tupleIdentityKey(confirmed.registry, authoredAllowed[0].Fields): struct{}{}}
	}
	disallowedTuples, _, err := confirmed.registry.projectConfirmed(selected, disallowedOutcomes)
	if err != nil {
		return ValidatedRuling{}, err
	}

	pairCount := 0
	for allowedIndex, allowedTuple := range allowedTuples {
		allowedKey := tupleIdentityKey(confirmed.registry, allowedTuple.Fields)
		for disallowedIndex, disallowedTuple := range disallowedTuples {
			pairCount++
			if allowedKey == tupleIdentityKey(confirmed.registry, disallowedTuple.Fields) {
				code := CodeAmbiguousScope
				if input.Action == ActionCustomExpectation {
					code = CodeCustomExpectationAlreadySeen
				}
				err := refusal(code, "selected fields do not separate an allowed tuple from a confirmed disallowed outcome")
				err.AllowedIndex = allowedIndex
				err.DisallowedIndex = disallowedIndex
				return ValidatedRuling{}, err
			}
		}
	}

	compilable := compilableRuling{
		action:             input.Action,
		registry:           confirmed.registry.clone(),
		selectedFields:     append([]FieldID(nil), selected...),
		allowedTuples:      cloneTuples(allowedTuples),
		disallowedTuples:   cloneTuples(disallowedTuples),
		allowedOutcomes:    refsOf(allowedOutcomes),
		disallowedOutcomes: refsOf(disallowedOutcomes),
		allowedTupleKeys:   cloneStringSet(allowedKeys),
		customReview:       input.CustomReview,
	}
	return ValidatedRuling{
		action:      input.Action,
		eligibility: compilable,
		separation: SeparationCheck{
			AllowedTupleCount:      len(allowedTuples),
			DisallowedTupleCount:   len(disallowedTuples),
			AllowedOutcomeCount:    len(allowedOutcomes),
			DisallowedOutcomeCount: len(disallowedOutcomes),
			ComparedPairCount:      pairCount,
		},
	}, nil
}

func (s ConfirmedOutcomeSet) partition(selected []ConfirmedOutcomeRef) ([]confirmedOutcome, []confirmedOutcome, error) {
	if len(selected) > len(s.ordered) {
		return nil, nil, refusal(CodeInputLimitExceeded, "observed selection count exceeds the sealed confirmed universe")
	}
	allowedIDs := make(map[string]struct{}, len(selected))
	for _, selectedRef := range selected {
		outcome, exists := s.byID[selectedRef.id.text]
		if !exists || selectedRef.id.text == "" || selectedRef.universe != s.seal || selectedRef.candidate != outcome.ref.candidate || selectedRef.fingerprint != outcome.ref.fingerprint {
			return nil, nil, refusal(CodeUnconfirmedOutcomeSelection, "ALLOW_OBSERVED selected a ref outside the sealed confirmed universe")
		}
		if _, duplicate := allowedIDs[selectedRef.id.text]; duplicate {
			return nil, nil, refusal(CodeDuplicateObservedSelection, "the same confirmed outcome was selected more than once")
		}
		allowedIDs[selectedRef.id.text] = struct{}{}
	}
	allowed := make([]confirmedOutcome, 0, len(selected))
	disallowed := make([]confirmedOutcome, 0, len(s.ordered)-len(selected))
	for _, outcome := range s.ordered {
		if _, selected := allowedIDs[outcome.ref.id.text]; selected {
			allowed = append(allowed, outcome)
		} else {
			disallowed = append(disallowed, outcome)
		}
	}
	return allowed, disallowed, nil
}

func (r FieldRegistry) resolveSelected(raw []string) ([]FieldID, error) {
	if len(raw) > len(r.definitions) {
		return nil, refusal(CodeInputLimitExceeded, "selected field count exceeds the closed registry")
	}
	seen := make(map[string]struct{}, len(raw))
	for _, text := range raw {
		fieldID, err := r.Resolve(text)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[fieldID.text]; exists {
			err := refusal(CodeDuplicateSelectedField, "selected field appears more than once")
			err.FieldID = fieldID.text
			return nil, err
		}
		seen[fieldID.text] = struct{}{}
	}
	selected := make([]FieldID, 0, len(raw))
	for _, fieldID := range r.orderedIDs { // MUTANT_P07B_SELECTED_LEXICAL_ORDER
		if _, present := seen[fieldID]; present {
			selected = append(selected, FieldID{text: fieldID})
		}
	}
	return selected, nil
}

func (r FieldRegistry) validateSelectedTuple(selected []FieldID, raw []FieldValue) (map[string]ExactValue, error) {
	if raw == nil {
		return nil, refusal(CodeIncompleteTuple, "selected tuple fields were omitted")
	}
	if len(raw) != len(selected) {
		return nil, refusal(CodeIncompleteTuple, "selected tuple must contain exactly the selected fields")
	}
	allowed := make(map[string]struct{}, len(selected))
	for _, field := range selected {
		allowed[field.text] = struct{}{}
	}
	values := make(map[string]ExactValue, len(raw))
	retainedBytes := 0
	for _, field := range raw {
		fieldID, err := r.Resolve(field.FieldID)
		if err != nil {
			return nil, err
		}
		if _, selectedField := allowed[fieldID.text]; !selectedField { // MUTANT_P07B_CUSTOM_UNSELECTED_FIELD
			err := refusal(CodeIncompleteTuple, "custom expectation contains an unselected field")
			err.FieldID = fieldID.text
			return nil, err
		}
		if _, duplicate := values[fieldID.text]; duplicate {
			err := refusal(CodeDuplicateTupleField, "selected tuple contains the field more than once")
			err.FieldID = fieldID.text
			return nil, err
		}
		if err := validateValue(r.definitions[fieldID.text], field.Value); err != nil {
			return nil, err
		}
		retainedBytes += exactValueRetainedBytes(field.Value)
		if retainedBytes > maxTupleRetainedBytes {
			return nil, refusal(CodeInputLimitExceeded, "selected tuple exceeds the retained canonical-byte ceiling")
		}
		values[fieldID.text] = cloneExactValue(field.Value)
	}
	for _, field := range selected {
		if _, present := values[field.text]; !present {
			err := refusal(CodeIncompleteTuple, "selected tuple omits a selected field")
			err.FieldID = field.text
			return nil, err
		}
	}
	selectedOrder := make([]string, len(selected))
	for index, field := range selected {
		selectedOrder[index] = field.text
	}
	if err := r.validatePortableValues(selectedOrder, values); err != nil {
		return nil, err
	}
	if err := r.validatePortableExpectation(selectedOrder, values, true); err != nil {
		return nil, err
	}
	return values, nil
}

func (r FieldRegistry) validateTuple(tuple CompleteTuple) (map[string]ExactValue, error) {
	if tuple.Fields == nil {
		return nil, refusal(CodeIncompleteTuple, "tuple fields were omitted")
	}
	if len(tuple.Fields) > len(r.definitions) {
		return nil, refusal(CodeInputLimitExceeded, "tuple field count exceeds the closed registry")
	}
	fields := make(map[string]ExactValue, len(tuple.Fields))
	retainedBytes := 0
	for _, field := range tuple.Fields {
		fieldID, err := r.Resolve(field.FieldID)
		if err != nil {
			return nil, err
		}
		if _, exists := fields[fieldID.text]; exists {
			err := refusal(CodeDuplicateTupleField, "tuple contains the field more than once")
			err.FieldID = fieldID.text
			return nil, err
		}
		definition := r.definitions[fieldID.text]
		if err := validateValue(definition, field.Value); err != nil {
			return nil, err
		}
		retainedBytes += exactValueRetainedBytes(field.Value)
		if retainedBytes > maxTupleRetainedBytes {
			return nil, refusal(CodeInputLimitExceeded, "tuple exceeds the retained canonical-byte ceiling")
		}
		fields[fieldID.text] = cloneExactValue(field.Value)
	}
	if len(fields) != len(r.definitions) {
		for _, fieldID := range r.orderedIDs {
			if _, exists := fields[fieldID]; !exists {
				err := refusal(CodeIncompleteTuple, "complete tuple omits a field from the closed registry")
				err.FieldID = fieldID
				return nil, err
			}
		}
	}
	if err := r.validatePortableValues(r.orderedIDs, fields); err != nil {
		return nil, err
	}
	if err := r.validatePortableExpectation(r.orderedIDs, fields, false); err != nil {
		return nil, err
	}
	return fields, nil
}

func (r FieldRegistry) validatePortableValues(order []string, values map[string]ExactValue) error {
	if r.mode != fieldRegistryPortable {
		return nil
	}
	portableValues := make([]portablevalue.Value, 0, len(order))
	for _, fieldID := range order {
		exact, exists := values[fieldID]
		if !exists {
			continue
		}
		portable, err := portableValueFromExact(exact)
		if err != nil || !portable.Valid() {
			code := CodeInvalidFieldType
			if portablevalue.IsCode(err, portablevalue.CodeLimitExceeded) {
				code = CodeInputLimitExceeded
			}
			refused := refusal(code, "portable tuple value could not be reconstructed exactly")
			refused.FieldID = fieldID
			return refused
		}
		if r.sourceKinds[fieldID] == "CANONICAL_JSON_OBJECT" && portable.Tag() == portablevalue.TagCanonicalJSON {
			canonical, _ := portable.CanonicalJSONBytes()
			parsed, parseErr := canon.Parse(canonical)
			if parseErr != nil || parsed.Kind() != canon.KindObject {
				refused := refusal(CodeInvalidFieldType, "portable value violates the adapter profile source kind")
				refused.FieldID = fieldID
				return refused
			}
		}
		portableValues = append(portableValues, portable)
	}
	if err := portablevalue.ValidateTuple(portableValues); err != nil {
		code := CodeInvalidFieldType
		if portablevalue.IsCode(err, portablevalue.CodeLimitExceeded) {
			code = CodeInputLimitExceeded
		}
		return refusal(code, "portable tuple exceeds the closed value profile")
	}
	return nil
}

func (r FieldRegistry) validatePortableExpectation(order []string, values map[string]ExactValue, custom bool) error {
	if r.mode != fieldRegistryPortable {
		return nil
	}
	if !r.expectationDomain.Valid() || r.expectationDomain.ProfileDigest() != r.profileDigest {
		return refusal(CodeInvalidFieldRegistry, "portable field registry lost its exact expectation-domain authority")
	}
	selected := make([]projectiontranslate.SelectedValue, 0, len(order))
	for _, fieldID := range order {
		exact, exists := values[fieldID]
		if !exists {
			continue
		}
		portable, err := portableValueFromExact(exact)
		if err != nil || !portable.Valid() {
			refused := refusal(CodeInvalidFieldType, "portable expectation value could not be reconstructed exactly")
			refused.FieldID = fieldID
			return refused
		}
		selected = append(selected, projectiontranslate.SelectedValue{FieldID: fieldID, Value: portable})
	}
	if err := r.expectationDomain.ValidateSelected(selected); err != nil {
		if custom && projectiontranslate.IsCode(err, projectiontranslate.CodeCustomExpectationNotRealizable) {
			return refusal(CodeCustomExpectationNotAdapterRealizable, "custom selected tuple has no complete adapter-projection extension")
		}
		return refusal(CodeInvalidFieldRegistry, "portable tuple disagrees with its sealed expectation domain: "+err.Error())
	}
	return nil
}

func exactValueRetainedBytes(value ExactValue) int {
	return len(value.text) + len(value.opaque) + len(value.canonical)
}

func validateValue(definition FieldDefinition, value ExactValue) error {
	if value.tag == ValueString && len(value.text) > maxExactStringBytes {
		err := refusal(CodeInputLimitExceeded, "string value exceeds the exact-value byte limit")
		err.FieldID = definition.ID
		return err
	}
	valid := false
	switch value.tag {
	case ValueMissing:
		valid = definition.AllowMissing && value.text == "" && !value.boolean && len(value.opaque) == 0 && len(value.canonical) == 0
	case ValueNull:
		valid = definition.AllowNull && value.text == "" && !value.boolean && len(value.opaque) == 0 && len(value.canonical) == 0
	case ValueString:
		valid = definition.Type == FieldString && !value.boolean && utf8.ValidString(value.text) && len(value.text) <= maxExactStringBytes && len(value.opaque) == 0 && len(value.canonical) == 0
	case ValueInteger:
		_, err := canon.IntegerFromString(value.text)
		valid = definition.Type == FieldInteger && !value.boolean && err == nil && len(value.opaque) == 0 && len(value.canonical) == 0
	case ValueBoolean:
		valid = definition.Type == FieldBoolean && value.text == "" && len(value.opaque) == 0 && len(value.canonical) == 0
	case ValueBytes:
		portable, err := portablevalue.Bytes(value.opaque)
		valid = definition.Type == FieldBytes && err == nil && value.text == "" && !value.boolean && len(value.canonical) == 0 && portable.Valid()
	case ValueOrderedStringList:
		portable, err := portablevalue.OrderedStringListFromCanonical(value.canonical)
		valid = definition.Type == FieldOrderedStringList && err == nil && value.text == "" && !value.boolean && len(value.opaque) == 0 && portable.Valid()
	case ValueCanonicalJSON:
		if definition.Type == FieldCanonicalJSON && !value.boolean && len(value.opaque) == 0 && len(value.canonical) > 0 {
			parsed, err := canon.Parse(value.canonical)
			checked, checkedErr := parsed.CanonicalChecked()
			if err == nil && checkedErr == nil && parsed.Kind() != canon.KindNull && bytes.Equal(checked, value.canonical) {
				digest, digestErr := canon.DigestValue("ChoiceCanonicalJSON", parsed)
				valid = digestErr == nil && digest.String() == value.text
			}
		}
	}
	if !valid {
		err := refusal(CodeInvalidFieldType, "value tag/payload is invalid for declared field type")
		err.FieldID = definition.ID
		return err
	}
	return nil
}

func (r FieldRegistry) projectConfirmed(selected []FieldID, outcomes []confirmedOutcome) ([]CompleteTuple, map[string]struct{}, error) {
	projected := make([]CompleteTuple, 0, len(outcomes))
	keys := make(map[string]struct{}, len(outcomes))
	for _, outcome := range outcomes {
		fields, err := r.validateTuple(outcome.tuple)
		if err != nil {
			return nil, nil, err
		}
		tuple, key, err := projectTuple(r, selected, fields)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicateProjection := keys[key]; duplicateProjection {
			continue
		}
		keys[key] = struct{}{}
		projected = append(projected, tuple)
	}
	sort.Slice(projected, func(i, j int) bool {
		return tupleIdentityKey(r, projected[i].Fields) < tupleIdentityKey(r, projected[j].Fields)
	})
	return projected, keys, nil
}

func projectTuple(registry FieldRegistry, selected []FieldID, fields map[string]ExactValue) (CompleteTuple, string, error) {
	projected := CompleteTuple{Fields: make([]FieldValue, 0, len(selected))}
	for _, fieldID := range selected {
		value, exists := fields[fieldID.text]
		if !exists {
			err := refusal(CodeIncompleteTuple, "tuple must explicitly carry selected fields; use MISSING for an observed absence")
			err.FieldID = fieldID.text
			return CompleteTuple{}, "", err
		}
		projected.Fields = append(projected.Fields, FieldValue{FieldID: fieldID.text, Value: cloneExactValue(value)})
	}
	return projected, tupleIdentityKey(registry, projected.Fields), nil
}

func selectedIdentityKey(selected []FieldID) string {
	var builder strings.Builder
	for _, field := range selected {
		builder.WriteString(lengthPrefix(field.text))
	}
	return builder.String()
}

func tupleIdentityKey(registry FieldRegistry, fields []FieldValue) string {
	var builder strings.Builder
	for _, field := range fields {
		builder.WriteString(lengthPrefix(field.FieldID))
		builder.WriteString(lengthPrefix(field.Value.identityKey(registry.mode)))
	}
	return builder.String()
}

func lengthPrefix(value string) string { return strconv.Itoa(len(value)) + ":" + value }

func cloneExactValue(value ExactValue) ExactValue {
	value.opaque = append([]byte(nil), value.opaque...)
	value.canonical = append([]byte(nil), value.canonical...)
	return value
}

func cloneTuple(input CompleteTuple) CompleteTuple {
	result := CompleteTuple{Fields: make([]FieldValue, len(input.Fields))}
	for index, field := range input.Fields {
		result.Fields[index] = FieldValue{FieldID: field.FieldID, Value: cloneExactValue(field.Value)}
	}
	return result
}

func cloneTuples(input []CompleteTuple) []CompleteTuple {
	result := make([]CompleteTuple, len(input))
	for index := range input {
		result[index] = cloneTuple(input[index])
	}
	return result
}

func cloneStringSet(input map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(input))
	for key := range input {
		result[key] = struct{}{}
	}
	return result
}

func refsOf(outcomes []confirmedOutcome) []ConfirmedOutcomeRef {
	refs := make([]ConfirmedOutcomeRef, len(outcomes))
	for index := range outcomes {
		refs[index] = outcomes[index].ref
	}
	return refs
}

func (r FieldRegistry) clone() FieldRegistry {
	clone := FieldRegistry{
		definitions:                make(map[string]FieldDefinition, len(r.definitions)),
		orderedIDs:                 append([]string(nil), r.orderedIDs...),
		sourceKinds:                make(map[string]string, len(r.sourceKinds)),
		fieldRegistryDigest:        r.fieldRegistryDigest,
		projectionDefinitionDigest: r.projectionDefinitionDigest,
		profileDigest:              r.profileDigest,
		expectationDomain:          r.expectationDomain,
		mode:                       r.mode,
	}
	for id, definition := range r.definitions {
		definition.Path = append([]string(nil), definition.Path...)
		clone.definitions[id] = definition
	}
	for id, sourceKind := range r.sourceKinds {
		clone.sourceKinds[id] = sourceKind
	}
	return clone
}
