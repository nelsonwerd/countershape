package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type schemaOverapproximationCase struct {
	name string
	body []byte
}

func schemaOverapproximationCases(t *testing.T, exact []byte) []schemaOverapproximationCase {
	t.Helper()
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			"duplicate evidence kind with distinct digest",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				refs := witness["process_closure"].(map[string]any)["evidence_refs"].([]any)
				refs[1].(map[string]any)["kind"] = "MATERIALIZATION_REVALIDATION"
			},
		},
		{
			"process evidence order",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				refs := witness["process_closure"].(map[string]any)["evidence_refs"].([]any)
				refs[0], refs[1] = refs[1], refs[0]
			},
		},
		{
			"clean process with primary",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				witness["process_closure"].(map[string]any)["primary_reason"] = "TIMEOUT"
			},
		},
		{
			"spawn process mismatch",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				witness["spawn_observation"] = map[string]any{"status": "START_ERROR", "error_code": "OS_START_ERROR"}
			},
		},
		{
			"scope order",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				checks := witness["standalone_scope"].(map[string]any)["checks"].([]any)
				checks[0], checks[1] = checks[1], checks[0]
			},
		},
		{
			"scope evidence role",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				checks := witness["standalone_scope"].(map[string]any)["checks"].([]any)
				checks[0].(map[string]any)["evidence_ref"].(map[string]any)["kind"] = "SERVICE_BINDINGS"
			},
		},
		{
			"scope aggregate",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				witness["standalone_scope"].(map[string]any)["status"] = "PARTIAL"
			},
		},
		{
			"private zero correlation",
			func(root map[string]any) {
				witness := root["closed_run_witness"].(map[string]any)
				private := witness["private_evidence"].(map[string]any)
				private["blob_count"] = int64(0)
				private["aggregate_byte_count"] = int64(1)
			},
		},
	}
	result := make([]schemaOverapproximationCase, 0, len(tests))
	for _, test := range tests {
		result = append(result, schemaOverapproximationCase{
			name: test.name,
			body: mutateCanonical(t, exact, test.mutate),
		})
	}
	return result
}

func c1RepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readCanonicalExample(t *testing.T, name string) []byte {
	t.Helper()
	pretty, err := os.ReadFile(filepath.Join(c1RepositoryRoot(t), "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.Canonicalize(pretty)
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func parseCheckedExampleGraph(t *testing.T) (ContractExecutionTarget, FinalizedContractRun, ContractExecution) {
	t.Helper()
	bundle := testBundle(t)
	targetExact := readCanonicalExample(t, "contract-execution-target.valid.json")
	targetDigest := digestForBytes(t, TargetKind, targetExact)
	target, err := ParseContractExecutionTarget(targetExact, targetDigest)
	if err != nil {
		t.Fatalf("checked target example: %v", err)
	}
	runExact := readCanonicalExample(t, "finalized-contract-run.valid.json")
	runDigest := digestForBytes(t, RunKind, runExact)
	run, err := ParseFinalizedContractRun(runExact, runDigest, target)
	if err != nil {
		t.Fatalf("checked run example: %v", err)
	}
	executionExact := readCanonicalExample(t, "contract-execution.valid.json")
	executionDigest := digestForBytes(t, ExecutionKind, executionExact)
	execution, err := ParseContractExecution(executionExact, executionDigest, bundle, target, run)
	if err != nil {
		t.Fatalf("checked execution example: %v", err)
	}
	return target, run, execution
}

func digestForBytes(t *testing.T, kind string, exact []byte) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes(kind, exact)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestCheckedExamplesParseRebuildAndClassifyExactly(t *testing.T) {
	target, run, execution := parseCheckedExampleGraph(t)
	for _, item := range []struct {
		name  string
		exact []byte
		got   []byte
	}{
		{"target", readCanonicalExample(t, "contract-execution-target.valid.json"), target.CanonicalBytes()},
		{"run", readCanonicalExample(t, "finalized-contract-run.valid.json"), run.CanonicalBytes()},
		{"execution", readCanonicalExample(t, "contract-execution.valid.json"), execution.CanonicalBytes()},
	} {
		if !bytes.Equal(item.exact, item.got) {
			t.Fatalf("%s checked example did not rebuild byte-exactly", item.name)
		}
	}
	if run.Disposition() != DispositionEligibleClean || execution.Result() != ResultConforms {
		t.Fatalf("checked graph disposition=%q result=%q", run.Disposition(), execution.Result())
	}
}

func TestSchemaProjectionOverapproximationsRefuseAtRuntime(t *testing.T) {
	target, run, _ := parseCheckedExampleGraph(t)
	for _, test := range schemaOverapproximationCases(t, run.CanonicalBytes()) {
		t.Run(test.name, func(t *testing.T) {
			digest := typedDigestForTest(t, RunKind, test.body)
			if _, err := ParseFinalizedContractRun(test.body, digest, target); !IsCode(err, CodeInvalidRun) {
				t.Fatalf("schema-overapproximation mutation error = %v", err)
			}
		})
	}
}

func TestC1SchemaRuntimeOverapproximationProbe(t *testing.T) {
	if os.Getenv("COUNTERSHAPE_C1_SCHEMA_RUNTIME_PROBE") != "1" {
		t.Skip("schema/runtime intersection probe is opt-in")
	}
	type wireCase struct {
		Name       string `json:"name"`
		BodyBase64 string `json:"body_base64"`
	}
	type wirePayload struct {
		SchemaVersion string     `json:"schema_version"`
		Cases         []wireCase `json:"cases"`
	}
	encodedPayload := os.Getenv("COUNTERSHAPE_C1_SCHEMA_RUNTIME_PAYLOAD_BASE64")
	if encodedPayload == "" || len(encodedPayload) > 3*1024*1024 {
		t.Fatal("schema/runtime probe payload is absent or exceeds its encoded ceiling")
	}
	payloadBytes, err := base64.StdEncoding.Strict().DecodeString(encodedPayload)
	if err != nil || len(payloadBytes) > 2*1024*1024 {
		t.Fatalf("decode schema/runtime probe payload: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payloadBytes))
	decoder.DisallowUnknownFields()
	var payload wirePayload
	if err := decoder.Decode(&payload); err != nil {
		t.Fatalf("decode schema/runtime probe payload: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("schema/runtime probe payload has trailing JSON: %v", err)
	}
	if payload.SchemaVersion != "countershape/c1-schema-runtime-overapproximation-probe/v1" {
		t.Fatalf("schema/runtime probe version = %q", payload.SchemaVersion)
	}
	target, run, _ := parseCheckedExampleGraph(t)
	expected := schemaOverapproximationCases(t, run.CanonicalBytes())
	if len(payload.Cases) != len(expected) {
		t.Fatalf("schema/runtime probe cases = %d, want %d", len(payload.Cases), len(expected))
	}
	corpusHash := sha256.New()
	for index, received := range payload.Cases {
		want := expected[index]
		if received.Name != want.name {
			t.Fatalf("schema/runtime probe case %d = %q, want %q", index, received.Name, want.name)
		}
		body, err := base64.StdEncoding.Strict().DecodeString(received.BodyBase64)
		if err != nil {
			t.Fatalf("schema/runtime probe case %q body: %v", received.Name, err)
		}
		if !bytes.Equal(body, want.body) {
			t.Fatalf("schema/runtime probe case %q differs from the Go corpus", received.Name)
		}
		corpusHash.Write([]byte(received.Name))
		corpusHash.Write([]byte{0})
		corpusHash.Write(body)
		corpusHash.Write([]byte{0})
		digest := typedDigestForTest(t, RunKind, body)
		if _, err := ParseFinalizedContractRun(body, digest, target); !IsCode(err, CodeInvalidRun) {
			t.Fatalf("schema/runtime probe case %q error = %v", received.Name, err)
		}
	}
	fmt.Printf("COUNTERSHAPE_C1_SCHEMA_RUNTIME_OVERAPPROXIMATION_V1|%d|sha256:%x\n", len(expected), corpusHash.Sum(nil))
}

func TestAllStartErrorCodesRoundTripWithExactClosure(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	partial := [5]ScopeCheckState{ScopeCheckMissing, ScopeCheckMissing, ScopeCheckMissing, ScopeCheckMissing, ScopeCheckMissing}
	for _, code := range []string{
		"OS_START_ERROR",
		"PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
		"PRESPAWN_RUNTIME_REVALIDATION_FAILED",
	} {
		t.Run(code, func(t *testing.T) {
			spawn, err := NewStartErrorObservation(code)
			if err != nil {
				t.Fatal(err)
			}
			process, err := NewControlledProcessClosure(
				domain.ControlStartError,
				false,
				false,
				testProcessEvidence(t, false),
			)
			if err != nil {
				t.Fatal(err)
			}
			witness, err := NewClosedRunWitness(
				spawn,
				process,
				NewNoCapture(),
				testScope(t, partial),
				testPrivateManifest(t),
			)
			if err != nil {
				t.Fatal(err)
			}
			run, err := NewFinalizedContractRun(target, testDigest('9'), witness)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := ParseFinalizedContractRun(run.CanonicalBytes(), run.Digest(), target)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := parsed.Witness().SpawnObservation().ErrorCode()
			if !ok || got != code || parsed.Disposition() != DispositionIneligibleControlStandalone {
				t.Fatalf("code=%q disposition=%q", got, parsed.Disposition())
			}
		})
	}
}
