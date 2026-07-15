package model

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTTPWireRefusesProxyAndAmbientPolicyHeadersMutationGuard(t *testing.T) {
	for _, name := range []string{"proxy-authorization", "proxy-connection", "accept-encoding", "cookie"} {
		if _, err := NewRequestHeader(name, "ambient"); err == nil {
			t.Fatalf("policy-bearing header %q was accepted", name)
		}
	}
}

func TestHTTPWirePreservesOrderedQueryAndHeaderBytesMutationGuard(t *testing.T) {
	flag := mustQueryFlag(t, "flag")
	empty := mustQueryValue(t, "empty", "")
	dupA := mustQueryValue(t, "dup", "a")
	dupB := mustQueryValue(t, "dup", "b")
	first := mustHeader(t, "x-first", "one")
	second := mustHeader(t, "x-second", "two")
	third := mustHeader(t, "x-first", "three")
	stimulus, err := NewHTTPStimulus(HTTPStimulusConfig{
		Method:  MethodGET,
		Path:    "/ordered",
		Query:   []HTTPQueryEntry{flag, empty, dupA, dupB},
		Headers: []HTTPRequestHeader{first, second, third},
		Body:    AbsentBody(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := EncodeRequest(stimulus, 7777)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := wire.Target(), "/ordered?flag&empty=&dup=a&dup=b"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
	encoded := string(wire.Bytes())
	if !strings.HasPrefix(encoded, "GET /ordered?flag&empty=&dup=a&dup=b HTTP/1.1\r\n") {
		t.Fatalf("request line lost ordered query semantics: %q", encoded)
	}
	firstIndex := strings.Index(encoded, "x-first: one\r\n")
	secondIndex := strings.Index(encoded, "x-second: two\r\n")
	thirdIndex := strings.Index(encoded, "x-first: three\r\n")
	if firstIndex < 0 || secondIndex <= firstIndex || thirdIndex <= secondIndex {
		t.Fatalf("request headers were not emitted in declared duplicate-preserving order: %q", encoded)
	}
}

func TestHTTPWireDistinguishesAbsentAndPresentEmptyBodyMutationGuard(t *testing.T) {
	absentStimulus, err := NewHTTPStimulus(HTTPStimulusConfig{Method: MethodPOST, Path: "/body", Body: AbsentBody()})
	if err != nil {
		t.Fatal(err)
	}
	presentEmpty := mustPresentBody(t, nil)
	presentStimulus, err := NewHTTPStimulus(HTTPStimulusConfig{Method: MethodPOST, Path: "/body", Body: presentEmpty})
	if err != nil {
		t.Fatal(err)
	}
	absentWire, err := EncodeRequest(absentStimulus, 7777)
	if err != nil {
		t.Fatal(err)
	}
	presentWire, err := EncodeRequest(presentStimulus, 7777)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(absentWire.Bytes(), presentWire.Bytes()) || absentStimulus.Digest() == presentStimulus.Digest() {
		t.Fatal("absent and present-empty bodies collapsed to one identity")
	}
	if bytes.Contains(absentWire.Bytes(), []byte("content-length: 0\r\n")) {
		t.Fatal("absent body synthesized a present-empty content length")
	}
	if !bytes.Contains(presentWire.Bytes(), []byte("content-length: 0\r\n")) {
		t.Fatal("present-empty body did not emit its explicit content length")
	}
}

func TestHTTPStatus500RemainsCompleteApplicationResponseMutationGuard(t *testing.T) {
	policy := mustCapturePolicy(t, 1<<20)
	body := []byte(`{"kind":"error","metadata":{}}`)
	raw := []byte("HTTP/1.1 500 Internal Server Error\r\ncontent-type: application/json\r\ncontent-length: " + decimal(len(body)) + "\r\n\r\n" + string(body))
	response, err := ParseResponse(raw, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Valid() || response.Status() != 500 || !bytes.Equal(response.Body(), body) {
		t.Fatalf("500 response was not retained as a complete application response: status=%d", response.Status())
	}
}

func TestHTTPResponseParserRejectsContentLengthTransferEncodingAndOverflowMutationGuard(t *testing.T) {
	policy := mustCapturePolicy(t, 3)
	cases := map[string][]byte{
		"content-length mismatch": []byte("HTTP/1.1 200 OK\r\ncontent-length: 3\r\n\r\nab"),
		"transfer encoding":       []byte("HTTP/1.1 200 OK\r\ntransfer-encoding: chunked\r\n\r\n0\r\n\r\n"),
		"body overflow":           []byte("HTTP/1.1 200 OK\r\ncontent-length: 4\r\n\r\nfour"),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResponse(raw, policy); err == nil {
				t.Fatal("malformed or over-budget response was accepted")
			}
		})
	}
}

func TestHTTPResponseParserRejectsContentLengthMismatchMutationGuard(t *testing.T) {
	policy := mustCapturePolicy(t, 3)
	raw := []byte("HTTP/1.1 200 OK\r\ncontent-length: 3\r\n\r\nab")
	if _, err := ParseResponse(raw, policy); err == nil {
		t.Fatal("content-length mismatch was accepted")
	}
}

func TestHTTPResponseParserRefusesBadStatusAndHeadOverflowWithExactControls(t *testing.T) {
	standardPolicy := mustCapturePolicy(t, 32)
	statusLimitedPolicy, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{
		StatusLineBytes: 16, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	headerLimitedPolicy, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 16, HeaderCount: 32, BodyBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	headerCountLimitedPolicy, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 1, BodyBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		raw    []byte
		policy HTTPCapturePolicy
		code   string
	}{
		{
			name:   "bad-final-status-grammar",
			raw:    []byte("HTTP/1.0 200 OK\r\ncontent-length: 0\r\n\r\n"),
			policy: standardPolicy,
			code:   CodeResponseStatus,
		},
		{
			name:   "status-line-byte-limit",
			raw:    []byte("HTTP/1.1 200 Very Long Reason\r\ncontent-length: 0\r\n\r\n"),
			policy: statusLimitedPolicy,
			code:   CodeResponseStatusLimit,
		},
		{
			name:   "header-byte-limit",
			raw:    []byte("HTTP/1.1 200 OK\r\ncontent-length: 0\r\n\r\n"),
			policy: headerLimitedPolicy,
			code:   CodeResponseHeaderLimit,
		},
		{
			name:   "header-count-limit",
			raw:    []byte("HTTP/1.1 200 OK\r\nx-one: 1\r\ncontent-length: 0\r\n\r\n"),
			policy: headerCountLimitedPolicy,
			code:   CodeResponseHeaderLimit,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, parseErr := ParseResponse(test.raw, test.policy)
			if parseErr == nil {
				t.Fatalf("response outside %s was accepted", test.code)
			}
			code, ok := RefusalCodeOf(parseErr)
			if !ok || code != test.code {
				t.Fatalf("refusal = %q present=%t, want %q: %v", code, ok, test.code, parseErr)
			}
		})
	}
}

func mustQueryFlag(t *testing.T, name string) HTTPQueryEntry {
	t.Helper()
	entry, err := QueryFlag(name)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func mustQueryValue(t *testing.T, name, value string) HTTPQueryEntry {
	t.Helper()
	entry, err := QueryValue(name, value)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func mustHeader(t *testing.T, name, value string) HTTPRequestHeader {
	t.Helper()
	header, err := NewRequestHeader(name, value)
	if err != nil {
		t.Fatal(err)
	}
	return header
}

func mustPresentBody(t *testing.T, value []byte) HTTPBody {
	t.Helper()
	body, err := PresentBody(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func mustCapturePolicy(t *testing.T, bodyBytes int64) HTTPCapturePolicy {
	t.Helper()
	policy, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: bodyBytes})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
