package portablevalue

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
)

func FuzzBytesRoundTripAndCopy(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x00, 0x80})
	f.Fuzz(func(t *testing.T, input []byte) {
		original := append([]byte(nil), input...)
		value, err := Bytes(input)
		if len(input) > MaxBytesValueBytes {
			if !IsCode(err, CodeLimitExceeded) {
				t.Fatalf("oversized Bytes error = %v", err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(input) > 0 {
			input[0] ^= 0xff
		}
		got, ok := value.BytesValue()
		if !ok || !bytes.Equal(got, original) {
			t.Fatalf("BytesValue() = (%x,%t), want %x", got, ok, original)
		}
	})
}

func FuzzStringRoundTrip(f *testing.F) {
	f.Add("")
	f.Add("plain text")
	f.Add("snowman ☃")
	f.Add(string([]byte{0xff}))
	f.Fuzz(func(t *testing.T, input string) {
		value, err := String(input)
		wantValid := len(input) <= MaxStringBytes && utf8.ValidString(input)
		if (err == nil) != wantValid {
			t.Fatalf("String(%q) error = %v, want valid=%t", input, err, wantValid)
		}
		if err != nil {
			return
		}
		got, ok := value.StringText()
		if !ok || got != input || !value.Valid() {
			t.Fatalf("accepted string did not round-trip: (%q,%t)", got, ok)
		}
	})
}

func FuzzIntegerIsCanonical(f *testing.F) {
	for _, seed := range []string{"0", "-0", "01", "1", "-1", "1e0", "9007199254740991"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		value, err := Integer(input)
		_, canonErr := canon.IntegerFromString(input)
		wantValid := len(input) <= 32 && utf8.ValidString(input) && canonErr == nil
		if (err == nil) != wantValid {
			t.Fatalf("Integer(%q) error = %v; canon error = %v", input, err, canonErr)
		}
		if err != nil {
			return
		}
		got, ok := value.IntegerText()
		if !ok || got != input || !value.Valid() {
			t.Fatalf("accepted integer did not round-trip: (%q,%t)", got, ok)
		}
	})
}

func FuzzCanonicalJSONOnlyAcceptsExact(f *testing.F) {
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`{"a":1 }`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, input []byte) {
		value, err := CanonicalJSON(input)
		parsed, parseErr := canon.Parse(input)
		var checked []byte
		if parseErr == nil {
			checked, parseErr = parsed.CanonicalChecked()
		}
		wantValid := len(input) > 0 && len(input) <= MaxCanonicalJSONBytes && parseErr == nil && bytes.Equal(checked, input)
		if (err == nil) != wantValid {
			t.Fatalf("CanonicalJSON(%q) error = %v, canonical oracle error = %v", input, err, parseErr)
		}
		if err != nil {
			return
		}
		got, ok := value.CanonicalJSONBytes()
		if !ok || !bytes.Equal(got, input) || !value.Valid() {
			t.Fatal("accepted canonical JSON did not preserve exact bytes")
		}
	})
}

func FuzzOrderedStringListPreservesMembers(f *testing.F) {
	f.Add("", "a", "a")
	f.Add("a", "b", "a")
	f.Add("\x00", "\\", "\"")
	f.Fuzz(func(t *testing.T, first, second, third string) {
		input := []string{first, second, third, first}
		value, err := OrderedStringList(input)
		retained := 0
		wantValid := true
		for _, member := range input {
			if !utf8.ValidString(member) || len(member) > MaxListMemberBytes {
				wantValid = false
			}
			retained += len(member)
		}
		canonical, canonicalErr := canon.CanonicalizeTyped(input)
		wantValid = wantValid && len(input) <= MaxListMembers && retained <= MaxListAggregateBytes &&
			canonicalErr == nil && len(canonical) <= MaxListCanonicalBytes
		if (err == nil) != wantValid {
			t.Fatalf("OrderedStringList(%q) error = %v, want valid=%t, canonical error=%v", input, err, wantValid, canonicalErr)
		}
		if err != nil {
			return
		}
		input[0] = "mutated"
		got, ok := value.OrderedStrings()
		if !ok || len(got) != 4 || got[0] != first || got[1] != second || got[2] != third || got[3] != first {
			t.Fatalf("OrderedStrings() = (%q,%t)", got, ok)
		}
		rebuilt, err := OrderedStringList(got)
		if err != nil || !value.Equal(rebuilt) {
			t.Fatalf("ordered list failed exact rebuild: %v", err)
		}
	})
}
