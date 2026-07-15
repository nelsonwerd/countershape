package gitobj

import (
	"encoding/hex"
	"testing"
)

func FuzzTreeListingParserNeverManufacturesUnsupportedEntries(f *testing.F) {
	sha1OID, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	sha256OID, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	f.Add(append(append([]byte("100644 file.txt"), 0), sha1OID...), false)
	f.Add(append(append([]byte("120000 link"), 0), sha1OID...), false)
	f.Add(append(append([]byte("100755 tool"), 0), sha256OID...), true)
	f.Fuzz(func(t *testing.T, input []byte, sha256Format bool) {
		format := ObjectSHA1
		if sha256Format {
			format = ObjectSHA256
		}
		parsed, err := parseRawTreeObject(format, input, newTreeParseBudget(64))
		if err != nil {
			return
		}
		if len(parsed) < 1 || len(parsed) > 512 {
			t.Fatalf("successful parser returned %d entries", len(parsed))
		}
		for _, entry := range parsed {
			if !validOID(format, entry.oid) {
				t.Fatalf("successful parser manufactured OID %q", entry.oid)
			}
			if len(entry.name) == 0 {
				t.Fatal("successful parser manufactured empty component")
			}
		}
	})
}

func FuzzRawPathValidationIsStable(f *testing.F) {
	for _, seed := range []string{"file.txt", "dir/tool", "../escape", ".git/config", "A/é", "A/e\u0301"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		left := validateRawPath(input)
		right := validateRawPath(append([]byte(nil), input...))
		if (left == nil) != (right == nil) {
			t.Fatalf("path admission changed across identical bytes: left=%v right=%v", left, right)
		}
	})
}
