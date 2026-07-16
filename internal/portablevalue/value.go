// Package portablevalue owns Countershape's closed, adapter-neutral exact
// value algebra. Values are immutable, bounded, and distinguish identity by
// tag plus exact payload bytes. The zero Value is intentionally invalid.
package portablevalue

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
)

// Tag is the closed portable-value sum.
type Tag string

const (
	TagMissing           Tag = "MISSING"
	TagNull              Tag = "NULL"
	TagBoolean           Tag = "BOOLEAN"
	TagInteger           Tag = "INTEGER"
	TagString            Tag = "STRING"
	TagBytes             Tag = "BYTES"
	TagOrderedStringList Tag = "ORDERED_STRING_LIST"
	TagCanonicalJSON     Tag = "CANONICAL_JSON"
)

// Code is a stable refusal class. Detail is diagnostic and not identity.
type Code string

const (
	CodeInvalidTag       Code = "INVALID_TAG"
	CodeInvalidUTF8      Code = "INVALID_UTF8"
	CodeInvalidInteger   Code = "INVALID_INTEGER"
	CodeNoncanonicalJSON Code = "NONCANONICAL_JSON"
	CodeLimitExceeded    Code = "LIMIT_EXCEEDED"
	CodeInvalidTuple     Code = "INVALID_TUPLE"
)

// Limits are deliberately below the durable one-MiB canonical object ceiling
// after outer field IDs and base64 expansion are added by owning artifacts.
const (
	MaxStringBytes        = 64 * 1024
	MaxBytesValueBytes    = 64 * 1024
	MaxListMembers        = 256
	MaxListMemberBytes    = 64 * 1024
	MaxListAggregateBytes = 64 * 1024
	MaxListCanonicalBytes = 64 * 1024
	MaxCanonicalJSONBytes = 64 * 1024
	MaxTupleFields        = 64
	MaxTupleRetainedBytes = 256 * 1024
	MaxTupleEncodedBytes  = 384 * 1024

	// This covers a 128-byte field ID, the fixed four-slot compatibility value
	// object, the containing field object, commas, quotes, and escaping slack.
	// The payload itself is measured exactly below.
	CompatibilityFieldOverheadBytes = 512
)

// Error is a bounded structural refusal.
type Error struct {
	Code   Code
	Detail string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Detail == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Detail
}

// IsCode reports whether err has the exact portable refusal code.
func IsCode(err error, code Code) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code Code, detail string) *Error { return &Error{Code: code, Detail: detail} }

// Value is immutable. Composite input and output always cross a copy boundary.
type Value struct {
	tag       Tag
	boolean   bool
	integer   string
	text      string
	bytes     []byte
	canonical []byte
}

// Missing constructs the tagged absence value.
func Missing() Value { return Value{tag: TagMissing} }

// Null constructs the tagged present-null value.
func Null() Value { return Value{tag: TagNull} }

// Boolean constructs an exact Boolean value.
func Boolean(value bool) Value { return Value{tag: TagBoolean, boolean: value} }

// Integer accepts only canon's safe-range canonical decimal spelling.
func Integer(canonical string) (Value, error) {
	if len(canonical) > 32 {
		return Value{}, refuse(CodeLimitExceeded, "integer spelling exceeds the byte ceiling")
	}
	if !utf8.ValidString(canonical) {
		return Value{}, refuse(CodeInvalidUTF8, "integer spelling is not valid UTF-8")
	}
	parsed, err := canon.IntegerFromString(canonical)
	if err != nil {
		return Value{}, refuse(CodeInvalidInteger, "integer is not one safe canonical decimal")
	}
	integer, ok := parsed.Int64()
	if !ok || strconv.FormatInt(integer, 10) != canonical {
		return Value{}, refuse(CodeInvalidInteger, "integer canonicalization changed its spelling")
	}
	return Value{tag: TagInteger, integer: canonical}, nil
}

// String constructs an exact UTF-8 string, including the empty string.
func String(value string) (Value, error) {
	if len(value) > MaxStringBytes {
		return Value{}, refuse(CodeLimitExceeded, "string exceeds the byte ceiling")
	}
	if !utf8.ValidString(value) {
		return Value{}, refuse(CodeInvalidUTF8, "string is not valid UTF-8")
	}
	return Value{tag: TagString, text: value}, nil
}

// Bytes constructs exact opaque bytes, including an empty byte string.
func Bytes(value []byte) (Value, error) {
	if len(value) > MaxBytesValueBytes {
		return Value{}, refuse(CodeLimitExceeded, "byte value exceeds the byte ceiling")
	}
	return Value{tag: TagBytes, bytes: append([]byte(nil), value...)}, nil
}

// OrderedStringList preserves order, duplicates, empty lists, and empty
// members. UTF-8 and both per-member and aggregate byte ceilings are strict.
func OrderedStringList(values []string) (Value, error) {
	if values == nil {
		return Value{}, refuse(CodeInvalidTuple, "ordered string list must be an explicit list")
	}
	if len(values) > MaxListMembers {
		return Value{}, refuse(CodeLimitExceeded, "ordered string list exceeds the member ceiling")
	}
	retained := 0
	for index, value := range values {
		if len(value) > MaxListMemberBytes {
			return Value{}, refuse(CodeLimitExceeded, fmt.Sprintf("ordered string list member %d exceeds the byte ceiling", index))
		}
		if !utf8.ValidString(value) {
			return Value{}, refuse(CodeInvalidUTF8, fmt.Sprintf("ordered string list member %d is not valid UTF-8", index))
		}
		retained += len(value)
		if retained > MaxListAggregateBytes {
			return Value{}, refuse(CodeLimitExceeded, "ordered string list exceeds the aggregate byte ceiling")
		}
	}
	canonical, err := canonicalStringList(values)
	if err != nil {
		return Value{}, err
	}
	if len(canonical) > MaxListCanonicalBytes {
		return Value{}, refuse(CodeLimitExceeded, "ordered string list exceeds the canonical encoded-byte ceiling")
	}
	return Value{tag: TagOrderedStringList, canonical: canonical}, nil
}

// OrderedStringListFromCanonical admits only the exact canonical compatibility
// bytes produced by OrderedStringList. It exists for strict durable-wire
// reconstruction; it never normalizes a merely equivalent JSON array.
func OrderedStringListFromCanonical(exact []byte) (Value, error) {
	members, ok := canonicalStringListMembers(exact)
	if !ok {
		return Value{}, refuse(CodeInvalidTuple, "ordered string list bytes are not one exact canonical string array")
	}
	rebuilt, err := OrderedStringList(members)
	if err != nil || !bytes.Equal(rebuilt.canonical, exact) {
		return Value{}, refuse(CodeInvalidTuple, "ordered string list bytes do not reconstruct exactly")
	}
	return rebuilt, nil
}

// CanonicalJSON admits exact bytes only. It retains the bytes themselves; a
// digest is never accepted as a substitute for the JSON preimage.
func CanonicalJSON(exact []byte) (Value, error) {
	if len(exact) == 0 {
		return Value{}, refuse(CodeNoncanonicalJSON, "canonical JSON is empty")
	}
	if len(exact) > MaxCanonicalJSONBytes {
		return Value{}, refuse(CodeLimitExceeded, "canonical JSON exceeds the byte ceiling")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return Value{}, refuse(CodeNoncanonicalJSON, "canonical JSON does not parse under the closed profile")
	}
	checked, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(checked, exact) {
		return Value{}, refuse(CodeNoncanonicalJSON, "canonical JSON bytes are not exact")
	}
	return Value{tag: TagCanonicalJSON, canonical: append([]byte(nil), exact...)}, nil
}

// Tag returns the closed sum discriminator.
func (v Value) Tag() Tag { return v.tag }

// Valid reports whether the value could have come from a public constructor.
func (v Value) Valid() bool {
	switch v.tag {
	case TagMissing, TagNull:
		return !v.boolean && v.integer == "" && v.text == "" && v.bytes == nil && v.canonical == nil
	case TagBoolean:
		return v.integer == "" && v.text == "" && v.bytes == nil && v.canonical == nil
	case TagInteger:
		rebuilt, err := Integer(v.integer)
		return err == nil && rebuilt.integer == v.integer && !v.boolean && v.text == "" && v.bytes == nil && v.canonical == nil
	case TagString:
		rebuilt, err := String(v.text)
		return err == nil && rebuilt.text == v.text && !v.boolean && v.integer == "" && v.bytes == nil && v.canonical == nil
	case TagBytes:
		return len(v.bytes) <= MaxBytesValueBytes && !v.boolean && v.integer == "" && v.text == "" && v.canonical == nil
	case TagOrderedStringList:
		members, ok := canonicalStringListMembers(v.canonical)
		if !ok {
			return false
		}
		rebuilt, err := OrderedStringList(members)
		return err == nil && bytes.Equal(rebuilt.canonical, v.canonical) && !v.boolean && v.integer == "" && v.text == "" && v.bytes == nil
	case TagCanonicalJSON:
		rebuilt, err := CanonicalJSON(v.canonical)
		return err == nil && bytes.Equal(rebuilt.canonical, v.canonical) && !v.boolean && v.integer == "" && v.text == "" && v.bytes == nil
	default:
		return false
	}
}

func (v Value) BooleanValue() (bool, bool)  { return v.boolean, v.tag == TagBoolean }
func (v Value) IntegerText() (string, bool) { return v.integer, v.tag == TagInteger }
func (v Value) StringText() (string, bool)  { return v.text, v.tag == TagString }

func (v Value) BytesValue() ([]byte, bool) {
	if v.tag != TagBytes {
		return nil, false
	}
	return append([]byte(nil), v.bytes...), true
}

func (v Value) OrderedStrings() ([]string, bool) {
	if v.tag != TagOrderedStringList {
		return nil, false
	}
	return canonicalStringListMembers(v.canonical)
}

func (v Value) CanonicalJSONBytes() ([]byte, bool) {
	if v.tag != TagCanonicalJSON {
		return nil, false
	}
	return append([]byte(nil), v.canonical...), true
}

// CanonicalStringListBytes returns strict canonical JSON array bytes for the
// ordered list. It is the compatibility payload used by Choice's historical
// four-slot exact-value wire.
func (v Value) CanonicalStringListBytes() ([]byte, bool) {
	if v.tag != TagOrderedStringList {
		return nil, false
	}
	return append([]byte(nil), v.canonical...), true
}

// Equal compares exact tagged identity, never display text or a digest alone.
func (v Value) Equal(other Value) bool {
	if v.tag == "" || v.tag != other.tag {
		return false
	}
	switch v.tag {
	case TagMissing, TagNull:
		return true
	case TagBoolean:
		return v.boolean == other.boolean
	case TagInteger:
		return v.integer == other.integer
	case TagString:
		return v.text == other.text
	case TagBytes:
		return bytes.Equal(v.bytes, other.bytes)
	case TagOrderedStringList, TagCanonicalJSON:
		return bytes.Equal(v.canonical, other.canonical)
	default:
		return false
	}
}

// IdentityBytes returns an injective bounded binary encoding of tag and exact
// payload. It is internal semantic material, not a public wire format.
func (v Value) IdentityBytes() []byte {
	if !v.Valid() {
		return nil
	}
	result := make([]byte, 0, v.RetainedBytes()+32)
	result = appendLength(result, []byte(v.tag))
	switch v.tag {
	case TagBoolean:
		if v.boolean {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	case TagInteger:
		result = appendLength(result, []byte(v.integer))
	case TagString:
		result = appendLength(result, []byte(v.text))
	case TagBytes:
		result = appendLength(result, v.bytes)
	case TagOrderedStringList:
		result = appendLength(result, v.canonical)
	case TagCanonicalJSON:
		result = appendLength(result, v.canonical)
	}
	return result
}

// RetainedBytes is semantic payload accounting used by the in-memory tuple
// cap. It deliberately does not pretend to count Go allocation headers.
func (v Value) RetainedBytes() int {
	return len(v.tag) + len(v.integer) + len(v.text) + len(v.bytes) + len(v.canonical)
}

// CompatibilityEncodedBytes returns the exact payload contribution after the
// historical Choice compatibility encoding: strings are JSON-escaped and
// opaque/canonical bytes use padded base64. Container overhead is accounted
// separately by ValidateTuple.
func (v Value) CompatibilityEncodedBytes() int {
	if !v.Valid() {
		return 0
	}
	switch v.tag {
	case TagMissing, TagNull:
		return 0
	case TagBoolean:
		return 5
	case TagInteger:
		return len(v.integer) + 2
	case TagString:
		value, err := canon.String(v.text)
		if err != nil {
			return 0
		}
		exact, err := value.CanonicalChecked()
		if err != nil {
			return 0
		}
		return len(exact)
	case TagBytes:
		return base64.StdEncoding.EncodedLen(len(v.bytes)) + 2
	case TagOrderedStringList, TagCanonicalJSON:
		return base64.StdEncoding.EncodedLen(len(v.canonical)) + 2
	default:
		return 0
	}
}

// ValidateTuple enforces the closed aggregate field and retained-byte limits.
func ValidateTuple(values []Value) error {
	if len(values) == 0 || len(values) > MaxTupleFields {
		return refuse(CodeInvalidTuple, "tuple must contain a bounded nonempty field set")
	}
	retained := 0
	encoded := 0
	for index, value := range values {
		if !value.Valid() {
			return refuse(CodeInvalidTag, fmt.Sprintf("tuple value %d is invalid", index))
		}
		retained += value.RetainedBytes()
		if retained > MaxTupleRetainedBytes {
			return refuse(CodeLimitExceeded, "tuple exceeds the retained-byte ceiling")
		}
		encoded += CompatibilityFieldOverheadBytes + value.CompatibilityEncodedBytes()
		if encoded > MaxTupleEncodedBytes {
			return refuse(CodeLimitExceeded, "tuple exceeds the compatibility encoded-byte ceiling")
		}
	}
	return nil
}

func appendLength(target, payload []byte) []byte {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(payload)))
	target = append(target, length[:]...)
	return append(target, payload...)
}

func canonicalStringList(values []string) ([]byte, error) {
	elements := make([]canon.Value, len(values))
	for index, member := range values {
		element, err := canon.String(member)
		if err != nil {
			return nil, refuse(CodeLimitExceeded, fmt.Sprintf("ordered string list member %d is outside the canonical profile", index))
		}
		elements[index] = element
	}
	array, err := canon.Array(elements...)
	if err != nil {
		return nil, refuse(CodeLimitExceeded, "ordered string list is outside the canonical aggregate profile")
	}
	exact, err := array.CanonicalChecked()
	if err != nil {
		return nil, refuse(CodeLimitExceeded, "ordered string list could not be encoded canonically")
	}
	return exact, nil
}

func canonicalStringListMembers(exact []byte) ([]string, bool) {
	if len(exact) == 0 || len(exact) > MaxListCanonicalBytes {
		return nil, false
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return nil, false
	}
	checked, err := value.CanonicalChecked()
	elements, array := value.Elements()
	if err != nil || !bytes.Equal(checked, exact) || !array || len(elements) > MaxListMembers {
		return nil, false
	}
	result := make([]string, len(elements))
	retained := 0
	for index, element := range elements {
		member, ok := element.Text()
		if !ok || len(member) > MaxListMemberBytes {
			return nil, false
		}
		retained += len(member)
		if retained > MaxListAggregateBytes {
			return nil, false
		}
		result[index] = member
	}
	return result, true
}
