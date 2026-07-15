package model

import (
	"bytes"
	"testing"
)

func TestHTTPReadinessContractRequiresExactPipeByteAndForbidsHTTPProbeMutationGuard(t *testing.T) {
	contract, err := NewHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	if !contract.Valid() || contract.Protocol() != ReadinessProtocolV1 ||
		contract.SuccessByte() != ReadinessSuccessByte || contract.SignalName() != ReadinessSignalNameV1 {
		t.Fatal("readiness contract lost its exact inherited-pipe byte authority")
	}
	canonical := contract.CanonicalBytes()
	if !bytes.Contains(canonical, []byte(`"http_probe":false`)) ||
		!bytes.Contains(canonical, []byte(`"eof_required":true`)) ||
		!bytes.Contains(canonical, []byte(`"success_byte":1`)) ||
		!bytes.Contains(canonical, []byte(`"success_length":1`)) {
		t.Fatalf("readiness contract admits HTTP probing or lost exact byte+EOF facts: %s", canonical)
	}
}
