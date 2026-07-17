package parity

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNodeParityRunnerRejectsInvalidFramesAtomically(t *testing.T) {
	request := parityRequest(CanonicalizeJSON, base64Input([]byte(`{}`)))
	frame := append(append([]byte(nil), request...), '\n')
	boundary := parityBoundaryRequest(t, MaxFrameBodyBytes)
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty stream"},
		{name: "blank frame", input: []byte{'\n'}},
		{name: "BOM", input: append(append([]byte{0xef, 0xbb, 0xbf}, request...), '\n')},
		{name: "CRLF", input: append(append([]byte(nil), request...), '\r', '\n')},
		{name: "embedded CR", input: append(append(append([]byte(nil), request[:10]...), '\r'), append(request[10:], '\n')...)},
		{name: "embedded LF", input: append(append(append([]byte(nil), request[:10]...), '\n'), append(request[10:], '\n')...)},
		{name: "two frames", input: append(append([]byte(nil), frame...), frame...)},
		{name: "noncanonical body", input: append(append([]byte{' '}, request...), '\n')},
		{name: "invalid JSON", input: []byte{'{', '\n'}},
		{name: "65,537 byte frame", input: append(parityBoundaryRequest(t, MaxFrameBodyBytes+1), '\n')},
		{name: "cap-sized missing LF", input: append(append([]byte(nil), boundary...), 'x')},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, runErr, contextErr := runNodeParityRaw(t, test.input)
			requireAtomicRunnerFailure(t, stdout, stderr, runErr, contextErr)
		})
	}
}

func TestNodeParityRunnerAcceptsExactWholeWireCap(t *testing.T) {
	body := parityBoundaryRequest(t, MaxFrameBodyBytes)
	if len(body) != MaxFrameBodyBytes {
		t.Fatalf("boundary request body = %d bytes; want %d", len(body), MaxFrameBodyBytes)
	}
	request, err := ParseRequest(body)
	if err != nil {
		t.Fatalf("Go boundary request was rejected: %v", err)
	}
	want, err := Evaluate(request).FrameBytes()
	if err != nil {
		t.Fatal(err)
	}
	wire := append(append([]byte(nil), body...), '\n')
	if len(wire) != MaxFrameBytes {
		t.Fatalf("boundary request wire = %d bytes; want %d", len(wire), MaxFrameBytes)
	}
	stdout, stderr, runErr, contextErr := runNodeParityRaw(t, wire)
	if contextErr != nil || runErr != nil || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), want) {
		t.Fatalf("boundary runner = context=%v err=%v stdout=%q stderr=%q; want %q", contextErr, runErr, stdout.Bytes(), stderr.Bytes(), want)
	}
}

func TestResultFrameBytesExactBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{name: "empty", size: 0, wantErr: true},
		{name: "maximum body", size: MaxFrameBodyBytes},
		{name: "one over maximum", size: MaxFrameBodyBytes + 1, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := bytes.Repeat([]byte{'x'}, test.size)
			frame, err := (Result{canonical: body}).FrameBytes()
			if test.wantErr {
				if !errors.Is(err, errFrameBodySize) || frame != nil {
					t.Fatalf("FrameBytes(%d) = %q, %v; want nil, errFrameBodySize", test.size, frame, err)
				}
				return
			}
			if err != nil || len(frame) != MaxFrameBytes || frame[len(frame)-1] != '\n' || !bytes.Equal(frame[:len(frame)-1], body) {
				t.Fatalf("FrameBytes(%d) = %d bytes, %v; want exact body plus LF", test.size, len(frame), err)
			}
			frame[0] = 'y'
			if body[0] != 'x' {
				t.Fatal("FrameBytes mutated its semantic source")
			}
		})
	}
}

func TestParityResponseFramingExactBodyBoundary(t *testing.T) {
	atCapRequest, atCapResult := exactExpandedHTTPResult(t, MaxFrameBodyBytes)
	atCapFrame, err := atCapResult.FrameBytes()
	if err != nil || len(atCapFrame) != MaxFrameBytes {
		t.Fatalf("exact-cap Go result frame = %d bytes, %v; want %d", len(atCapFrame), err, MaxFrameBytes)
	}
	stdout, stderr, runErr, contextErr := runNodeParityRaw(t, append(append([]byte(nil), atCapRequest...), '\n'))
	if contextErr != nil || runErr != nil || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), atCapFrame) {
		t.Fatalf("exact-cap Node result = context=%v err=%v stdout-bytes=%d stderr=%q", contextErr, runErr, stdout.Len(), stderr.Bytes())
	}

	overRequest, overResult := exactExpandedHTTPResult(t, MaxFrameBodyBytes+1)
	if frame, err := overResult.FrameBytes(); !errors.Is(err, errFrameBodySize) || frame != nil {
		t.Fatalf("over-cap Go result = %q, %v; want nil, errFrameBodySize", frame, err)
	}
	stdout, stderr, runErr, contextErr = runNodeParityRaw(t, append(append([]byte(nil), overRequest...), '\n'))
	requireAtomicRunnerFailure(t, stdout, stderr, runErr, contextErr)
}

func parityBoundaryRequest(t testing.TB, size int) []byte {
	t.Helper()
	prefix := []byte(`{"input":{"padding":"`)
	suffix := []byte(`"},"operation":"UNKNOWN"}`)
	padding := size - len(prefix) - len(suffix)
	if padding < 0 {
		t.Fatalf("request size %d is smaller than the boundary request envelope", size)
	}
	body := make([]byte, 0, size)
	body = append(body, prefix...)
	body = append(body, bytes.Repeat([]byte{'a'}, padding)...)
	body = append(body, suffix...)
	return body
}

func exactExpandedHTTPResult(t testing.TB, target int) ([]byte, Result) {
	t.Helper()
	const repeatedHeaders = 1023
	const baseValueBytes = 39
	build := func(extra int) []byte {
		var raw strings.Builder
		raw.WriteString("HTTP/1.1 200 OK\r\n")
		for index := 0; index < repeatedHeaders; index++ {
			raw.WriteString("x: ")
			raw.WriteString(strings.Repeat("a", baseValueBytes))
			if index == repeatedHeaders-1 {
				raw.WriteString(strings.Repeat("b", extra))
			}
			raw.WriteString("\r\n")
		}
		raw.WriteString("content-length: 0\r\n\r\n")
		return parityRequest(ParseHTTPResponse, httpParseInput([]byte(raw.String()), 1, 1<<20, 1024, 1024))
	}
	evaluate := func(requestBytes []byte) Result {
		request, err := ParseRequest(requestBytes)
		if err != nil {
			t.Fatalf("expanding witness request was rejected: %v", err)
		}
		result := Evaluate(request)
		if !bytes.HasPrefix(result.CanonicalBytes(), []byte(`{"status":"OK","value":`)) {
			t.Fatalf("expanding witness semantic result is not OK: %.200s", result.CanonicalBytes())
		}
		return result
	}
	baseRequest := build(0)
	baseResult := evaluate(baseRequest)
	extra := target - len(baseResult.CanonicalBytes())
	if extra < 0 {
		t.Fatalf("expanding witness base result = %d; exceeds target %d", len(baseResult.CanonicalBytes()), target)
	}
	requestBytes := build(extra)
	if len(requestBytes) > MaxFrameBodyBytes {
		t.Fatalf("expanding witness request = %d; exceeds request body cap %d", len(requestBytes), MaxFrameBodyBytes)
	}
	result := evaluate(requestBytes)
	if got := len(result.CanonicalBytes()); got != target {
		t.Fatalf("expanding witness result = %d; want exactly %d (base=%d extra=%d request=%d)", got, target, len(baseResult.CanonicalBytes()), extra, len(requestBytes))
	}
	if len(requestBytes)+1 > MaxFrameBytes {
		t.Fatalf("expanding witness wire = %d; exceeds cap %d", len(requestBytes)+1, MaxFrameBytes)
	}
	return requestBytes, result
}

func runNodeParityRaw(t testing.TB, input []byte) (bytes.Buffer, bytes.Buffer, error, error) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, "runner.mjs")
	command.Env = []string{"LANG=C", "LC_ALL=C", "NO_COLOR=1", "TZ=UTC"}
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	return stdout, stderr, runErr, ctx.Err()
}

func requireAtomicRunnerFailure(t testing.TB, stdout, stderr bytes.Buffer, runErr, contextErr error) {
	t.Helper()
	var exitError *exec.ExitError
	if contextErr != nil || !errors.As(runErr, &exitError) || exitError.ExitCode() != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("runner failure = context=%v err=%v stdout=%q stderr=%q; want exit 1 and empty streams", contextErr, runErr, stdout.Bytes(), stderr.Bytes())
	}
}
