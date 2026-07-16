package portablevalue

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
)

func TestPortableValueDistinctExactIdentities(t *testing.T) {
	emptyString, err := String("")
	if err != nil {
		t.Fatal(err)
	}
	emptyBytes, err := Bytes([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	emptyList, err := OrderedStringList([]string{})
	if err != nil {
		t.Fatal(err)
	}
	oneEmpty, err := OrderedStringList([]string{""})
	if err != nil {
		t.Fatal(err)
	}
	values := []Value{Missing(), Null(), emptyString, emptyBytes, emptyList, oneEmpty}
	for left := range values {
		if !values[left].Valid() || !values[left].Equal(values[left]) {
			t.Fatalf("value %d is not self-identical", left)
		}
		for right := left + 1; right < len(values); right++ {
			if values[left].Equal(values[right]) {
				t.Fatalf("values %d and %d collapsed", left, right)
			}
		}
	}
}

func TestIntegerUsesOneSafeCanonicalSpelling(t *testing.T) {
	for _, valid := range []string{"0", "1", "-1", "9007199254740991", "-9007199254740991"} {
		value, err := Integer(valid)
		if err != nil {
			t.Fatalf("Integer(%q): %v", valid, err)
		}
		if got, ok := value.IntegerText(); !ok || got != valid {
			t.Fatalf("Integer(%q) = (%q,%t)", valid, got, ok)
		}
	}
	for _, invalid := range []string{"", "-0", "+1", "01", "-01", "1.0", "1e0", " 1", "9007199254740992", "-9007199254740992"} {
		if _, err := Integer(invalid); !IsCode(err, CodeInvalidInteger) {
			t.Fatalf("Integer(%q) error = %v", invalid, err)
		}
	}
}

func TestOrderedStringListPreservesOrderDuplicatesAndCopies(t *testing.T) {
	input := []string{"a", "", "a", "b"}
	value, err := OrderedStringList(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = "changed"
	got, ok := value.OrderedStrings()
	if !ok || got == nil || strings.Join(got, "|") != "a||a|b" {
		t.Fatalf("OrderedStrings() = (%v,%t)", got, ok)
	}
	got[0] = "changed-again"
	again, _ := value.OrderedStrings()
	if again[0] != "a" {
		t.Fatal("OrderedStrings exposed retained storage")
	}
	exact, ok := value.CanonicalStringListBytes()
	if !ok || string(exact) != `["a","","a","b"]` {
		t.Fatalf("CanonicalStringListBytes() = (%q,%t)", exact, ok)
	}
	reversed, err := OrderedStringList([]string{"b", "a", "a", ""})
	if err != nil {
		t.Fatal(err)
	}
	if value.Equal(reversed) {
		t.Fatal("ordered lists collapsed order")
	}
	empty, err := OrderedStringList([]string{})
	if err != nil {
		t.Fatal(err)
	}
	emptyMembers, ok := empty.OrderedStrings()
	if !ok || emptyMembers == nil || len(emptyMembers) != 0 {
		t.Fatalf("empty OrderedStrings() = (%#v,%t)", emptyMembers, ok)
	}
	if _, err := OrderedStringList(emptyMembers); err != nil {
		t.Fatalf("empty list getter did not round-trip: %v", err)
	}
}

func TestBytesPreserveNonUTF8AndCopy(t *testing.T) {
	input := []byte{0xff, 0x00, 0x80}
	value, err := Bytes(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = 0
	got, ok := value.BytesValue()
	if !ok || !bytes.Equal(got, []byte{0xff, 0x00, 0x80}) {
		t.Fatalf("BytesValue() = (%x,%t)", got, ok)
	}
	got[1] = 1
	again, _ := value.BytesValue()
	if !bytes.Equal(again, []byte{0xff, 0x00, 0x80}) {
		t.Fatal("BytesValue exposed retained storage")
	}
}

func TestCanonicalJSONRequiresExactBytesAndCopies(t *testing.T) {
	input := []byte(`{"a":[1,true],"z":"x"}`)
	value, err := CanonicalJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	input[2] = 'b'
	got, ok := value.CanonicalJSONBytes()
	if !ok || string(got) != `{"a":[1,true],"z":"x"}` {
		t.Fatalf("CanonicalJSONBytes() = (%q,%t)", got, ok)
	}
	got[2] = 'c'
	again, _ := value.CanonicalJSONBytes()
	if string(again) != `{"a":[1,true],"z":"x"}` {
		t.Fatal("CanonicalJSONBytes exposed retained storage")
	}
	for _, invalid := range [][]byte{
		[]byte(`{"z":"x","a":[1,true]}`),
		[]byte("{\"a\":1}\n"),
		[]byte(`{"a":1.0}`),
	} {
		if _, err := CanonicalJSON(invalid); err == nil {
			t.Fatalf("CanonicalJSON(%q) unexpectedly succeeded", invalid)
		}
	}
}

func TestEveryTagHasAnInjectiveIdentity(t *testing.T) {
	integer, _ := Integer("7")
	text, _ := String("7")
	bytesValue, _ := Bytes([]byte("7"))
	list, _ := OrderedStringList([]string{"7"})
	jsonValue, _ := CanonicalJSON([]byte(`"7"`))
	values := []Value{Missing(), Null(), Boolean(false), Boolean(true), integer, text, bytesValue, list, jsonValue}
	seen := map[string]bool{}
	for _, value := range values {
		identity := string(value.IdentityBytes())
		if identity == "" || seen[identity] {
			t.Fatalf("identity collision for %s", value.Tag())
		}
		seen[identity] = true
	}
}

func TestCompatibilityBudgetMeasuresExactJSONStringExpansion(t *testing.T) {
	text := "\x00\"\\é"
	value, err := String(text)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canon.String(text)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canonical.CanonicalChecked()
	if err != nil {
		t.Fatal(err)
	}
	if got := value.CompatibilityEncodedBytes(); got != len(exact) {
		t.Fatalf("CompatibilityEncodedBytes() = %d, want %d", got, len(exact))
	}
	missing := make([]Value, MaxTupleFields)
	for index := range missing {
		missing[index] = Missing()
	}
	if err := ValidateTuple(missing); err != nil {
		t.Fatalf("maximum field-count tuple should fit its encoded budget: %v", err)
	}
}

func TestEqualDoesNotReparseOrAllocateSealedPayloads(t *testing.T) {
	left, err := CanonicalJSON([]byte(`{"nested":["x",1,true]}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := CanonicalJSON([]byte(`{"nested":["x",1,true]}`))
	if err != nil {
		t.Fatal(err)
	}
	if allocations := testing.AllocsPerRun(1000, func() {
		if !left.Equal(right) {
			t.Fatal("equal canonical payloads differed")
		}
	}); allocations != 0 {
		t.Fatalf("Equal allocated %.2f times per call", allocations)
	}
}

func TestExplicitResourceCeilings(t *testing.T) {
	if _, err := String(strings.Repeat("x", MaxStringBytes+1)); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized string error = %v", err)
	}
	if _, err := Bytes(make([]byte, MaxBytesValueBytes+1)); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized bytes error = %v", err)
	}
	if _, err := OrderedStringList(make([]string, MaxListMembers+1)); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized list error = %v", err)
	}
	if _, err := OrderedStringList([]string{strings.Repeat("x", MaxListMemberBytes+1)}); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized member error = %v", err)
	}
	if _, err := OrderedStringList([]string{strings.Repeat("\x00", MaxListCanonicalBytes/5)}); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized canonical list encoding error = %v", err)
	}
	if _, err := CanonicalJSON(bytes.Repeat([]byte(" "), MaxCanonicalJSONBytes+1)); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("oversized JSON error = %v", err)
	}
	if _, err := CanonicalJSON(nil); !IsCode(err, CodeNoncanonicalJSON) {
		t.Fatalf("empty JSON error = %v", err)
	}
	if err := ValidateTuple(make([]Value, MaxTupleFields+1)); !IsCode(err, CodeInvalidTuple) {
		t.Fatalf("oversized tuple error = %v", err)
	}
	large, err := Bytes(make([]byte, MaxBytesValueBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTuple([]Value{large, large, large, large, large}); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("aggregate tuple error = %v", err)
	}
	expanding, err := String(strings.Repeat("\x00", MaxStringBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTuple([]Value{expanding}); !IsCode(err, CodeLimitExceeded) {
		t.Fatalf("JSON-escape expansion tuple error = %v", err)
	}
}

func TestZeroValueAndNilListAreInvalid(t *testing.T) {
	var zero Value
	if zero.Valid() || zero.IdentityBytes() != nil {
		t.Fatal("zero value acquired authority")
	}
	if _, err := OrderedStringList(nil); !IsCode(err, CodeInvalidTuple) {
		t.Fatalf("nil ordered list error = %v", err)
	}
	if err := ValidateTuple(nil); !IsCode(err, CodeInvalidTuple) {
		t.Fatalf("nil tuple error = %v", err)
	}
}

func TestCanonicalJSONUsesCanonSafeIntegerProfile(t *testing.T) {
	if _, err := CanonicalJSON([]byte(`9007199254740992`)); !IsCode(err, CodeNoncanonicalJSON) {
		t.Fatalf("unsafe JSON integer error = %v", err)
	}
	value, err := CanonicalJSON([]byte(`9007199254740991`))
	if err != nil || !value.Valid() {
		t.Fatalf("safe JSON integer = (%v,%v)", value, err)
	}
	parsed, err := canon.Parse([]byte(`9007199254740991`))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := parsed.CanonicalChecked()
	got, _ := value.CanonicalJSONBytes()
	if !bytes.Equal(got, want) {
		t.Fatalf("canonical JSON = %q, want %q", got, want)
	}
}
