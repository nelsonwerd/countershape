package model

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func fuzzDigest(kind string, body []byte) (domain.Digest, bool) {
	digest, err := canon.DigestBytes(kind, body)
	if err != nil {
		return "", false
	}
	parsed, err := domain.ParseDigest(digest.String())
	return parsed, err == nil
}

func FuzzContractExecutionTargetParser(f *testing.F) {
	bundle := testBundle(f)
	valid := testTarget(f, bundle)
	f.Add(valid.CanonicalBytes())
	f.Add([]byte("{}"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > canon.MaxInputBytes+1 {
			return
		}
		digest, ok := fuzzDigest(TargetKind, body)
		if !ok {
			return
		}
		parsed, err := ParseContractExecutionTarget(body, digest)
		if err != nil {
			return
		}
		if !parsed.Valid() || !bytes.Equal(parsed.CanonicalBytes(), body) || parsed.Digest() != digest {
			t.Fatal("accepted target was not exact, stable, and valid")
		}
		reparsed, err := ParseContractExecutionTarget(parsed.CanonicalBytes(), parsed.Digest())
		if err != nil || !reparsed.Equal(parsed) {
			t.Fatalf("accepted target did not reparse exactly: %v", err)
		}
	})
}

func FuzzFinalizedContractRunParser(f *testing.F) {
	bundle := testBundle(f)
	target := testTarget(f, bundle)
	valid, err := NewFinalizedContractRun(target, testDigest('9'), testCleanWitness(f, bundle))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid.CanonicalBytes())
	f.Add([]byte("{}"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > canon.MaxInputBytes+1 {
			return
		}
		digest, ok := fuzzDigest(RunKind, body)
		if !ok {
			return
		}
		parsed, err := ParseFinalizedContractRun(body, digest, target)
		if err != nil {
			return
		}
		if !parsed.Valid() || !bytes.Equal(parsed.CanonicalBytes(), body) || parsed.Digest() != digest {
			t.Fatal("accepted finalized run was not exact, stable, and valid")
		}
		reparsed, err := ParseFinalizedContractRun(parsed.CanonicalBytes(), parsed.Digest(), target)
		if err != nil || !reparsed.Equal(parsed) {
			t.Fatalf("accepted finalized run did not reparse exactly: %v", err)
		}
	})
}

func FuzzContractExecutionParser(f *testing.F) {
	bundle := testBundle(f)
	target := testTarget(f, bundle)
	run, err := NewFinalizedContractRun(target, testDigest('9'), testCleanWitness(f, bundle))
	if err != nil {
		f.Fatal(err)
	}
	valid, err := DeriveContractExecution(bundle, target, run)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid.CanonicalBytes())
	f.Add([]byte("{}"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > canon.MaxInputBytes+1 {
			return
		}
		digest, ok := fuzzDigest(ExecutionKind, body)
		if !ok {
			return
		}
		parsed, err := ParseContractExecution(body, digest, bundle, target, run)
		if err != nil {
			return
		}
		if !parsed.Valid() || !bytes.Equal(parsed.CanonicalBytes(), body) || parsed.Digest() != digest {
			t.Fatal("accepted execution was not exact, stable, and valid")
		}
		reparsed, err := ParseContractExecution(parsed.CanonicalBytes(), parsed.Digest(), bundle, target, run)
		if err != nil || !reparsed.Equal(parsed) {
			t.Fatalf("accepted execution did not reparse exactly: %v", err)
		}
	})
}
