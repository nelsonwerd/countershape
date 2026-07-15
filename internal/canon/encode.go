package canon

import (
	"bytes"
	"strconv"
	"unicode/utf8"
)

var lowerHex = "0123456789abcdef"

// Canonicalize strictly parses input and returns deterministic canonical bytes.
func Canonicalize(input []byte) ([]byte, error) {
	value, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return value.CanonicalChecked()
}

// Canonical returns a new byte slice containing v's deterministic encoding.
// Project constructors and Parse only create valid values. If package memory is
// corrupted into an impossible kind, Canonical panics rather than silently
// mapping that state onto a valid identity. Boundary code that admits an
// untrusted Value should use CanonicalChecked and propagate its typed refusal.
func (v Value) Canonical() []byte {
	canonical, err := v.CanonicalChecked()
	if err != nil {
		panic(err)
	}
	return canonical
}

// CanonicalChecked validates closed-kind and resource invariants while
// encoding. It is the fail-closed form used by digest and typed authorities.
func (v Value) CanonicalChecked() ([]byte, error) {
	budget := traversalBudget{}
	return v.appendCanonicalChecked(nil, &budget, 0)
}

// Equal reports exact equality under the v1 canonical profile.
func (v Value) Equal(other Value) bool {
	return bytes.Equal(v.Canonical(), other.Canonical())
}

func (v Value) appendCanonicalChecked(output []byte, budget *traversalBudget, depth int) ([]byte, error) {
	if depth > maxNestingDepth {
		return nil, refusal(CodeDepthExceeded, UnknownOffset, "value exceeds the maximum nesting depth")
	}
	if err := budget.enter(); err != nil {
		return nil, err
	}
	switch v.kind {
	case KindNull:
		return append(output, "null"...), nil
	case KindBoolean:
		return strconv.AppendBool(output, v.boolean), nil
	case KindInteger:
		if v.integer < MinSafeInteger || v.integer > MaxSafeInteger {
			return nil, refusal(CodeUnsafeInteger, UnknownOffset, "stored integer exceeds the exact interoperability range")
		}
		return strconv.AppendInt(output, v.integer, 10), nil
	case KindString:
		if !utf8.ValidString(v.text) {
			return nil, refusal(CodeInvalidUTF8, UnknownOffset, "stored string is not valid UTF-8")
		}
		if err := budget.addString(v.text); err != nil {
			return nil, err
		}
		return appendCanonicalString(output, v.text), nil
	case KindArray:
		if len(v.array) > MaxContainerMembers {
			return nil, refusal(CodeMemberLimit, UnknownOffset, "array exceeds the v1 member ceiling")
		}
		output = append(output, '[')
		for index := range v.array {
			if index > 0 {
				output = append(output, ',')
			}
			var err error
			output, err = v.array[index].appendCanonicalChecked(output, budget, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return append(output, ']'), nil
	case KindObject:
		if len(v.object) > MaxContainerMembers {
			return nil, refusal(CodeMemberLimit, UnknownOffset, "object exceeds the v1 member ceiling")
		}
		output = append(output, '{')
		previous := ""
		for index := range v.object {
			name := v.object[index].Name
			if !utf8.ValidString(name) {
				return nil, refusal(CodeInvalidUTF8, UnknownOffset, "stored object name is not valid UTF-8")
			}
			if index > 0 && name <= previous {
				return nil, refusal(CodeInvalidValue, UnknownOffset, "stored object members are duplicated or not in canonical order")
			}
			previous = name
			if err := budget.addString(name); err != nil {
				return nil, err
			}
			if index > 0 {
				output = append(output, ',')
			}
			output = appendCanonicalString(output, name)
			output = append(output, ':')
			var err error
			output, err = v.object[index].Value.appendCanonicalChecked(output, budget, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return append(output, '}'), nil
	default:
		return nil, refusal(CodeInvalidKind, UnknownOffset, "stored value has an impossible kind")
	}
}

func appendCanonicalString(output []byte, value string) []byte {
	output = append(output, '"')
	for offset := 0; offset < len(value); {
		b := value[offset]
		switch b {
		case '"', '\\':
			output = append(output, '\\', b)
			offset++
		case '\b':
			output = append(output, '\\', 'b')
			offset++
		case '\f':
			output = append(output, '\\', 'f')
			offset++
		case '\n':
			output = append(output, '\\', 'n')
			offset++
		case '\r':
			output = append(output, '\\', 'r')
			offset++
		case '\t':
			output = append(output, '\\', 't')
			offset++
		default:
			if b < 0x20 {
				output = append(output, '\\', 'u', '0', '0', lowerHex[b>>4], lowerHex[b&0x0f])
				offset++
				continue
			}
			if b < utf8.RuneSelf {
				output = append(output, b)
				offset++
				continue
			}
			_, size := utf8.DecodeRuneInString(value[offset:])
			output = append(output, value[offset:offset+size]...)
			offset += size
		}
	}
	return append(output, '"')
}
