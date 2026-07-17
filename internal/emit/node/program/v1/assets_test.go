package program

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestFixedAssetsMatchReviewedRawSHA256Pins(t *testing.T) {
	tests := []struct {
		path     string
		reviewed string
		embedded *[]byte
	}{
		{path: ContractTestPath, reviewed: contractTestReviewedRawSHA256, embedded: &contractTestSource},
		{path: HarnessPath, reviewed: harnessReviewedRawSHA256, embedded: &harnessSource},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if !strings.HasPrefix(test.reviewed, "sha256:") || len(test.reviewed) != len("sha256:")+sha256.Size*2 {
				t.Fatalf("reviewed digest has the wrong shape: %q", test.reviewed)
			}
			digest := sha256.Sum256(*test.embedded)
			independent := "sha256:" + hex.EncodeToString(digest[:])
			if independent != test.reviewed {
				t.Fatalf("embedded bytes differ from reviewed pin: got %s want %s", independent, test.reviewed)
			}
			got, err := RawSHA256(test.path)
			if err != nil {
				t.Fatalf("RawSHA256: %v", err)
			}
			if got != test.reviewed {
				t.Fatalf("RawSHA256 = %q, want %q", got, test.reviewed)
			}
		})
	}
}

func TestFixedAssetLoaderIsDefensiveAndFailClosed(t *testing.T) {
	for _, path := range []string{ContractTestPath, HarnessPath} {
		first, err := Bytes(path)
		if err != nil {
			t.Fatalf("Bytes(%s) first: %v", path, err)
		}
		if len(first) == 0 {
			t.Fatalf("embedded asset %s is empty", path)
		}
		original := first[0]
		first[0] ^= 0xff
		second, err := Bytes(path)
		if err != nil {
			t.Fatalf("Bytes(%s) second: %v", path, err)
		}
		if second[0] != original {
			t.Fatalf("mutating a returned slice changed embedded asset %s", path)
		}
	}

	tampered := append([]byte(nil), harnessSource...)
	tampered[len(tampered)/2] ^= 1
	if err := verifyReviewedDigest(tampered, harnessReviewedRawSHA256); err == nil || !strings.Contains(err.Error(), "differs from its reviewed raw SHA-256") {
		t.Fatalf("tampered bytes were not rejected by the pure digest verifier: %v", err)
	}
	if _, err := Bytes("other.mjs"); err == nil || !strings.Contains(err.Error(), "unknown Node program asset") {
		t.Fatalf("unknown asset was not rejected: %v", err)
	}
	if got, err := RawSHA256("other.mjs"); err == nil || got != "" || !strings.Contains(err.Error(), "unknown Node program asset") {
		t.Fatalf("unknown asset digest was not rejected: digest=%q err=%v", got, err)
	}
}
