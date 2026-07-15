package canon

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestResourceLimitCodesAreStable(t *testing.T) {
	want := map[ErrorCode]string{
		CodeInputLimit:     "CANON_INPUT_LIMIT",
		CodeTokenLimit:     "CANON_TOKEN_LIMIT",
		CodeMemberLimit:    "CANON_MEMBER_LIMIT",
		CodeTypedNodeLimit: "CANON_TYPED_NODE_LIMIT",
		CodeTypedByteLimit: "CANON_TYPED_BYTE_LIMIT",
	}
	for code, text := range want {
		if string(code) != text {
			t.Fatalf("resource error code = %q, want stable %q", code, text)
		}
	}
}

func TestInputAndTokenCeilingsFailClosed(t *testing.T) {
	oversized := make([]byte, MaxInputBytes+1)
	_, err := Scan(oversized)
	requireExactCode(t, err, CodeInputLimit)

	// Alternating scalar and whitespace tokens stays below the byte ceiling
	// while crossing the independently enforced complete-token ceiling.
	tokenHeavy := []byte(strings.Repeat("0 ", MaxTokenCount))
	_, err = Scan(tokenHeavy)
	requireExactCode(t, err, CodeTokenLimit)
}

func TestArrayAndObjectMemberCeilingsFailClosed(t *testing.T) {
	var array strings.Builder
	array.WriteByte('[')
	for index := 0; index <= MaxContainerMembers; index++ {
		if index > 0 {
			array.WriteByte(',')
		}
		array.WriteByte('0')
	}
	array.WriteByte(']')
	_, err := Parse([]byte(array.String()))
	requireExactCode(t, err, CodeMemberLimit)

	var object strings.Builder
	object.WriteByte('{')
	for index := 0; index <= MaxContainerMembers; index++ {
		if index > 0 {
			object.WriteByte(',')
		}
		object.WriteString(strconv.Quote(strconv.Itoa(index)))
		object.WriteString(":0")
	}
	object.WriteByte('}')
	_, err = Parse([]byte(object.String()))
	requireExactCode(t, err, CodeMemberLimit)
}

func TestTypedCeilingsFailClosedBeforeJSONEncoding(t *testing.T) {
	tooWide := make([]int, MaxContainerMembers+1)
	_, err := MarshalTyped(tooWide)
	requireExactCode(t, err, CodeMemberLimit)

	tooManyNodes := make([][]int, MaxTypedNodes/MaxContainerMembers+1)
	for index := range tooManyNodes {
		tooManyNodes[index] = make([]int, MaxContainerMembers)
	}
	_, err = MarshalTyped(tooManyNodes)
	requireExactCode(t, err, CodeTypedNodeLimit)

	_, err = MarshalTyped(strings.Repeat("x", MaxTypedStringBytes+1))
	requireExactCode(t, err, CodeTypedByteLimit)
}

func TestProgrammaticConstructorsEnforceTheCompleteResourceProfile(t *testing.T) {
	_, err := Array(make([]Value, MaxContainerMembers+1)...)
	requireExactCode(t, err, CodeMemberLimit)

	_, err = String(strings.Repeat("x", MaxTypedStringBytes+1))
	requireExactCode(t, err, CodeTypedByteLimit)

	left, err := String(strings.Repeat("a", MaxTypedStringBytes/2+1))
	if err != nil {
		t.Fatal(err)
	}
	right, err := String(strings.Repeat("b", MaxTypedStringBytes/2+1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Array(left, right)
	requireExactCode(t, err, CodeTypedByteLimit)
	_, err = Object(Member{Name: "left", Value: left}, Member{Name: "right", Value: right})
	requireExactCode(t, err, CodeTypedByteLimit)

	value := Null()
	for depth := 0; depth <= maxNestingDepth; depth++ {
		value, err = Array(value)
		if depth < maxNestingDepth && err != nil {
			t.Fatalf("depth %d refused too early: %v", depth, err)
		}
	}
	requireExactCode(t, err, CodeDepthExceeded)
}

func TestTypedMapRefusalsAreKeyOrdered(t *testing.T) {
	input := map[string]any{
		"z": uint64(MaxSafeInteger) + 1,
		"a": float64(1),
	}
	for attempt := 0; attempt < 100; attempt++ {
		_, err := MarshalTyped(input)
		requireExactCode(t, err, CodeUnsupportedNumber)
	}
}

func TestImpossibleValueKindCannotAliasCanonicalNull(t *testing.T) {
	impossible := Value{kind: ValueKind(255)}
	_, err := impossible.CanonicalChecked()
	requireExactCode(t, err, CodeInvalidKind)

	deferred := func() (recovered any) {
		defer func() { recovered = recover() }()
		_ = impossible.Canonical()
		return nil
	}()
	canonErr, ok := deferred.(*Error)
	if !ok || canonErr.Code != CodeInvalidKind {
		t.Fatalf("Canonical impossible-kind panic = %#v, want *canon.Error with %s", deferred, CodeInvalidKind)
	}
}

func requireExactCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	var canonErr *Error
	if !errors.As(err, &canonErr) {
		t.Fatalf("error = %v, want *canon.Error with %s", err, want)
	}
	if canonErr.Code != want {
		t.Fatalf("error code = %s, want %s", canonErr.Code, want)
	}
}
