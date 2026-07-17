package parity

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func parityRequest(operation Operation, input string) []byte {
	return []byte(fmt.Sprintf(`{"input":%s,"operation":%q}`, input, operation))
}

func base64Input(exact []byte) string {
	return fmt.Sprintf(`{"bytes_base64":%q}`, base64.StdEncoding.EncodeToString(exact))
}

func httpParseInput(raw []byte, body, headerBytes, headerCount, statusLine int) string {
	return fmt.Sprintf(
		`{"body_bytes":%d,"header_bytes":%d,"header_count":%d,"response_base64":%q,"status_line_bytes":%d}`,
		body, headerBytes, headerCount, base64.StdEncoding.EncodeToString(raw), statusLine,
	)
}

func TestGoAndNodeParityEvaluatorsMatchLiteralOracle(t *testing.T) {
	digest := "sha256:" + strings.Repeat("0", 64)
	manifest := `{"files":[` +
		`{"byte_count":1,"byte_sha256":"` + digest + `","mode":"100644","path":"README.md"},` +
		`{"byte_count":1,"byte_sha256":"` + digest + `","mode":"100644","path":"contract.test.mjs"},` +
		`{"byte_count":1,"byte_sha256":"` + digest + `","mode":"100644","path":"decision.json"},` +
		`{"byte_count":1,"byte_sha256":"` + digest + `","mode":"100644","path":"fixture.json"},` +
		`{"byte_count":1,"byte_sha256":"` + digest + `","mode":"100644","path":"harness.mjs"}` +
		`],"kind":"IntegrityManifest","manifest_version":"countershape-manifest/v1","schema_version":"countershape-contract/v1"}`
	httpRaw := []byte("HTTP/1.1 500 Nope\r\ncontent-length: 0\r\ncontent-type: \r\ncontent-type: application/json\r\n\r\n")
	httpBodyLimitRaw := []byte("HTTP/1.1 200 OK\r\ncontent-length: 2\r\n\r\nhi")
	httpIntegerOverflowRaw := []byte("HTTP/1.1 200 OK\r\ncontent-length: 9223372036854775808\r\n\r\n")
	httpMalformedHeaderRaw := []byte("HTTP/1.1 200 OK\r\ncontent-length:0\r\n\r\n")
	httpHighBitStatusRaw := []byte("HTTP/1.1 200 OK\r\ncontent-length: 0\r\n\r\n")
	httpHighBitStatusRaw[9] |= 0x80
	httpBody := `{"kind":"ok","metadata":{"mode":"test"},"request_id":"r","scratch_root":"/tmp/root"}`
	httpProjectionRaw := []byte(fmt.Sprintf(
		"HTTP/1.1 200 OK\r\ncontent-length: %d\r\ncontent-type: application/json\r\n\r\n%s",
		len(httpBody), httpBody,
	))
	tupleExited := `{"fields":[{"field_id":"cli.completion.kind","value":{"tag":"STRING","value":"EXITED"}}]}`
	tupleSignaled := `{"fields":[{"field_id":"cli.completion.kind","value":{"tag":"STRING","value":"SIGNALED"}}]}`
	predicateInput := `{"adapter":"CLI","allowed_tuples":[` + tupleExited + `],"decision_action":"ALLOW_OBSERVED","observed_tuple":` + tupleExited +
		`,"profile_fields":["cli.completion.kind"],"selected_fields":["cli.completion.kind"],"stimulus_digest":"` + digest + `"}`
	unsortedPredicateInput := `{"adapter":"CLI","allowed_tuples":[` + tupleSignaled + `,` + tupleExited +
		`],"decision_action":"ALLOW_OBSERVED","observed_tuple":` + tupleExited +
		`,"profile_fields":["cli.completion.kind"],"selected_fields":["cli.completion.kind"],"stimulus_digest":"` + digest + `"}`

	tests := []struct {
		name      string
		operation Operation
		input     string
		want      string
	}{
		{
			"canonicalize normalization", CanonicalizeJSON, base64Input([]byte(` {"b":2,"a":1} `)),
			`{"status":"OK","value":{"canonical_base64":"eyJhIjoxLCJiIjoyfQ=="}}`,
		},
		{"canonicalize invalid base64", CanonicalizeJSON, `{"bytes_base64":"%%%"}`, `{"code":"INVALID_BASE64","status":"REFUSED"}`},
		{"canonicalize duplicate key", CanonicalizeJSON, base64Input([]byte(`{"a":1,"a":2}`)), `{"code":"INVALID_JSON","status":"REFUSED"}`},
		{"canonical parse rejects whitespace", ParseCanonicalJSON, base64Input([]byte(` {"a":1}`)), `{"code":"NONCANONICAL_JSON","status":"REFUSED"}`},
		{
			"manifest exact envelope", ParseManifestEnvelope, base64Input([]byte(manifest + "\n")),
			`{"status":"OK","value":{"manifest":` + manifest + `}}`,
		},
		{"manifest missing LF", ParseManifestEnvelope, base64Input([]byte(manifest)), `{"code":"INVALID_MANIFEST","status":"REFUSED"}`},
		{
			"readiness upper port", ParseReadyFrame,
			fmt.Sprintf(`{"eof_observed":true,"frame_base64":%q}`, base64.StdEncoding.EncodeToString([]byte("COUNTERSHAPE_READY_V1 65535\n"))),
			`{"status":"OK","value":{"port":65535}}`,
		},
		{
			"readiness requires EOF", ParseReadyFrame,
			fmt.Sprintf(`{"eof_observed":false,"frame_base64":%q}`, base64.StdEncoding.EncodeToString([]byte("COUNTERSHAPE_READY_V1 1\n"))),
			`{"code":"READINESS_FAILED","status":"REFUSED"}`,
		},
		{
			"HTTP exact response", ParseHTTPResponse, httpParseInput(httpRaw, 1024, 4096, 32, 1024),
			`{"status":"OK","value":{"body_base64":"","content_length":0,"headers":[{"name":"content-length","value":"0"},{"name":"content-type","value":""},{"name":"content-type","value":"application/json"}],"reason":"Nope","status":500}}`,
		},
		{
			"HTTP total overflow", ParseHTTPResponse, httpParseInput(httpRaw, 1, 16, 1, 16),
			`{"code":"OUTPUT_LIMIT","status":"REFUSED"}`,
		},
		{
			"HTTP status line limit", ParseHTTPResponse, httpParseInput(httpRaw, 1024, 4096, 32, 16),
			`{"code":"OUTPUT_LIMIT","status":"REFUSED"}`,
		},
		{
			"HTTP header bytes limit", ParseHTTPResponse, httpParseInput(httpRaw, 1024, 16, 32, 1024),
			`{"code":"OUTPUT_LIMIT","status":"REFUSED"}`,
		},
		{
			"HTTP header count limit", ParseHTTPResponse, httpParseInput(httpRaw, 1024, 4096, 1, 1024),
			`{"code":"OUTPUT_LIMIT","status":"REFUSED"}`,
		},
		{
			"HTTP declared body limit", ParseHTTPResponse, httpParseInput(httpBodyLimitRaw, 1, 4096, 32, 1024),
			`{"code":"OUTPUT_LIMIT","status":"REFUSED"}`,
		},
		{
			"HTTP content length integer overflow", ParseHTTPResponse, httpParseInput(httpIntegerOverflowRaw, 1024, 4096, 32, 1024),
			`{"code":"RESPONSE_PARSE_FAILED","status":"REFUSED"}`,
		},
		{
			"HTTP malformed header", ParseHTTPResponse, httpParseInput(httpMalformedHeaderRaw, 1024, 4096, 32, 1024),
			`{"code":"RESPONSE_PARSE_FAILED","status":"REFUSED"}`,
		},
		{
			"HTTP high bit status alias", ParseHTTPResponse, httpParseInput(httpHighBitStatusRaw, 1024, 4096, 32, 1024),
			`{"code":"RESPONSE_PARSE_FAILED","status":"REFUSED"}`,
		},
		{
			"CLI exact projection", ProjectCLIObservation,
			fmt.Sprintf(`{"completion":{"code":2,"kind":"EXITED"},"selected_fields":["cli.completion.kind","cli.exit.code","cli.stdout.bytes"],"stderr_base64":"","stdout_base64":%q}`, base64.StdEncoding.EncodeToString([]byte("opaque"))),
			`{"status":"OK","value":{"tuple":{"fields":[{"field_id":"cli.completion.kind","value":{"tag":"STRING","value":"EXITED"}},{"field_id":"cli.exit.code","value":{"canonical":"2","tag":"INTEGER"}},{"field_id":"cli.stdout.bytes","value":{"base64":"b3BhcXVl","tag":"BYTES"}}]}}}`,
		},
		{
			"CLI selected stderr rejects invalid UTF8", ProjectCLIObservation,
			`{"completion":{"code":0,"kind":"EXITED"},"selected_fields":["cli.stderr.text"],"stderr_base64":"/w==","stdout_base64":""}`,
			`{"code":"PROJECTION_FAILED","status":"REFUSED"}`,
		},
		{
			"HTTP selected projection", ProjectHTTPObservation,
			fmt.Sprintf(`{"body_bytes":1024,"header_bytes":4096,"header_count":32,"response_base64":%q,"scratch_root":"/tmp/root","selected_fields":["http.body.kind"],"status_line_bytes":1024}`, base64.StdEncoding.EncodeToString(httpProjectionRaw)),
			`{"status":"OK","value":{"tuple":{"fields":[{"field_id":"http.body.kind","value":{"tag":"STRING","value":"ok"}}]}}}`,
		},
		{
			"HTTP scratch root trailing slash", ProjectHTTPObservation,
			fmt.Sprintf(`{"body_bytes":1024,"header_bytes":4096,"header_count":32,"response_base64":%q,"scratch_root":"/tmp/root/","selected_fields":["http.body.kind"],"status_line_bytes":1024}`, base64.StdEncoding.EncodeToString(httpProjectionRaw)),
			`{"code":"INVALID_OPERATION_INPUT","status":"REFUSED"}`,
		},
		{"predicate correlated match", EvaluateExactPredicate, predicateInput, `{"status":"OK","value":{"match":true}}`},
		{"predicate rejects unsorted allowed tuples", EvaluateExactPredicate, unsortedPredicateInput, `{"code":"INVALID_PREDICATE","status":"REFUSED"}`},
		{
			"direct safety precedence", SelectDirectResult,
			`{"ineligible_reasons":["TIMEOUT","ORPHAN_RISK"],"internal_failure":true,"predicate_match":true,"tamper":true}`,
			`{"status":"OK","value":{"outcome":"INELIGIBLE_EXECUTION","reason":"ORPHAN_RISK"}}`,
		},
		{
			"owner cancellation", SelectOwnerEligibility,
			`{"kind":"CONTROL_INELIGIBLE","reasons":["CANCELLED","TEARDOWN_ERROR"]}`,
			`{"status":"OK","value":{"eligibility":"INELIGIBLE","reasons":["CANCELLED","TEARDOWN_ERROR"]}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestBytes := parityRequest(test.operation, test.input)
			request, err := ParseRequest(requestBytes)
			if err != nil {
				t.Fatalf("ParseRequest() = %v\n%s", err, requestBytes)
			}
			result := Evaluate(request)
			if got := string(result.CanonicalBytes()); got != test.want {
				t.Fatalf("Go result:\n%s\nwant:\n%s", got, test.want)
			}
			frame, err := result.FrameBytes()
			if err != nil || string(frame) != test.want+"\n" {
				t.Fatalf("Go frame = %q, %v; want %q", frame, err, test.want+"\n")
			}
			if got := runNodeParity(t, requestBytes); got != test.want+"\n" {
				t.Fatalf("Node result:\n%s\nwant:\n%s", got, test.want+"\n")
			}
		})
	}
}

func runNodeParity(t *testing.T, request []byte) string {
	t.Helper()
	stdout, stderr, runErr, contextErr := runNodeParityRaw(t, append(append([]byte(nil), request...), '\n'))
	if contextErr != nil || runErr != nil {
		t.Fatalf("Node parity runner failed: context=%v err=%v stdout=%q stderr=%q", contextErr, runErr, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 || stdout.Len() > MaxFrameBytes {
		t.Fatalf("Node parity framing: stdout=%d stderr=%q", stdout.Len(), stderr.String())
	}
	return stdout.String()
}

func TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr(t *testing.T) {
	stdout, stderr, runErr, contextErr := runNodeParityRaw(t, []byte(`{"input":{"bytes_base64":"e30="},"operation":"CANONICALIZE_JSON"}`))
	requireAtomicRunnerFailure(t, stdout, stderr, runErr, contextErr)
}

func TestParityFramingRejectsExpandedSemanticResultAtomically(t *testing.T) {
	var raw strings.Builder
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	for range 1023 {
		raw.WriteString("x: ")
		raw.WriteString(strings.Repeat("a", 42))
		raw.WriteString("\r\n")
	}
	raw.WriteString("content-length: 0\r\n\r\n")
	requestBytes := parityRequest(ParseHTTPResponse, httpParseInput([]byte(raw.String()), 1, 1<<20, 1024, 1024))
	if len(requestBytes)+1 > MaxFrameBytes {
		t.Fatalf("expanding witness request frame = %d; cap = %d", len(requestBytes)+1, MaxFrameBytes)
	}
	request, err := ParseRequest(requestBytes)
	if err != nil {
		t.Fatal(err)
	}
	result := Evaluate(request)
	if !bytes.HasPrefix(result.CanonicalBytes(), []byte(`{"status":"OK","value":`)) {
		t.Fatalf("semantic result is not OK: %.200s", result.CanonicalBytes())
	}
	if len(result.CanonicalBytes()) <= MaxFrameBodyBytes {
		t.Fatalf("expanding witness result = %d; want > %d", len(result.CanonicalBytes()), MaxFrameBodyBytes)
	}
	if frame, err := result.FrameBytes(); err == nil || frame != nil {
		t.Fatalf("oversized Go frame = %q, %v; want nil, error", frame, err)
	}

	stdout, stderr, runErr, contextErr := runNodeParityRaw(t, append(append([]byte(nil), requestBytes...), '\n'))
	requireAtomicRunnerFailure(t, stdout, stderr, runErr, contextErr)
}
