package canon

import (
	"bytes"
	"testing"
)

func FuzzParseCanonicalIdempotence(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`null`),
		[]byte(`{"b":2,"a":1}`),
		[]byte(`{"a":1,"\u0061":2}`),
		[]byte(`[-0,0.0,9007199254740992]`),
		[]byte(`{"emoji":"\uD83D\uDE00"}`),
		{'"', 0xff, '"'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		value, err := Parse(input)
		if err != nil {
			return
		}
		canonical := value.Canonical()
		again, err := Parse(canonical)
		if err != nil {
			t.Fatalf("Parse(canonical %q) error = %v; original %q", canonical, err, input)
		}
		if !bytes.Equal(canonical, again.Canonical()) {
			t.Fatalf("canonical bytes are not idempotent: %q != %q", canonical, again.Canonical())
		}
		tokens, err := Scan(canonical)
		if err != nil {
			t.Fatalf("Scan(canonical %q) error = %v", canonical, err)
		}
		var rebuilt []byte
		for _, token := range tokens {
			rebuilt = append(rebuilt, token.Raw...)
		}
		if !bytes.Equal(rebuilt, canonical) {
			t.Fatalf("lossless tokens rebuilt %q, want %q", rebuilt, canonical)
		}
	})
}
