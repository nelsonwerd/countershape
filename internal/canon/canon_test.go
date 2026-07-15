package canon

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"unicode/utf8"
)

type typedJSONBypass string

func (typedJSONBypass) MarshalJSON() ([]byte, error) {
	return []byte(`"bypassed"`), nil
}

func duplicateTaggedStruct() any {
	typeOfString := reflect.TypeOf("")
	typeOfValue := reflect.StructOf([]reflect.StructField{
		{Name: "First", Type: typeOfString, Tag: `json:"value"`},
		{Name: "Second", Type: typeOfString, Tag: `json:"value"`},
	})
	value := reflect.New(typeOfValue).Elem()
	value.Field(0).SetString("one")
	value.Field(1).SetString("two")
	return value.Interface()
}

func TestCanonicalGoldenBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "whitespace and object order",
			input: " \n { \"b\" : 2, \"a\" : 1 } \t",
			want:  `{"a":1,"b":2}`,
		},
		{
			name:  "nested order and array order",
			input: `{"z":{"β":2,"a":1},"a":[3,2,1]}`,
			want:  `{"a":[3,2,1],"z":{"a":1,"β":2}}`,
		},
		{
			name:  "escaped names and canonical escapes",
			input: `{"\u0062":1,"a":2,"s":"\u0061\/\b\f\n\r\t\u0001"}`,
			want:  `{"a":2,"b":1,"s":"a/\b\f\n\r\t\u0001"}`,
		},
		{
			name:  "surrogate pair becomes exact UTF-8",
			input: `{"emoji":"\uD83D\uDE00"}`,
			want:  `{"emoji":"😀"}`,
		},
		{
			name:  "numbers remain integral",
			input: `[-9007199254740991,0,9007199254740991]`,
			want:  `[-9007199254740991,0,9007199254740991]`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Canonicalize([]byte(test.input))
			if err != nil {
				t.Fatalf("Canonicalize() error = %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("Canonicalize() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestScanIsLosslessOwnedAndOffsetBearing(t *testing.T) {
	t.Parallel()
	input := []byte(" { \"b\" : [true, null] }\n")
	want := append([]byte(nil), input...)
	tokens, err := Scan(input)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	input[0] = 'X'

	var rebuilt []byte
	for _, token := range tokens {
		if token.Kind != TokenEOF {
			rebuilt = append(rebuilt, token.Raw...)
		}
	}
	if !bytes.Equal(rebuilt, want) {
		t.Fatalf("joined token bytes = %q, want exact %q", rebuilt, want)
	}
	if len(tokens) == 0 || tokens[0].Kind != TokenWhitespace || tokens[0].Offset != 0 {
		t.Fatalf("first token = %#v, want whitespace at byte 0", tokens[0])
	}
	eof := tokens[len(tokens)-1]
	if eof.Kind != TokenEOF || eof.Offset != len(want) || len(eof.Raw) != 0 {
		t.Fatalf("EOF token = %#v, want offset %d with no raw bytes", eof, len(want))
	}
}

func TestScanRejectsLoneSurrogates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		offset int
	}{
		{name: "high without low", input: `"\uD800"`, offset: 1},
		{name: "low without high", input: `"\uDC00"`, offset: 1},
		{name: "high followed by scalar", input: `"\uD800\u0041"`, offset: 7},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Scan([]byte(test.input))
			requireCanonError(t, err, CodeLoneSurrogate, test.offset)
		})
	}
	if _, err := Scan([]byte(`"\uD83D\uDE00"`)); err != nil {
		t.Fatalf("Scan(valid surrogate pair) error = %v", err)
	}
}

func TestParseRejectsDuplicateKeys(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(`{"a":1,"\u0061":2}`))
	requireCanonError(t, err, CodeDuplicateKey, 7)

	value, err := Integer(1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Object(Member{Name: "a", Value: value}, Member{Name: "a", Value: value})
	requireCanonError(t, err, CodeDuplicateKey, UnknownOffset)
}

func TestParseRejectsNegativeZero(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(`{"n":-0}`))
	requireCanonError(t, err, CodeNegativeZero, 5)
}

func TestNumericProfileBoundariesAndRefusals(t *testing.T) {
	t.Parallel()
	accepted := []struct {
		input string
		want  int64
	}{
		{input: "0", want: 0},
		{input: "1", want: 1},
		{input: "-1", want: -1},
		{input: "9007199254740991", want: MaxSafeInteger},
		{input: "-9007199254740991", want: MinSafeInteger},
	}
	for _, test := range accepted {
		value, err := Parse([]byte(test.input))
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", test.input, err)
		}
		got, ok := value.Int64()
		if !ok || got != test.want {
			t.Fatalf("Parse(%q) integer = (%d,%t), want (%d,true)", test.input, got, ok, test.want)
		}
	}

	rejected := []struct {
		input  string
		code   ErrorCode
		offset int
	}{
		{input: "9007199254740992", code: CodeUnsafeInteger, offset: 0},
		{input: "-9007199254740992", code: CodeUnsafeInteger, offset: 0},
		{input: "-0", code: CodeNegativeZero, offset: 0},
		{input: "0.0", code: CodeUnsupportedNumber, offset: 1},
		{input: "1e0", code: CodeUnsupportedNumber, offset: 1},
		{input: "1E+2", code: CodeUnsupportedNumber, offset: 1},
		{input: "01", code: CodeUnsupportedNumber, offset: 1},
		{input: "-01", code: CodeUnsupportedNumber, offset: 2},
		{input: "+1", code: CodeUnsupportedNumber, offset: 0},
		{input: "NaN", code: CodeUnsupportedNumber, offset: 0},
		{input: "Infinity", code: CodeUnsupportedNumber, offset: 0},
		{input: "-Infinity", code: CodeUnsupportedNumber, offset: 1},
		{input: "1x", code: CodeUnsupportedNumber, offset: 1},
	}
	for _, test := range rejected {
		_, err := Parse([]byte(test.input))
		requireCanonError(t, err, test.code, test.offset)
	}

	if _, err := Integer(MaxSafeInteger + 1); err == nil {
		t.Fatal("Integer() accepted a value above the safe range")
	}
	if _, err := Integer(MinSafeInteger - 1); err == nil {
		t.Fatal("Integer() accepted a value below the safe range")
	}
}

func TestIntegerFromStringUsesTheSameProfile(t *testing.T) {
	t.Parallel()
	value, err := IntegerFromString("-9007199254740991")
	if err != nil {
		t.Fatalf("IntegerFromString() error = %v", err)
	}
	got, ok := value.Int64()
	if !ok || got != MinSafeInteger {
		t.Fatalf("IntegerFromString() = (%d,%t), want (%d,true)", got, ok, MinSafeInteger)
	}
	for _, input := range []string{"-0", "01", "1.0", "9007199254740992"} {
		if _, err := IntegerFromString(input); err == nil {
			t.Fatalf("IntegerFromString(%q) unexpectedly succeeded", input)
		}
	}
}

func TestErrorCodesAndByteOffsets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  []byte
		code   ErrorCode
		offset int
	}{
		{name: "invalid UTF-8", input: []byte{'"', 0xff, '"'}, code: CodeInvalidUTF8, offset: 1},
		{name: "unescaped control", input: []byte{'"', 0x01, '"'}, code: CodeControlCharacter, offset: 1},
		{name: "invalid escape", input: []byte(`"\q"`), code: CodeInvalidEscape, offset: 2},
		{name: "truncated string", input: []byte(`"abc`), code: CodeUnexpectedEOF, offset: 4},
		{name: "trailing value", input: []byte(`null true`), code: CodeTrailingData, offset: 5},
		{name: "invalid literal", input: []byte(`nullx`), code: CodeInvalidLiteral, offset: 0},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(test.input)
			requireCanonError(t, err, test.code, test.offset)
		})
	}
}

func TestErrorCodeVocabularyIsStable(t *testing.T) {
	t.Parallel()
	want := map[ErrorCode]string{
		CodeInvalidUTF8:       "CANON_INVALID_UTF8",
		CodeUnexpectedByte:    "CANON_UNEXPECTED_BYTE",
		CodeUnexpectedEOF:     "CANON_UNEXPECTED_EOF",
		CodeInvalidLiteral:    "CANON_INVALID_LITERAL",
		CodeInvalidEscape:     "CANON_INVALID_ESCAPE",
		CodeLoneSurrogate:     "CANON_LONE_SURROGATE",
		CodeControlCharacter:  "CANON_CONTROL_CHARACTER",
		CodeUnsupportedNumber: "CANON_UNSUPPORTED_NUMBER",
		CodeNegativeZero:      "CANON_NEGATIVE_ZERO",
		CodeUnsafeInteger:     "CANON_UNSAFE_INTEGER",
		CodeDuplicateKey:      "CANON_DUPLICATE_KEY",
		CodeUnexpectedToken:   "CANON_UNEXPECTED_TOKEN",
		CodeTrailingData:      "CANON_TRAILING_DATA",
		CodeDepthExceeded:     "CANON_DEPTH_EXCEEDED",
		CodeInvalidKind:       "CANON_INVALID_KIND",
		CodeInvalidValue:      "CANON_INVALID_VALUE",
	}
	for code, text := range want {
		if string(code) != text {
			t.Fatalf("error code = %q, want stable %q", code, text)
		}
	}
}

func TestUTF8KeyOrderWithoutUnicodeNormalization(t *testing.T) {
	t.Parallel()
	input := []byte(`{"é":1,"😀":2,"z":3,"e\u0301":4}`)
	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}
	want := `{"é":4,"z":3,"é":1,"😀":2}`
	if string(got) != want {
		t.Fatalf("canonical UTF-8 order = %q, want %q", got, want)
	}

	composed, err := String("é")
	if err != nil {
		t.Fatal(err)
	}
	decomposed, err := String("e\u0301")
	if err != nil {
		t.Fatal(err)
	}
	if composed.Equal(decomposed) {
		t.Fatal("composed and decomposed strings were Unicode-normalized")
	}
	left, err := DigestValue("ProjectionResult", composed)
	if err != nil {
		t.Fatal(err)
	}
	right, err := DigestValue("ProjectionResult", decomposed)
	if err != nil {
		t.Fatal(err)
	}
	if left.Equal(right) {
		t.Fatal("distinct non-normalized values received the same digest")
	}
}

func TestMissingAndPresentEmptyRemainDistinct(t *testing.T) {
	t.Parallel()
	emptyText, err := String("")
	if err != nil {
		t.Fatal(err)
	}
	missing, err := Object()
	if err != nil {
		t.Fatal(err)
	}
	presentEmpty, err := Object(Member{Name: "field", Value: emptyText})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(missing.Canonical(), presentEmpty.Canonical()) {
		t.Fatal("missing key and present-empty string have identical canonical bytes")
	}
	missingDigest, err := DigestValue("ExactTuple", missing)
	if err != nil {
		t.Fatal(err)
	}
	presentDigest, err := DigestValue("ExactTuple", presentEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if missingDigest.Equal(presentDigest) {
		t.Fatal("missing key and present-empty string have identical digests")
	}
}

func TestConstructorsAreImmutableAndOrdered(t *testing.T) {
	t.Parallel()
	one, err := Integer(1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Integer(2)
	if err != nil {
		t.Fatal(err)
	}
	values := []Value{one, two}
	array, err := Array(values...)
	if err != nil {
		t.Fatal(err)
	}
	values[0] = Null()
	if string(array.Canonical()) != "[1,2]" {
		t.Fatalf("array changed through constructor input: %s", array.Canonical())
	}

	object, err := Object(Member{Name: "z", Value: two}, Member{Name: "a", Value: one})
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Canonical()) != `{"a":1,"z":2}` {
		t.Fatalf("Object canonical bytes = %s", object.Canonical())
	}
	members, ok := object.Members()
	if !ok {
		t.Fatal("Members() rejected an object")
	}
	members[0].Name = "mutated"
	if string(object.Canonical()) != `{"a":1,"z":2}` {
		t.Fatalf("object changed through Members result: %s", object.Canonical())
	}
}

func TestLookupMemberFindsExactOrderedNamesWithoutExposingStorage(t *testing.T) {
	t.Parallel()
	one, err := Integer(1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Integer(2)
	if err != nil {
		t.Fatal(err)
	}
	nested, err := Object(Member{Name: "value", Value: two})
	if err != nil {
		t.Fatal(err)
	}
	object, err := Object(
		Member{Name: "z", Value: two},
		Member{Name: "a", Value: one},
		Member{Name: "middle", Value: nested},
	)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]Value{"a": one, "middle": nested, "z": two} {
		got, ok := object.LookupMember(name)
		if !ok || !got.Equal(want) {
			t.Fatalf("LookupMember(%q) = (%s,%t), want (%s,true)", name, got.Canonical(), ok, want.Canonical())
		}
	}
	if _, ok := object.LookupMember("absent"); ok {
		t.Fatal("LookupMember reported an absent object member")
	}
	if _, ok := one.LookupMember("a"); ok {
		t.Fatal("LookupMember treated a non-object as an object")
	}

	lookedUp, _ := object.LookupMember("middle")
	members, ok := lookedUp.Members()
	if !ok {
		t.Fatal("looked-up nested object lost its kind")
	}
	members[0].Name = "mutated"
	if string(object.Canonical()) != `{"a":1,"middle":{"value":2},"z":2}` {
		t.Fatalf("looked-up value exposed mutable object storage: %s", object.Canonical())
	}
}

func TestDigestKindSeparation(t *testing.T) {
	t.Parallel()
	left, err := DigestBytes("A", []byte("B"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := DigestBytes("AB", nil)
	if err != nil {
		t.Fatal(err)
	}
	if left.Equal(right) {
		t.Fatal("kind/bytes boundary is ambiguous; the NUL separator is absent")
	}

	const fixedGolden = "sha256:4772e7d18b46280a24135d39a4bc8024a7d0ce5b0e4ed01c00b11d243aeca064"
	if left.String() != fixedGolden {
		t.Fatalf("DigestBytes fixed golden mismatch: got %s, want %s", left.String(), fixedGolden)
	}
	again, err := DigestBytes("A", []byte("B"))
	if err != nil {
		t.Fatal(err)
	}
	if !left.Equal(again) {
		t.Fatal("same kind and bytes produced different digests")
	}
	if !strings.HasPrefix(left.String(), "sha256:") || len(left.String()) != 71 {
		t.Fatalf("Digest.String() = %q", left.String())
	}
}

func TestTypedCanonicalMarshalFixedGolden(t *testing.T) {
	t.Parallel()
	input := struct {
		SchemaVersion  string   `json:"schema_version"`
		Kind           string   `json:"kind"`
		SelectedFields []string `json:"selected_fields"`
		Compilable     bool     `json:"compilable"`
	}{
		SchemaVersion:  "countershape/v1",
		Kind:           "DecisionRecord",
		SelectedFields: []string{"http.status", "http.body.kind"},
		Compilable:     true,
	}

	const fixedCanonical = `{"compilable":true,"kind":"DecisionRecord","schema_version":"countershape/v1","selected_fields":["http.status","http.body.kind"]}`
	value, err := MarshalTyped(input)
	if err != nil {
		t.Fatalf("MarshalTyped() error = %v", err)
	}
	if got := string(value.Canonical()); got != fixedCanonical {
		t.Fatalf("MarshalTyped() = %s, want fixed %s", got, fixedCanonical)
	}
	digest, canonical, err := DigestTyped("DecisionRecord", input)
	if err != nil {
		t.Fatalf("DigestTyped() error = %v", err)
	}
	if string(canonical) != fixedCanonical {
		t.Fatalf("DigestTyped canonical = %s, want fixed %s", canonical, fixedCanonical)
	}
	wantDigest, err := DigestBytes("DecisionRecord", []byte(fixedCanonical))
	if err != nil {
		t.Fatal(err)
	}
	if !digest.Equal(wantDigest) {
		t.Fatalf("DigestTyped() = %s, want %s", digest.String(), wantDigest.String())
	}
}

func TestTypedCanonicalMarshalRejectsInvalidUTF8BeforeJSONEncoding(t *testing.T) {
	t.Parallel()
	invalid := string([]byte{0xff})
	tests := []any{
		struct {
			Value string `json:"value"`
		}{Value: invalid},
		map[string]string{"valid": invalid},
		map[string]string{invalid: "valid"},
		[]string{"valid", invalid},
	}
	for _, input := range tests {
		_, err := MarshalTyped(input)
		requireCanonError(t, err, CodeInvalidUTF8, UnknownOffset)
	}
}

func TestTypedCanonicalMarshalRejectsUnsafeOrCoercedNumbers(t *testing.T) {
	t.Parallel()
	unsafeSigned := struct {
		Number int64 `json:"number"`
	}{Number: MaxSafeInteger + 1}
	_, err := MarshalTyped(unsafeSigned)
	requireCanonError(t, err, CodeUnsafeInteger, UnknownOffset)

	unsafeUnsigned := struct {
		Number uint64 `json:"number"`
	}{Number: uint64(MaxSafeInteger) + 1}
	_, err = MarshalTyped(unsafeUnsigned)
	requireCanonError(t, err, CodeUnsafeInteger, UnknownOffset)

	for _, input := range []any{float64(1), float64(-0.0), json.Number("1"), json.Number("-0")} {
		_, err = MarshalTyped(input)
		requireCanonError(t, err, CodeUnsupportedNumber, UnknownOffset)
	}
}

func TestTypedCanonicalMarshalRejectsImplicitEncodingSemantics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input any
		code  ErrorCode
	}{
		{name: "custom marshaler", input: typedJSONBypass("value"), code: CodeInvalidValue},
		{name: "byte slice base64", input: []byte("value"), code: CodeInvalidValue},
		{name: "nil byte slice", input: []byte(nil), code: CodeInvalidValue},
		{name: "byte array implicit numbers", input: [2]byte{1, 2}, code: CodeInvalidValue},
		{name: "non-string map key", input: map[int]string{1: "value"}, code: CodeInvalidValue},
		{name: "nil non-string-keyed map", input: map[int]string(nil), code: CodeInvalidValue},
		{name: "tag option", input: struct {
			Value string `json:"value,omitempty"`
		}{Value: "value"}, code: CodeInvalidValue},
		{name: "tag silently reinterpreted by encoding/json", input: struct {
			Value string `json:"bad\\name"`
		}{Value: "value"}, code: CodeInvalidValue},
		{name: "omitted field", input: struct {
			Value string `json:"-"`
		}{Value: "value"}, code: CodeInvalidValue},
		{name: "duplicate tag", input: duplicateTaggedStruct(), code: CodeDuplicateKey},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := MarshalTyped(test.input)
			requireCanonError(t, err, test.code, UnknownOffset)
		})
	}
}

func TestTypedCanonicalMarshalDoesNotDefaultNilContainers(t *testing.T) {
	t.Parallel()
	type container struct {
		Values []string `json:"values"`
	}
	nilCanonical, err := CanonicalizeTyped(container{})
	if err != nil {
		t.Fatal(err)
	}
	emptyCanonical, err := CanonicalizeTyped(container{Values: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if string(nilCanonical) != `{"values":null}` {
		t.Fatalf("nil slice = %s, want explicit null", nilCanonical)
	}
	if string(emptyCanonical) != `{"values":[]}` {
		t.Fatalf("empty slice = %s, want explicit array", emptyCanonical)
	}
	if bytes.Equal(nilCanonical, emptyCanonical) {
		t.Fatal("typed marshal defaulted nil and empty containers to one identity")
	}
}

func TestTypedCanonicalMarshalBoundsCyclesBeforeJSONEncoding(t *testing.T) {
	t.Parallel()
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	_, err := MarshalTyped(cyclic)
	requireCanonError(t, err, CodeDepthExceeded, UnknownOffset)
}

func TestDigestKindValidation(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"", "bad\x00kind", "bad\nkind", string([]byte{0xff})} {
		if _, err := DigestBytes(kind, nil); err == nil {
			t.Fatalf("DigestBytes(%q) accepted invalid kind", kind)
		}
	}
	if _, err := DigestBytes("Projection�Result", nil); err != nil {
		t.Fatalf("valid replacement rune was rejected: %v", err)
	}
}

func TestCanonicalizationIsIdempotent(t *testing.T) {
	t.Parallel()
	inputs := [][]byte{
		[]byte(`null`),
		[]byte(` [ true, false, null, {"b":2,"a":1} ] `),
		[]byte(`{"emoji":"\uD83D\uDE00","escaped":"\u0061\/b"}`),
		[]byte(`{"e\u0301":1,"é":2}`),
	}
	for _, input := range inputs {
		first, err := Canonicalize(input)
		if err != nil {
			t.Fatalf("Canonicalize(%q) error = %v", input, err)
		}
		second, err := Canonicalize(first)
		if err != nil {
			t.Fatalf("Canonicalize(canonical %q) error = %v", first, err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("canonicalization is not idempotent: %q != %q", first, second)
		}
	}
}

func TestArrayOrderIsIdentityBearing(t *testing.T) {
	t.Parallel()
	left, err := Parse([]byte(`[1,2,3]`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := Parse([]byte(`[3,2,1]`))
	if err != nil {
		t.Fatal(err)
	}
	if left.Equal(right) {
		t.Fatal("array order was treated as unordered")
	}
	leftDigest, err := DigestValue("OrderedValues", left)
	if err != nil {
		t.Fatal(err)
	}
	rightDigest, err := DigestValue("OrderedValues", right)
	if err != nil {
		t.Fatal(err)
	}
	if leftDigest.Equal(rightDigest) {
		t.Fatal("array permutation preserved identity")
	}
}

func TestStructuralRefusalsAndDepthBound(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "[1,]", `{"a" 1}`, `{"a":1,}`, "[1 2]", "{}{}"} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", input)
		}
	}
	tooDeep := strings.Repeat("[", maxNestingDepth+2) + strings.Repeat("]", maxNestingDepth+2)
	_, err := Parse([]byte(tooDeep))
	var canonErr *Error
	if !errors.As(err, &canonErr) || canonErr.Code != CodeDepthExceeded {
		t.Fatalf("deep Parse() error = %v, want %s", err, CodeDepthExceeded)
	}
}

func TestPropertyParseEncodeRoundTrip(t *testing.T) {
	property := func(number int64, text string) bool {
		number %= MaxSafeInteger
		if !utf8.ValidString(text) {
			return true
		}
		integer, err := Integer(number)
		if err != nil {
			return false
		}
		stringValue, err := String(text)
		if err != nil {
			return false
		}
		value, err := Object(
			Member{Name: "integer", Value: integer},
			Member{Name: "string", Value: stringValue},
		)
		if err != nil {
			return false
		}
		parsed, err := Parse(value.Canonical())
		return err == nil && value.Equal(parsed)
	}
	const seed int64 = 0xC01A5E
	config := &quick.Config{MaxCount: 500, Rand: rand.New(rand.NewSource(seed))}
	if err := quick.Check(property, config); err != nil {
		t.Fatalf("seed=%d: %v", seed, err)
	}
}

func TestPropertyObjectPermutationInvariance(t *testing.T) {
	t.Parallel()
	base := make([]Member, 8)
	for index := range base {
		value, err := Integer(int64(index))
		if err != nil {
			t.Fatal(err)
		}
		base[index] = Member{Name: string(rune('a' + index)), Value: value}
	}
	want, err := Object(base...)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes := want.Canonical()
	for seed := int64(0); seed < 250; seed++ {
		shuffled := append([]Member(nil), base...)
		rand.New(rand.NewSource(seed)).Shuffle(len(shuffled), func(left, right int) {
			shuffled[left], shuffled[right] = shuffled[right], shuffled[left]
		})
		got, err := Object(shuffled...)
		if err != nil {
			t.Fatalf("seed %d: Object() error = %v", seed, err)
		}
		if !bytes.Equal(got.Canonical(), wantBytes) {
			t.Fatalf("seed %d changed object identity: %s", seed, got.Canonical())
		}
	}
}

func requireCanonError(t *testing.T, err error, code ErrorCode, offset int) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s at byte %d", code, offset)
	}
	var canonErr *Error
	if !errors.As(err, &canonErr) {
		t.Fatalf("error type = %T, want *canon.Error: %v", err, err)
	}
	if canonErr.Code != code || canonErr.Offset != offset {
		t.Fatalf("error = (%s,%d), want (%s,%d): %v", canonErr.Code, canonErr.Offset, code, offset, err)
	}
}
