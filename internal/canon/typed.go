package canon

import (
	"encoding"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	jsonNumberType    = reflect.TypeOf(json.Number(""))
	valueType         = reflect.TypeOf(Value{})
)

// MarshalTyped converts a project-owned typed value into the strict canonical
// model. It validates every reachable string and numeric value before
// encoding/json can repair invalid UTF-8 or choose an unsupported number
// spelling. Custom marshalers, implicit byte containers, embedded fields, tag
// options, and non-string map keys are refused rather than interpreted.
//
// This helper is for inert Go structs, slices, string-keyed maps, pointers, and
// interfaces assembled by Countershape. Untrusted JSON bytes must still enter
// through Parse so duplicate names and source offsets remain observable.
// Nil slices/maps remain JSON null and are never defaulted to empty containers;
// domain constructors that require arrays or objects must normalize them before
// calling this helper.
func MarshalTyped(input any) (Value, error) {
	if value, ok := input.(Value); ok {
		if _, err := value.CanonicalChecked(); err != nil {
			return Value{}, err
		}
		return value, nil
	}
	if value, ok := input.(*Value); ok {
		if value == nil {
			return Null(), nil
		}
		if _, err := value.CanonicalChecked(); err != nil {
			return Value{}, err
		}
		return *value, nil
	}
	budget := traversalBudget{}
	if err := validateTypedValue(reflect.ValueOf(input), 0, &budget); err != nil {
		return Value{}, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return Value{}, refusal(CodeInvalidValue, UnknownOffset, "typed value cannot be encoded as strict JSON")
	}
	return Parse(raw)
}

// CanonicalizeTyped returns deterministic bytes for a validated typed value.
func CanonicalizeTyped(input any) ([]byte, error) {
	value, err := MarshalTyped(input)
	if err != nil {
		return nil, err
	}
	return value.CanonicalChecked()
}

// DigestTyped validates a typed value, returns its canonical bytes, and hashes
// those exact bytes in the named Countershape domain.
func DigestTyped(kind string, input any) (Digest, []byte, error) {
	canonical, err := CanonicalizeTyped(input)
	if err != nil {
		return Digest{}, nil, err
	}
	digest, err := DigestBytes(kind, canonical)
	if err != nil {
		return Digest{}, nil, err
	}
	return digest, canonical, nil
}

func validateTypedValue(value reflect.Value, depth int, budget *traversalBudget) error {
	if depth > maxNestingDepth {
		return refusal(CodeDepthExceeded, UnknownOffset, "typed value exceeds the maximum nesting depth")
	}
	if err := budget.enter(); err != nil {
		return err
	}
	if !value.IsValid() {
		return nil
	}

	typeOfValue := value.Type()
	if typeOfValue == valueType {
		return refusal(CodeInvalidValue, UnknownOffset, "nested canon.Value requires explicit object or array construction")
	}
	if typeOfValue == jsonNumberType {
		return refusal(CodeUnsupportedNumber, UnknownOffset, "json.Number bypasses the closed integer profile")
	}
	if implementsCustomMarshaler(typeOfValue) {
		return refusal(CodeInvalidValue, UnknownOffset, "custom JSON or text marshalers are unsupported in typed identity")
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return validateTypedValue(value.Elem(), depth+1, budget)
	case reflect.Bool:
		return nil
	case reflect.String:
		if !utf8.ValidString(value.String()) {
			return refusal(CodeInvalidUTF8, UnknownOffset, "typed string is not valid UTF-8")
		}
		return budget.addString(value.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer := value.Int()
		if integer < MinSafeInteger || integer > MaxSafeInteger {
			return refusal(CodeUnsafeInteger, UnknownOffset, "typed integer exceeds the exact interoperability range")
		}
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() > uint64(MaxSafeInteger) {
			return refusal(CodeUnsafeInteger, UnknownOffset, "typed unsigned integer exceeds the exact interoperability range")
		}
		return nil
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Uintptr:
		return refusal(CodeUnsupportedNumber, UnknownOffset, "typed identity supports safe-range integers only")
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return refusal(CodeInvalidValue, UnknownOffset, "byte slices require an explicit typed representation")
		}
		if value.IsNil() {
			return nil
		}
		if value.Len() > MaxContainerMembers {
			return refusal(CodeMemberLimit, UnknownOffset, "typed slice exceeds the v1 member ceiling")
		}
		for index := 0; index < value.Len(); index++ {
			if err := validateTypedValue(value.Index(index), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	case reflect.Array:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return refusal(CodeInvalidValue, UnknownOffset, "byte arrays require an explicit typed representation")
		}
		if value.Len() > MaxContainerMembers {
			return refusal(CodeMemberLimit, UnknownOffset, "typed array exceeds the v1 member ceiling")
		}
		for index := 0; index < value.Len(); index++ {
			if err := validateTypedValue(value.Index(index), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return refusal(CodeInvalidValue, UnknownOffset, "typed identity maps require string keys")
		}
		if implementsCustomMarshaler(value.Type().Key()) {
			return refusal(CodeInvalidValue, UnknownOffset, "typed identity map keys cannot use custom marshalers")
		}
		if value.IsNil() {
			return nil
		}
		if value.Len() > MaxContainerMembers {
			return refusal(CodeMemberLimit, UnknownOffset, "typed map exceeds the v1 member ceiling")
		}
		keys := value.MapKeys()
		sort.Slice(keys, func(left, right int) bool {
			return keys[left].String() < keys[right].String()
		})
		for _, key := range keys {
			if !utf8.ValidString(key.String()) {
				return refusal(CodeInvalidUTF8, UnknownOffset, "typed map key is not valid UTF-8")
			}
			if err := budget.addString(key.String()); err != nil {
				return err
			}
			if err := validateTypedValue(value.MapIndex(key), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		return validateTypedStruct(value, depth, budget)
	default:
		return refusal(CodeInvalidValue, UnknownOffset, "typed value contains an unsupported Go kind")
	}
}

func validateTypedStruct(value reflect.Value, depth int, budget *traversalBudget) error {
	typeOfValue := value.Type()
	if typeOfValue.NumField() > MaxContainerMembers {
		return refusal(CodeMemberLimit, UnknownOffset, "typed struct exceeds the v1 member ceiling")
	}
	seenNames := make(map[string]struct{}, typeOfValue.NumField())
	for index := 0; index < typeOfValue.NumField(); index++ {
		field := typeOfValue.Field(index)
		if field.PkgPath != "" {
			return refusal(CodeInvalidValue, UnknownOffset, "unexported struct fields are unsupported in typed identity")
		}
		if field.Anonymous {
			return refusal(CodeInvalidValue, UnknownOffset, "anonymous fields are unsupported in typed identity")
		}
		name, include, err := typedJSONFieldName(field)
		if err != nil {
			return err
		}
		if !include {
			continue
		}
		if _, duplicate := seenNames[name]; duplicate {
			return refusal(CodeDuplicateKey, UnknownOffset, "typed struct contains duplicate JSON field names")
		}
		seenNames[name] = struct{}{}
		if err := budget.addString(name); err != nil {
			return err
		}
		if err := validateTypedValue(value.Field(index), depth+1, budget); err != nil {
			return err
		}
	}
	return nil
}

func typedJSONFieldName(field reflect.StructField) (string, bool, error) {
	name := field.Name
	if tag, exists := field.Tag.Lookup("json"); exists {
		parts := strings.Split(tag, ",")
		if parts[0] == "-" {
			return "", false, refusal(CodeInvalidValue, UnknownOffset, "JSON field omission is unsupported in typed identity")
		}
		if len(parts) > 1 {
			return "", false, refusal(CodeInvalidValue, UnknownOffset, "JSON tag options are unsupported in typed identity")
		}
		if parts[0] != "" {
			if !utf8.ValidString(parts[0]) {
				return "", false, refusal(CodeInvalidUTF8, UnknownOffset, "typed JSON field name is not valid UTF-8")
			}
			if !validTypedJSONFieldName(parts[0]) {
				return "", false, refusal(CodeInvalidValue, UnknownOffset, "JSON field name would be reinterpreted by encoding/json")
			}
			name = parts[0]
		}
	}
	if !utf8.ValidString(name) {
		return "", false, refusal(CodeInvalidUTF8, UnknownOffset, "typed JSON field name is not valid UTF-8")
	}
	return name, true, nil
}

// Keep this admission rule aligned with encoding/json's exported-field tag
// grammar. Refusing a tag that the encoder would silently ignore is critical:
// validation must reason about the same member name that reaches Parse.
func validTypedJSONFieldName(name string) bool {
	if name == "" {
		return false
	}
	for _, character := range name {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", character):
		case unicode.IsLetter(character), unicode.IsDigit(character):
		default:
			return false
		}
	}
	return true
}

func implementsCustomMarshaler(typeOfValue reflect.Type) bool {
	if typeOfValue.Implements(jsonMarshalerType) || typeOfValue.Implements(textMarshalerType) {
		return true
	}
	if typeOfValue.Kind() != reflect.Pointer {
		pointer := reflect.PointerTo(typeOfValue)
		return pointer.Implements(jsonMarshalerType) || pointer.Implements(textMarshalerType)
	}
	return false
}
