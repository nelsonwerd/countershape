package http

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestHTTPProjectionMandatesStatusAndDisclosureMetadataMutationGuard(t *testing.T) {
	definition, err := NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	fields := definition.Fields()
	wantFields := []HTTPFieldID{
		HTTPFieldStatus,
		HTTPFieldContentType,
		HTTPFieldBodyKind,
		HTTPFieldBodyMetadata,
	}
	if len(fields) != len(wantFields) {
		t.Fatalf("fixed projection fields = %v; status and disclosure metadata must remain mandatory", fields)
	}
	for index := range fields {
		if fields[index] != wantFields[index] {
			t.Fatalf("fixed projection fields = %v, want %v", fields, wantFields)
		}
	}
	body := []byte(`{"request_id":"request-id-must-not-leak","scratch_root":"/tmp/scratch-root-must-not-leak","kind":"invoice.selected","metadata":{"disclosed":["amount"],"owner":"tenant-selected"}}`)
	response := mustParsedHTTPResponse(t, 500, "Internal Server Error", body)
	projected, err := ProjectParsedResponse(response, "/tmp/scratch-root-must-not-leak")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range [][]byte{
		[]byte(`"FieldID":"` + string(HTTPFieldStatus) + `"`),
		[]byte(`"FieldID":"` + string(HTTPFieldContentType) + `"`),
		[]byte(`"FieldID":"` + string(HTTPFieldBodyKind) + `"`),
		[]byte(`"FieldID":"` + string(HTTPFieldBodyMetadata) + `"`),
		[]byte(`"Integer":500`),
		[]byte(`application/json`),
		[]byte(`invoice.selected`),
		[]byte(`tenant-selected`),
		[]byte(`amount`),
	} {
		if !bytes.Contains(projected, required) {
			t.Fatalf("canonical projection omitted selected source %q: %s", required, projected)
		}
	}
	for _, forbidden := range [][]byte{
		[]byte(`request_id`),
		[]byte(`request-id-must-not-leak`),
		[]byte(`scratch_root`),
		[]byte(`/tmp/scratch-root-must-not-leak`),
		[]byte(`placeholder`),
	} {
		if bytes.Contains(bytes.ToLower(projected), bytes.ToLower(forbidden)) {
			t.Fatalf("validate-then-omit source %q leaked into projection bytes: %s", forbidden, projected)
		}
	}
}

func TestHTTPProjectionTranscriptNamesTruthfulValidateThenOmitOperations(t *testing.T) {
	definition, err := NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	operations := definition.Operations()
	if len(operations) <= 5 {
		t.Fatalf("projection transcript has only %d operations", len(operations))
	}
	if got := operations[4].Name(); got != "http.validate-then-omit-request-id/v1" {
		t.Fatalf("request id validate-then-omit is not visible: %q", got)
	}
	if got := operations[4].Semantics(); got != "validate the top-level request_id value as a string, then omit that member from projection output" {
		t.Fatalf("request id operation claims different behavior: %q", got)
	}
	if got := operations[5].Name(); got != "http.validate-then-omit-scratch-root/v1" {
		t.Fatalf("scratch root validate-then-omit is not visible: %q", got)
	}
	if got := operations[5].Semantics(); got != "validate the top-level scratch_root value as a string exactly equal to the captured runtime scratch-root authority, then omit that member from projection output" {
		t.Fatalf("scratch root operation claims different behavior: %q", got)
	}
}

func TestHTTPProjectionRuntimeTranscriptUsesExactSourcesForEveryOperation(t *testing.T) {
	definition, observation := mustProjectableHTTPObservation(t, nil)
	projected, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	transcript := projected.Transcript()
	if !projected.Valid() || len(transcript) != len(definition.Operations()) {
		t.Fatal("runtime projection did not retain the complete sealed transcript")
	}
	want := [][]HTTPProjectionSourceLink{
		{
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "controls"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "transport_kind"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "readiness_accepted"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "request_attempted"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "request_complete"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "response_parsed"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "seed_overlay_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "readiness_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "exchange_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "invocation_receipt_presence"),
		},
		{httpSourceLink(HTTPChannelStatus, "status")},
		{httpSourceLink(HTTPChannelHeaders, "content-type")},
		{httpSourceLink(HTTPChannelBody, "captured_bytes")},
		{httpSourceLink(HTTPChannelBody, "strict_json", "request_id")},
		{
			httpSourceLink(HTTPChannelBody, "strict_json", "scratch_root"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "scratch_root"),
		},
		{httpSourceLink(HTTPChannelBody, "strict_json", "kind")},
		{httpSourceLink(HTTPChannelBody, "strict_json", "metadata")},
		{
			httpSourceLink(HTTPChannelStatus, "status"),
			httpSourceLink(HTTPChannelHeaders, "content-type"),
			httpSourceLink(HTTPChannelBody, "strict_json", "kind"),
			httpSourceLink(HTTPChannelBody, "strict_json", "metadata"),
		},
	}
	if len(transcript) != len(want) {
		t.Fatalf("transcript entries = %d, want %d", len(transcript), len(want))
	}
	for index := range want {
		assertHTTPSourceLinks(t, transcript[index].SourceLinks(), want[index])
	}
	if got := definition.Binding().AcceptedChannels(); !httpStringsEqual(got, []string{"http.body", "http.headers", "http.status"}) {
		t.Fatalf("selected projection channels = %q; trace-only runtime provenance must not expand the fixed output tuple", got)
	}
}

func TestHTTPProjectionResultRejectsRehashedGenericOrTamperedSourceLinks(t *testing.T) {
	definition, observation := mustProjectableHTTPObservation(t, nil)
	projected, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*HTTPProjectionResult)
	}{
		{
			name: "generic-request-id-body-link",
			mutate: func(result *HTTPProjectionResult) {
				result.transcript[4].sourceLinks = []HTTPProjectionSourceLink{httpSourceLink(HTTPChannelBody, "body")}
			},
		},
		{
			name: "missing-runtime-scratch-root-authority",
			mutate: func(result *HTTPProjectionResult) {
				result.transcript[5].sourceLinks = result.transcript[5].sourceLinks[:1]
			},
		},
		{
			name: "generic-final-encode-link",
			mutate: func(result *HTTPProjectionResult) {
				result.transcript[8].sourceLinks = []HTTPProjectionSourceLink{httpSourceLink(HTTPChannelBody, "body")}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tampered := projected
			tampered.transcript = cloneTrace(projected.transcript)
			test.mutate(&tampered)
			rehashHTTPProjectionResultForTest(t, &tampered)
			if tampered.Valid() {
				t.Fatal("rehashed projection with non-exact source links remained valid")
			}
		})
	}
}

func TestHTTPProjectionTranscriptAccessorsDeepCloneSelectors(t *testing.T) {
	definition, observation := mustProjectableHTTPObservation(t, nil)
	projected, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	transcriptCopy := projected.Transcript()
	transcriptCopy[5].sourceLinks[0].selector[0] = "tampered-copy"
	linksCopy := projected.Transcript()[5].SourceLinks()
	linksCopy[0].selector[0] = "tampered-link-copy"
	if !projected.Valid() {
		t.Fatal("mutating accessor-owned selector copies changed the sealed result")
	}
	fresh := projected.Transcript()[5].SourceLinks()
	if fresh[0].Selector()[0] != "strict_json" || fresh[1].Selector()[0] != "captured_observation" {
		t.Fatalf("source-link cloning leaked a selector mutation: %#v", fresh)
	}
}

func TestHTTPProjectionRefusesTeardownControlledResponseMutationGuard(t *testing.T) {
	definition, observation := mustProjectableHTTPObservation(
		t, []domain.ControlReason{domain.ControlTeardownError},
	)
	_, err := definition.Project(observation)
	var rejection *ProjectionRejection
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionControl {
		t.Fatalf("teardown-controlled complete response became eligible: %v", err)
	}
}

func TestProjectParsedResponseRefusesZeroResponseAndNonCanonicalRoot(t *testing.T) {
	if _, err := ProjectParsedResponse(HTTPResponse{}, "/tmp/root"); err == nil {
		t.Fatal("zero response was accepted by the pure projection seam")
	}
	body := []byte(`{"request_id":"volatile","scratch_root":"/tmp/root","kind":"invoice","metadata":{}}`)
	response := mustParsedHTTPResponse(t, 200, "OK", body)
	for _, root := range []string{"", "relative", "/tmp/../tmp/root", "/tmp/root/"} {
		if _, err := ProjectParsedResponse(response, root); err == nil {
			t.Fatalf("noncanonical expected scratch root %q was accepted", root)
		}
	}
}

func mustParsedHTTPResponse(t *testing.T, status int, reason string, body []byte) HTTPResponse {
	t.Helper()
	policy, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\ncontent-type: application/json\r\ncontent-length: %d\r\n\r\n%s", status, reason, len(body), body))
	response, err := ParseResponse(raw, policy)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func mustProjectableHTTPObservation(
	t *testing.T,
	controls []domain.ControlReason,
) (HTTPProjectionDefinition, HTTPCapturedObservation) {
	t.Helper()
	definition, err := NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	input := validCapturedInput(t)
	input.AdapterProjectionDefinitionDigest = definition.Digest()
	input.ProjectionDefinitionDigest = definition.Binding().Digest()
	input.TransportKind = TransportCompleteResponse
	input.Response = mustParsedHTTPResponse(
		t, 200, "OK",
		[]byte(`{"request_id":"volatile","scratch_root":"/tmp/http-capture","kind":"invoice","metadata":{"owner":"tenant-b"}}`),
	)
	input.Controls = append([]domain.ControlReason(nil), controls...)
	input.SeedOverlayReceiptPresent = true
	input.SeedOverlayReceiptDigest = testHTTPDigest(t, "projection-seed")
	input.ReadinessReceiptPresent = true
	input.ReadinessReceiptDigest = testHTTPDigest(t, "projection-readiness")
	input.ExchangeReceiptPresent = true
	input.ExchangeReceiptDigest = testHTTPDigest(t, "projection-exchange")
	if len(controls) == 0 {
		input.InvocationReceiptPresent = true
		input.InvocationReceiptDigest = testHTTPDigest(t, "projection-invocation")
	}
	input.ReadinessAccepted = true
	input.RequestAttempted = true
	input.RequestComplete = true
	input.ResponseParsed = true
	observation, err := newCapturedObservation(input)
	if err != nil {
		t.Fatal(err)
	}
	return definition, observation
}

func assertHTTPSourceLinks(t *testing.T, got, want []HTTPProjectionSourceLink) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("source links = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index].Channel() != want[index].Channel() || !httpStringsEqual(got[index].Selector(), want[index].Selector()) {
			t.Fatalf("source link %d = (%q, %q), want (%q, %q)", index, got[index].Channel(), got[index].Selector(), want[index].Channel(), want[index].Selector())
		}
	}
}

func httpStringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func rehashHTTPProjectionResultForTest(t *testing.T, result *HTTPProjectionResult) {
	t.Helper()
	projectionByteDigest, err := digestBytes("HTTPProjectionCanonicalBytes", result.projectionBytes)
	if err != nil {
		t.Fatal(err)
	}
	identity := struct {
		SchemaVersion, Kind, ObservationDigest, DefinitionDigest, ProjectionByteDigest string
		Transcript                                                                     any
	}{
		domain.SchemaVersion,
		"HTTPProjectionDerivation",
		result.observationDigest.String(),
		result.definitionDigest.String(),
		projectionByteDigest.String(),
		traceIdentities(result.transcript),
	}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionDerivation", identity)
	if err != nil {
		t.Fatal(err)
	}
	result.digest = digest
	result.canonicalBytes = canonicalBytes
}
