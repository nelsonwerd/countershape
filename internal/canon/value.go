package canon

import (
	"sort"
	"strconv"
	"unicode/utf8"
)

// ValueKind is the closed set admitted by Countershape's v1 canonical JSON
// profile. Numbers are safe-range integers only.
type ValueKind uint8

const (
	KindNull ValueKind = iota
	KindBoolean
	KindInteger
	KindString
	KindArray
	KindObject
)

func (k ValueKind) String() string {
	switch k {
	case KindNull:
		return "null"
	case KindBoolean:
		return "boolean"
	case KindInteger:
		return "integer"
	case KindString:
		return "string"
	case KindArray:
		return "array"
	case KindObject:
		return "object"
	default:
		return "unknown"
	}
}

// Value is an immutable canonical value. Its zero value is canonical null.
// Composite storage is never exposed without a copy.
type Value struct {
	kind    ValueKind
	boolean bool
	integer int64
	text    string
	array   []Value
	object  []Member
}

// Member is one decoded object name and immutable value. Object validates
// names, refuses duplicates, and stores members in UTF-8 byte order.
type Member struct {
	Name  string
	Value Value
}

func Null() Value {
	return Value{}
}

func Bool(value bool) Value {
	return Value{kind: KindBoolean, boolean: value}
}

// Integer constructs a safe interoperability-range integer.
func Integer(value int64) (Value, error) {
	if value < MinSafeInteger || value > MaxSafeInteger {
		return Value{}, refusal(CodeUnsafeInteger, UnknownOffset, "integer exceeds the exact interoperability range")
	}
	return Value{kind: KindInteger, integer: value}, nil
}

// IntegerFromString constructs an integer from its strict canonical decimal
// spelling. It is useful where an outer schema deliberately carries integer
// values as strings.
func IntegerFromString(canonical string) (Value, error) {
	if offset := firstInvalidUTF8([]byte(canonical)); offset >= 0 {
		return Value{}, refusal(CodeInvalidUTF8, offset, "integer spelling is not valid UTF-8")
	}
	if err := validateIntegerLexeme([]byte(canonical), 0); err != nil {
		return Value{}, err
	}
	value, err := strconv.ParseInt(canonical, 10, 64)
	if err != nil {
		return Value{}, refusal(CodeUnsafeInteger, 0, "integer exceeds the exact interoperability range")
	}
	return Value{kind: KindInteger, integer: value}, nil
}

func String(value string) (Value, error) {
	if offset := firstInvalidUTF8([]byte(value)); offset >= 0 {
		return Value{}, refusal(CodeInvalidUTF8, offset, "string is not valid UTF-8")
	}
	if len(value) > MaxTypedStringBytes {
		return Value{}, refusal(CodeTypedByteLimit, UnknownOffset, "string exceeds the aggregate typed string-byte ceiling")
	}
	return Value{kind: KindString, text: value}, nil
}

func Array(values ...Value) (Value, error) {
	if len(values) > MaxContainerMembers {
		return Value{}, refusal(CodeMemberLimit, UnknownOffset, "array exceeds the v1 member ceiling")
	}
	value := Value{kind: KindArray, array: append([]Value(nil), values...)}
	// MUTANT_U1_CANON_BYPASS_PROGRAMMATIC_ARRAY_PROFILE: aggregate resource checks are constructor authority.
	if _, err := value.CanonicalChecked(); err != nil {
		return Value{}, err
	}
	return value, nil
}

func Object(members ...Member) (Value, error) {
	if len(members) > MaxContainerMembers {
		return Value{}, refusal(CodeMemberLimit, UnknownOffset, "object exceeds the v1 member ceiling")
	}
	seen := make(map[string]bool, len(members))
	for index := range members {
		if err := admitObjectName(seen, members[index].Name, UnknownOffset); err != nil {
			return Value{}, err
		}
	}
	value := orderedObject(members)
	if _, err := value.CanonicalChecked(); err != nil {
		return Value{}, err
	}
	return value, nil
}

func admitObjectName(seen map[string]bool, name string, offset int) error {
	if !utf8.ValidString(name) {
		return refusal(CodeInvalidUTF8, offset, "object name is not valid UTF-8")
	}
	if duplicate := seen[name]; duplicate { // MUTANT_U1_CANON_BYPASS_DUPLICATE_KEY
		return refusal(CodeDuplicateKey, offset, "decoded object name occurs more than once")
	}
	seen[name] = true
	return nil
}

func orderedObject(members []Member) Value {
	ordered := append([]Member(nil), members...)
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].Name < ordered[right].Name
	})
	return Value{kind: KindObject, object: ordered}
}

func (v Value) Kind() ValueKind {
	return v.kind
}

func (v Value) Boolean() (bool, bool) {
	return v.boolean, v.kind == KindBoolean
}

func (v Value) Int64() (int64, bool) {
	return v.integer, v.kind == KindInteger
}

func (v Value) Text() (string, bool) {
	return v.text, v.kind == KindString
}

func (v Value) Elements() ([]Value, bool) {
	if v.kind != KindArray {
		return nil, false
	}
	return append([]Value(nil), v.array...), true
}

func (v Value) Members() ([]Member, bool) {
	if v.kind != KindObject {
		return nil, false
	}
	return append([]Member(nil), v.object...), true
}

// LookupMember returns one object member by its exact decoded name. Objects
// retain UTF-8 byte ordering, so lookup does not allocate or expose composite
// storage. A non-object and an absent member both return the zero value,false;
// callers that need to distinguish them can inspect Kind first.
func (v Value) LookupMember(name string) (Value, bool) {
	if v.kind != KindObject {
		return Value{}, false
	}
	index := sort.Search(len(v.object), func(index int) bool {
		return v.object[index].Name >= name
	})
	if index >= len(v.object) || v.object[index].Name != name {
		return Value{}, false
	}
	return v.object[index].Value, true
}
