//go:build darwin

package world

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func httpTestDigest(t *testing.T, kind, value string) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes(kind, []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func legacyHTTPReadiness(t *testing.T) httpmodel.HTTPReadinessContract {
	t.Helper()
	contract, err := httpmodel.NewHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func runReadinessBytes(t *testing.T, payload []byte, closeWriter bool) (httpReadinessRead, domain.ControlReason, string) {
	return runReadinessContract(t, payload, closeWriter, legacyHTTPReadiness(t))
}

func runReadinessContract(
	t *testing.T,
	payload []byte,
	closeWriter bool,
	contract httpmodel.HTTPReadinessContract,
) (httpReadinessRead, domain.ControlReason, string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	overflow := make(chan struct{}, 2)
	stdout := newCappedCapture(1024, overflow)
	stderr := newCappedCapture(1024, overflow)
	waitC := make(chan waitResult, 1)
	go func() {
		_, _ = writer.Write(payload)
		if closeWriter {
			_ = writer.Close()
		}
	}()
	result, _, control, diagnostic := awaitExactHTTPReadiness(
		context.Background(), reader, 10*time.Millisecond, stdout, stderr, overflow, waitC, contract,
	)
	_ = writer.Close()
	return result, control, diagnostic
}

func TestHTTPPortableReadinessBindsExactChildReportedPortFrame(t *testing.T) {
	contract, err := httpmodel.NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	frame, err := httpmodel.NewHTTPReadyPortFrame(43127)
	if err != nil {
		t.Fatal(err)
	}
	result, control, diagnostic := runReadinessContract(t, frame.CanonicalBytes(), true, contract)
	if control != "" || diagnostic != "" || !result.eof || !bytes.Equal(result.bytes, frame.CanonicalBytes()) {
		t.Fatalf("portable readiness frame was not accepted exactly: result=%+v control=%s diagnostic=%s", result, control, diagnostic)
	}
	receipt, err := newHTTPReadinessReceipt(
		httpTestDigest(t, "world", "portable"), httpTestDigest(t, "attempt", "portable"), httpTestDigest(t, "binding", "portable"),
		httpReadinessPhysical{
			protocol: httpmodel.PortableReadinessProtocolV1, listenerFD: 0, readinessFD: httpPortableReadinessChildFD,
			endpoint: "127.0.0.1:43127", port: 43127, frameBytes: frame.CanonicalBytes(),
			bytesObserved: int64(len(frame.CanonicalBytes())), observedByte: frame.CanonicalBytes()[0], eofObserved: true, accepted: true,
		},
	)
	if err != nil || !receipt.Valid() || receipt.Authority() != httpPortableReadinessAuthority ||
		receipt.ListenerFDPresent() || receipt.ListenerFD() != 0 || receipt.ReadinessFD() != httpPortableReadinessChildFD ||
		receipt.Protocol() != httpmodel.PortableReadinessProtocolV1 || receipt.Port() != 43127 ||
		!bytes.Equal(receipt.FrameBytes(), frame.CanonicalBytes()) || !receipt.FrameDigest().Valid() {
		t.Fatalf("portable readiness receipt did not bind the child frame: receipt=%+v err=%v", receipt, err)
	}
	const portableReceiptDigest = "sha256:4d0ad3b13e6712d6b8f9498a043b364ed10fd43f3a120a12e071ea4377d26d21"
	const portableReceiptBytes = `{"attempt_artifact_digest":"sha256:06b90f70849baa42d83c67d4da9d0d6bc66c926a3b8e4f3e826f28b92fc151fc","authority":"P07B_CHILD_BIND_PIPE_FRAME_READINESS_RECEIPT_V1","child_reported_port":43127,"diagnostic_code":"","http_execution_binding_digest":"sha256:47a39c434570bddb2a899a9e1ad198556c3f3468d24e757be830076f93cd8689","inherited_listener_present":false,"inherited_readiness_fd":3,"kind":"HTTPReadinessReceipt","literal_loopback_endpoint":"127.0.0.1:43127","readiness_accepted":true,"readiness_bytes_observed":28,"readiness_eof_observed":true,"readiness_frame_base64":"Q09VTlRFUlNIQVBFX1JFQURZX1YxIDQzMTI3Cg==","readiness_frame_digest":"sha256:d2831697f4969c339de4421ca8381079418deed2a197d4ade40507c6be179351","readiness_protocol":"ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1","schema_version":"countershape/v1","world_instance_digest":"sha256:855b9092a3bfc51b4a48e818e37b07bf89ef451a7df360c925c6512dee945199"}`
	if receipt.Digest().String() != portableReceiptDigest || string(receipt.CanonicalBytes()) != portableReceiptBytes {
		t.Fatalf("portable readiness receipt golden changed:\ndigest=%s\nbytes=%s", receipt.Digest(), receipt.CanonicalBytes())
	}
	frameCopy := receipt.FrameBytes()
	frameCopy[0] ^= 0xff
	if bytes.Equal(frameCopy, receipt.FrameBytes()) {
		t.Fatal("portable readiness frame getter returned shared mutable bytes")
	}
	tamperedAuthority := receipt
	tamperedAuthority.authority = "FORGED_PORT_OWNERSHIP_AUTHORITY"
	if tamperedAuthority.Valid() {
		t.Fatal("readiness receipt accepted a tampered cached authority")
	}
	tamperedFrame := receipt
	tamperedFrame.frameBytes = append([]byte(nil), receipt.frameBytes...)
	tamperedFrame.frameBytes[0] ^= 0xff
	if tamperedFrame.Valid() {
		t.Fatal("readiness receipt accepted tampered cached frame bytes")
	}
	basePhysical := httpReadinessPhysical{
		protocol: httpmodel.PortableReadinessProtocolV1, listenerFD: 0, readinessFD: httpPortableReadinessChildFD,
		endpoint: "127.0.0.1:43127", port: 43127, frameBytes: frame.CanonicalBytes(),
		bytesObserved: int64(len(frame.CanonicalBytes())), observedByte: frame.CanonicalBytes()[0], eofObserved: true, accepted: true,
	}
	for _, hostile := range []struct {
		name   string
		mutate func(*httpReadinessPhysical)
	}{
		{name: "listener-present", mutate: func(value *httpReadinessPhysical) { value.listenerFD = 1 }},
		{name: "readiness-fd", mutate: func(value *httpReadinessPhysical) { value.readinessFD = 4 }},
		{name: "endpoint", mutate: func(value *httpReadinessPhysical) { value.endpoint = "127.0.0.1:43128" }},
		{name: "port", mutate: func(value *httpReadinessPhysical) { value.port = 43128 }},
		{name: "byte-count", mutate: func(value *httpReadinessPhysical) { value.bytesObserved-- }},
		{name: "missing-eof", mutate: func(value *httpReadinessPhysical) { value.eofObserved = false }},
		{name: "false-rejection", mutate: func(value *httpReadinessPhysical) { value.accepted = false }},
		{name: "overflow", mutate: func(value *httpReadinessPhysical) {
			value.frameBytes = append(bytes.Repeat([]byte{'X'}, httpmodel.PortableReadinessFrameMax+1), '\n')
			value.bytesObserved = int64(len(value.frameBytes))
			value.observedByte = value.frameBytes[0]
		}},
	} {
		physical := basePhysical
		physical.frameBytes = append([]byte(nil), basePhysical.frameBytes...)
		hostile.mutate(&physical)
		if _, err := newHTTPReadinessReceipt(
			httpTestDigest(t, "world", hostile.name), httpTestDigest(t, "attempt", hostile.name), httpTestDigest(t, "binding", hostile.name), physical,
		); err == nil {
			t.Fatalf("receipt-rejects-%s: hostile portable receipt input was accepted", hostile.name)
		}
	}
	if _, err := newHTTPReadinessReceipt(
		httpTestDigest(t, "world", "empty-byte"), httpTestDigest(t, "attempt", "empty-byte"), httpTestDigest(t, "binding", "empty-byte"),
		httpReadinessPhysical{
			protocol: httpmodel.PortableReadinessProtocolV1, listenerFD: 0, readinessFD: httpPortableReadinessChildFD,
			observedByte: 'X', eofObserved: true, accepted: false, diagnosticCode: "HTTP_READINESS_EMPTY",
		},
	); err == nil {
		t.Fatal("empty portable readiness frame retained a fabricated first byte")
	}
	for _, invalid := range []struct {
		bytes   []byte
		wantEOF bool
	}{
		{bytes: []byte("COUNTERSHAPE_READY_V1 043127\n"), wantEOF: true},
		{bytes: []byte("COUNTERSHAPE_READY_V1 43127\r\n"), wantEOF: true},
		// The bounded reader returns at max+1 bytes without waiting for EOF;
		// that is the positive overflow witness and prevents an unbounded child
		// from turning readiness rejection into memory growth.
		{bytes: []byte("COUNTERSHAPE_READY_V1 43127\nextra"), wantEOF: false},
	} {
		observed, refusal, _ := runReadinessContract(t, invalid.bytes, true, contract)
		if refusal != domain.ControlReadinessError || observed.eof != invalid.wantEOF ||
			!bytes.Equal(observed.bytes, invalid.bytes) {
			t.Fatalf("portable readiness admitted or misclassified malformed bytes %q: result=%+v control=%s", invalid.bytes, observed, refusal)
		}
	}
	environment, err := appendHTTPStartEnvironment([]string{"LANG=C"}, httpmodel.HTTPPortableStartAuthorityV1, 0)
	if err != nil || !slices.Equal(environment, []string{"COUNTERSHAPE_HTTP_READINESS_FD=3", "LANG=C"}) {
		t.Fatalf("portable child received listener/port authority: %q %v", environment, err)
	}
}

func TestHTTPReadinessUsesInheritedPipeWithoutHTTPWarmup(t *testing.T) {
	result, control, diagnostic := runReadinessBytes(t, []byte{0x01}, true)
	if control != "" || diagnostic != "" || !result.eof || !bytes.Equal(result.bytes, []byte{0x01}) {
		t.Fatalf("dedicated readiness pipe was not accepted exactly: result=%+v control=%s diagnostic=%s", result, control, diagnostic)
	}
	receipt, err := newHTTPReadinessReceipt(
		httpTestDigest(t, "world", "a"), httpTestDigest(t, "attempt", "a"), httpTestDigest(t, "binding", "a"),
		httpReadinessPhysical{listenerFD: 3, readinessFD: 4, endpoint: "127.0.0.1:43127", port: 43127,
			bytesObserved: 1, observedByte: 0x01, eofObserved: true, accepted: true},
	)
	if err != nil || !receipt.Valid() || !receipt.ListenerFDPresent() || receipt.ListenerFD() != 3 || receipt.ReadinessFD() != 4 {
		t.Fatalf("readiness receipt did not bind inherited listener/readiness descriptors: receipt=%+v err=%v", receipt, err)
	}
	const legacyReceiptDigest = "sha256:6c08fbecf7a99e61a811037a8a4df90ac635a2efb49e5a629613b1c860363d90"
	const legacyReceiptBytes = `{"allocated_port":43127,"attempt_artifact_digest":"sha256:6e03e4db9f18f0e80f8e03aec6100be9a2d201869c9731e0101dc9e690863759","authority":"U4_INHERITED_FD_READINESS_RECEIPT_V1","diagnostic_code":"","http_execution_binding_digest":"sha256:94d02fbd8392647fd7d5c4e3fb24632f602f2ca1ca0aafd580dddac9fa4f7036","inherited_listener_fd":3,"inherited_readiness_fd":4,"kind":"HTTPReadinessReceipt","literal_loopback_endpoint":"127.0.0.1:43127","readiness_accepted":true,"readiness_bytes_observed":1,"readiness_eof_observed":true,"readiness_first_byte":1,"readiness_protocol":"ONE_BYTE_0X01_THEN_EOF_V1","schema_version":"countershape/v1","world_instance_digest":"sha256:2e7a5276a199c26bab019774bd1daa4e71d2b9afc20c98799d5dbd487bb6fa4d"}`
	if receipt.Digest().String() != legacyReceiptDigest || string(receipt.CanonicalBytes()) != legacyReceiptBytes {
		t.Fatalf("legacy readiness receipt golden changed:\ndigest=%s\nbytes=%s", receipt.Digest(), receipt.CanonicalBytes())
	}
	if len(receipt.FrameBytes()) != 0 || receipt.FrameDigest().Valid() {
		t.Fatal("legacy receipt exposed portable-only frame evidence")
	}
	withFrame := httpReadinessPhysical{listenerFD: 3, readinessFD: 4, endpoint: "127.0.0.1:43127", port: 43127,
		bytesObserved: 1, observedByte: 0x01, frameBytes: []byte{0x01}, eofObserved: true, accepted: true}
	if _, err := newHTTPReadinessReceipt(
		httpTestDigest(t, "world", "legacy-frame"), httpTestDigest(t, "attempt", "legacy-frame"), httpTestDigest(t, "binding", "legacy-frame"), withFrame,
	); err == nil {
		t.Fatal("legacy readiness accepted unbound portable frame evidence")
	}
	tamperedAuthority := receipt
	tamperedAuthority.authority = "FORGED_LEGACY_AUTHORITY"
	if tamperedAuthority.Valid() {
		t.Fatal("legacy readiness accepted a tampered cached authority")
	}
	tamperedFrame := receipt
	tamperedFrame.frameBytes = []byte{0x01, 0x00}
	if tamperedFrame.Valid() {
		t.Fatal("legacy readiness accepted cached frame bytes absent from its historical identity")
	}
}

func TestOnlyRejectedPortableReadinessMayOmitExchange(t *testing.T) {
	base := httpReadinessPhysical{protocol: httpmodel.PortableReadinessProtocolV1, accepted: false}
	if !rejectedPortableReadinessOnly(httpmodel.HTTPPortableExecutionAuthorityV1, base) {
		t.Fatal("rejected portable readiness could not retain a readiness-only physical receipt")
	}
	accepted := base
	accepted.accepted = true
	legacyProtocol := base
	legacyProtocol.protocol = httpmodel.ReadinessProtocolV1
	if rejectedPortableReadinessOnly(httpmodel.HTTPPortableExecutionAuthorityV1, accepted) ||
		rejectedPortableReadinessOnly(httpmodel.HTTPPortableExecutionAuthorityV1, legacyProtocol) ||
		rejectedPortableReadinessOnly(httpmodel.HTTPExecutionAuthorityV1, base) {
		t.Fatal("accepted, cross-protocol, or legacy readiness escaped the missing-exchange invariant")
	}
}

func TestHTTPReadinessRequiresExactByteAndEOF(t *testing.T) {
	for _, test := range []struct {
		name        string
		payload     []byte
		closeWriter bool
		wantBytes   []byte
	}{
		{name: "empty-eof", payload: nil, closeWriter: true, wantBytes: nil},
		{name: "wrong-byte", payload: []byte{0x02}, closeWriter: true, wantBytes: []byte{0x02}},
		{name: "extra-byte", payload: []byte{0x01, 0x00}, closeWriter: true, wantBytes: []byte{0x01, 0x00}},
		{name: "missing-eof", payload: []byte{0x01}, closeWriter: false, wantBytes: []byte{0x01}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, control, _ := runReadinessBytes(t, test.payload, test.closeWriter)
			if control != domain.ControlReadinessError {
				t.Fatalf("malformed readiness became eligible: %s", control)
			}
			if !bytes.Equal(result.bytes, test.wantBytes) {
				t.Fatalf("malformed readiness evidence = %x, want %x", result.bytes, test.wantBytes)
			}
		})
	}
}

func TestHTTPReadinessRejectsWrongByteMutationGuard(t *testing.T) {
	result, control, diagnostic := runReadinessBytes(t, []byte{0x02}, true)
	if control != domain.ControlReadinessError {
		t.Fatalf("wrong readiness byte became eligible: result=%+v control=%s diagnostic=%s", result, control, diagnostic)
	}
	if !result.eof || !bytes.Equal(result.bytes, []byte{0x02}) {
		t.Fatalf("wrong readiness evidence was not retained exactly: result=%+v", result)
	}
}

func TestHTTPReadinessOwnerPriorityExitBeforeExactBytes(t *testing.T) {
	for iteration := 0; iteration < 50; iteration++ {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte{httpmodel.ReadinessSuccessByte}); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		waitC := make(chan waitResult, 1)
		waitC <- waitResult{}
		overflow := make(chan struct{}, 2)
		observed, waited, control, diagnostic := awaitExactHTTPReadiness(
			context.Background(), reader, time.Second,
			newCappedCapture(1024, overflow), newCappedCapture(1024, overflow), overflow, waitC, legacyHTTPReadiness(t),
		)
		_ = reader.Close()
		if waited == nil || control != domain.ControlReadinessError || diagnostic != "HTTP_PROCESS_EXITED_BEFORE_READINESS" {
			t.Fatalf("iteration %d classified simultaneous exit/readiness nondeterministically: bytes=%x waited=%+v control=%s diagnostic=%s",
				iteration, observed.bytes, waited, control, diagnostic)
		}
		if !observed.eof || !bytes.Equal(observed.bytes, []byte{httpmodel.ReadinessSuccessByte}) {
			t.Fatalf("iteration %d lost exact readiness evidence while exit retained precedence: %+v", iteration, observed)
		}
	}
}

func TestHTTPReadinessCancellationTimeoutAndEarlyExitControls(t *testing.T) {
	newCaptures := func() (*cappedCapture, *cappedCapture, chan struct{}) {
		overflow := make(chan struct{}, 2)
		return newCappedCapture(1024, overflow), newCappedCapture(1024, overflow), overflow
	}

	t.Run("cancellation", func(t *testing.T) {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		stdout, stderr, overflow := newCaptures()
		_, _, control, diagnostic := awaitExactHTTPReadiness(ctx, reader, time.Second, stdout, stderr, overflow, make(chan waitResult, 1), legacyHTTPReadiness(t))
		if control != domain.ControlCancelled || diagnostic != "HTTP_READINESS_CANCELLED" {
			t.Fatalf("cancelled readiness = %s %s", control, diagnostic)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()
		stdout, stderr, overflow := newCaptures()
		_, _, control, diagnostic := awaitExactHTTPReadiness(context.Background(), reader, time.Millisecond, stdout, stderr, overflow, make(chan waitResult, 1), legacyHTTPReadiness(t))
		if control != domain.ControlReadinessError || diagnostic != "HTTP_READINESS_TIMEOUT" {
			t.Fatalf("timed-out readiness = %s %s", control, diagnostic)
		}
	})

	t.Run("early-exit", func(t *testing.T) {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()
		waitC := make(chan waitResult, 1)
		waitC <- waitResult{}
		stdout, stderr, overflow := newCaptures()
		_, waited, control, diagnostic := awaitExactHTTPReadiness(context.Background(), reader, time.Second, stdout, stderr, overflow, waitC, legacyHTTPReadiness(t))
		if waited == nil || control != domain.ControlReadinessError || diagnostic != "HTTP_PROCESS_EXITED_BEFORE_READINESS" {
			t.Fatalf("early-exit readiness = waited=%+v %s %s", waited, control, diagnostic)
		}
	})
}

func TestHTTPServiceControlPrecedenceOutputBeforeCancellationAndTimeout(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	overflow := make(chan struct{}, 2)
	stdout := newCappedCapture(1, overflow)
	stderr := newCappedCapture(1, overflow)
	stdout.retain([]byte("overflow"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, control, diagnostic := awaitExactHTTPReadiness(
		ctx, reader, time.Nanosecond, stdout, stderr, overflow, make(chan waitResult, 1), legacyHTTPReadiness(t),
	)
	if control != domain.ControlOutputLimit || diagnostic != "HTTP_PROCESS_OUTPUT_LIMIT_BEFORE_READINESS" {
		t.Fatalf("owner priority did not preserve output limit: %s %s", control, diagnostic)
	}
}

func TestHTTPDescriptorEnvironmentDoesNotConsultAmbientProxyPolicy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:2")
	t.Setenv("NO_PROXY", "*")
	environment, err := appendHTTPDescriptorEnvironment([]string{"LANG=C"}, 43127)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY":
			t.Fatalf("ambient proxy policy escaped into the HTTP service environment: %q", entry)
		}
	}
}

func TestHTTPDescriptorEnvironmentRejectsHostilePlanCollision(t *testing.T) {
	for _, owned := range []string{
		httpListenerFDEnvironment,
		httpReadinessFDEnvironment,
		httpListenerPortEnvironment,
	} {
		t.Run(owned, func(t *testing.T) {
			if _, err := appendHTTPDescriptorEnvironment([]string{"LANG=C", owned + "=999"}, 43127); err == nil {
				t.Fatalf("runner-owned descriptor %s admitted a duplicate candidate-visible value", owned)
			}
			roots := Roots{
				home: "/owned/home", temporary: "/owned/tmp", xdgConfig: "/owned/config",
				xdgCache: "/owned/cache", xdgData: "/owned/data", xdgState: "/owned/xdg-state",
				state: "/owned/state", evidence: "/owned/evidence", fixture: "/owned/fixture",
			}
			if _, err := buildHTTPEnvironment(
				[]domain.EnvironmentEntry{{Name: owned, Value: "999"}}, roots, "attempt:hostile",
				httpTestDigest(t, "stimulus", "hostile"), 0, 2,
			); err == nil {
				t.Fatalf("plan collision with runner-owned descriptor %s was not rejected before spawn", owned)
			}
		})
	}
	if _, err := appendHTTPDescriptorEnvironment([]string{"LANG=C", "LANG=en_US"}, 43127); err == nil {
		t.Fatal("duplicate environment names reached exec.Cmd")
	}
}

func TestHTTPProductAllocationHasNoSharedRootAndAllocatesFreshAttempts(t *testing.T) {
	typeOfRequest := reflect.TypeOf(HTTPRequest{})
	for index := 0; index < typeOfRequest.NumField(); index++ {
		name := strings.ToLower(typeOfRequest.Field(index).Name)
		if strings.Contains(name, "shared") || strings.Contains(name, "port") || strings.Contains(name, "proxy") || strings.Contains(name, "url") {
			t.Fatalf("product HTTP request exposes forbidden execution knob: %s", typeOfRequest.Field(index).Name)
		}
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := markerInput{SchemaVersion: domain.SchemaVersion, Kind: "AttemptMarker", Purpose: "DISCOVERY", MarkerOrdering: "DURABLE_BEFORE_SPAWN"}
	left, _, _, _, err := allocateOwnedRoots(base, marker)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(left.attempt)
	right, _, _, _, err := allocateOwnedRoots(base, marker)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(right.attempt)
	if left.attempt == right.attempt || left.fixture == right.fixture || left.state == right.state {
		t.Fatal("two HTTP attempts reused an owned root")
	}
}

func TestHTTPExchangeReceiptEnforcesOneDirectProbeNoRedirectOrRetry(t *testing.T) {
	request := []byte("GET / HTTP/1.1\r\nhost: 127.0.0.1:43127\r\nconnection: close\r\n\r\n")
	physical := httpExchangePhysical{
		connectionAttempts: 1, requestWire: request, requestSemanticDigest: httpTestDigest(t, "request", "semantic"),
		requestRawSHA256: rawSHA256Hex(request), requestWritten: int64(len(request)), requestComplete: true,
		diagnosticCode: "HTTP_RESPONSE_NOT_ATTEMPTED",
	}
	receipt, err := newHTTPExchangeReceipt(
		httpTestDigest(t, "world", "b"), httpTestDigest(t, "attempt", "b"), httpTestDigest(t, "binding", "b"),
		"127.0.0.1:43127", physical,
	)
	if err != nil || !receipt.Valid() || receipt.ConnectionAttempts() != 1 || receipt.RedirectsFollowed() != 0 ||
		receipt.Retries() != 0 || receipt.Proxy() != "NONE" || receipt.DNS() != "NONE_LITERAL_IP" ||
		receipt.CookieJar() != "NONE" || receipt.Compression() != "NONE" {
		t.Fatalf("one direct exchange authority failed: receipt=%+v err=%v", receipt, err)
	}
	physical.connectionAttempts = 2
	if _, err := newHTTPExchangeReceipt(
		httpTestDigest(t, "world", "b"), httpTestDigest(t, "attempt", "b"), httpTestDigest(t, "binding", "b"),
		"127.0.0.1:43127", physical,
	); err == nil {
		t.Fatal("exchange receipt admitted a retry/redirect connection")
	}
}

func TestHTTPInvocationEvidenceBindsAttemptStimulusAndExactRequest(t *testing.T) {
	evidenceRoot := t.TempDir()
	attemptID := "attempt:http-exact"
	stimulusDigest := httpTestDigest(t, "stimulus", "exact")
	bindingDigest := httpTestDigest(t, "binding", "exact")
	request := []byte("GET /invoice HTTP/1.1\r\nhost: 127.0.0.1:43127\r\nconnection: close\r\n\r\n")
	payload := []byte(fmt.Sprintf(
		`{"attempt_id":%q,"invocation_count":1,"kind":"HTTPFixtureInvocationEvidence","request_byte_count":%d,"request_byte_sha256":%q,"schema_version":"countershape/v1","stimulus_digest":%q}`,
		attemptID, len(request), rawSHA256(request), stimulusDigest.String(),
	))
	if err := os.WriteFile(filepath.Join(evidenceRoot, httpInvocationFilename), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	worldDigest := httpTestDigest(t, "world", "exact")
	attemptDigest := httpTestDigest(t, "attempt", "exact")
	receipt, err := inspectHTTPInvocationEvidence(worldDigest, attemptDigest, bindingDigest, stimulusDigest, evidenceRoot, attemptID, request)
	if err != nil || !receipt.Valid() || !receipt.Validated() || !receipt.RequestByteCountValidated() || !receipt.RequestDigestValidated() {
		t.Fatalf("exact invocation evidence was not validated: receipt=%+v err=%v", receipt, err)
	}
	changed := append([]byte(nil), request...)
	changed[len(changed)-5] ^= 0x01
	mismatch, err := inspectHTTPInvocationEvidence(worldDigest, attemptDigest, bindingDigest, stimulusDigest, evidenceRoot, attemptID, changed)
	if err != nil || mismatch.Validated() {
		t.Fatalf("substituted request was accepted: receipt=%+v err=%v", mismatch, err)
	}
}

func TestHTTPSuccessCannotEraseTeardownOrOrphanControls(t *testing.T) {
	attempt, err := domain.NewAttempt("attempt:http", httpTestDigest(t, "attempt", "teardown"), domain.AttemptDiscovery)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady,
		domain.AttemptProbing, domain.AttemptCapturing, domain.AttemptTearingDown,
	} {
		attempt, err = attempt.Advance(state)
		if err != nil {
			t.Fatal(err)
		}
	}
	attempt, err = attempt.RecordTeardownControl(domain.ControlTeardownError)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = attempt.RecordTeardownControl(domain.ControlOrphanRisk)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = attempt.Advance(domain.AttemptFinalized)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if !finalized.HasControls() || len(finalized.TeardownControls()) != 2 {
		t.Fatal("successful capture erased teardown/orphan ineligibility")
	}
}

func TestHTTPNaturalNonzeroServiceExitCannotRemainEligible(t *testing.T) {
	for _, test := range []struct {
		name        string
		exitCode    int
		exitSignal  string
		termSent    bool
		killSent    bool
		wantControl bool
	}{
		{name: "clean-natural-exit", exitCode: 0},
		{name: "natural-exit-64", exitCode: 64, wantControl: true},
		{name: "natural-signal", exitCode: -1, exitSignal: "terminated", wantControl: true},
		{name: "owner-term", exitCode: -1, exitSignal: "terminated", termSent: true},
		{name: "owner-kill", exitCode: -1, exitSignal: "killed", killSent: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			process := physicalProcessResult{
				directChildWaited: true,
				exitCode:          test.exitCode,
				exitSignal:        test.exitSignal,
				termSent:          test.termSent,
				killSent:          test.killSent,
			}
			applyHTTPNaturalExitControl(&process)
			if got := process.primary != ""; got != test.wantControl {
				t.Fatalf("control present=%t, want %t: %+v", got, test.wantControl, process)
			}
			if test.wantControl && (process.primary != domain.ControlProbeTransportError || process.diagnosticCode != "HTTP_SERVICE_NONZERO_EXIT") {
				t.Fatalf("natural service failure lost exact control: %+v", process)
			}
		})
	}
}

func TestHTTPExchangeEstablishedCancellationInterruptsRead(t *testing.T) {
	requestRead := make(chan []byte, 1)
	release := make(chan struct{})
	endpoint, peerDone := startHTTPTestPeer(t, func(connection net.Conn) error {
		request, err := io.ReadAll(connection)
		if err != nil {
			return err
		}
		requestRead <- request
		<-release
		return nil
	})
	request := []byte("GET / HTTP/1.1\r\nhost: " + endpoint + "\r\nconnection: close\r\n\r\n")
	ctx, cancel := context.WithCancel(context.Background())
	resultC := make(chan httpExchangeResult, 1)
	go func() { resultC <- performOneHTTPExchange(ctx, endpoint, request, 4096, 5*time.Second) }()
	received := awaitHTTPTestRequest(t, requestRead)
	if !bytes.Equal(received, request) {
		t.Fatal("established peer did not receive the exact request before cancellation")
	}
	started := time.Now()
	cancel()
	result := awaitHTTPExchangeResult(t, resultC, time.Second)
	if time.Since(started) > 500*time.Millisecond {
		t.Fatal("established cancellation did not promptly interrupt the read")
	}
	if result.control != domain.ControlCancelled || result.diagnostic != "HTTP_PROBE_CANCELLED" || !result.complete || result.written != int64(len(request)) {
		t.Fatalf("established cancellation = %+v", result)
	}
	close(release)
	awaitHTTPTestPeer(t, peerDone)
}

func TestHTTPExchangeParentDeadlineOverridesLongProbeBudget(t *testing.T) {
	requestRead := make(chan []byte, 1)
	release := make(chan struct{})
	endpoint, peerDone := startHTTPTestPeer(t, func(connection net.Conn) error {
		request, err := io.ReadAll(connection)
		if err != nil {
			return err
		}
		requestRead <- request
		<-release
		return nil
	})
	request := []byte("GET / HTTP/1.1\r\nhost: " + endpoint + "\r\nconnection: close\r\n\r\n")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	resultC := make(chan httpExchangeResult, 1)
	started := time.Now()
	go func() { resultC <- performOneHTTPExchange(ctx, endpoint, request, 4096, 5*time.Second) }()
	_ = awaitHTTPTestRequest(t, requestRead)
	result := awaitHTTPExchangeResult(t, resultC, time.Second)
	if time.Since(started) > 500*time.Millisecond {
		t.Fatal("shorter parent deadline did not bound the established exchange")
	}
	if result.control != domain.ControlTimeout || result.diagnostic != "HTTP_PROBE_TIMEOUT" {
		t.Fatalf("parent-deadline exchange = %+v", result)
	}
	close(release)
	awaitHTTPTestPeer(t, peerDone)
}

func TestHTTPExchangeConnectionRefusalIsTransportControl(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	result := performOneHTTPExchange(context.Background(), endpoint, []byte("GET / HTTP/1.1\r\nhost: "+endpoint+"\r\nconnection: close\r\n\r\n"), 4096, time.Second)
	if result.control != domain.ControlProbeTransportError || result.diagnostic != "HTTP_CONNECT_FAILED" || result.complete || result.written != 0 {
		t.Fatalf("connection refusal became response behavior: %+v", result)
	}
}

func TestHTTPExchangeIgnoresAmbientProxyRoutingMutationGuard(t *testing.T) {
	proxy, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	if err := proxy.SetDeadline(time.Now().Add(750 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	proxyAccepted := make(chan error, 1)
	go func() {
		connection, acceptErr := proxy.AcceptTCP()
		if acceptErr == nil {
			_ = connection.Close()
		}
		proxyAccepted <- acceptErr
	}()

	response := []byte("HTTP/1.1 200 OK\r\ncontent-length: 2\r\n\r\nok")
	target, targetDone := startHTTPTestPeer(t, func(connection net.Conn) error {
		if _, readErr := io.ReadAll(connection); readErr != nil {
			return readErr
		}
		return writeHTTPTestPeerAll(connection, response)
	})
	proxyURL := "http://" + proxy.Addr().String()
	t.Setenv("HTTP_PROXY", proxyURL)
	t.Setenv("HTTPS_PROXY", proxyURL)
	t.Setenv("ALL_PROXY", proxyURL)
	t.Setenv("NO_PROXY", "")
	request := []byte("GET /direct HTTP/1.1\r\nhost: " + target + "\r\nconnection: close\r\n\r\n")
	result := performOneHTTPExchange(context.Background(), target, request, 4096, time.Second)
	awaitHTTPTestPeer(t, targetDone)
	if result.control != "" || !result.complete || !bytes.Equal(result.wire, response) {
		t.Fatalf("ambient proxy variables changed the literal direct exchange: %+v", result)
	}
	select {
	case acceptErr := <-proxyAccepted:
		if acceptErr == nil {
			t.Fatal("literal loopback exchange was routed through the ambient proxy")
		}
		var networkError net.Error
		if !errors.As(acceptErr, &networkError) || !networkError.Timeout() {
			t.Fatalf("proxy sentinel ended without an expected no-connection timeout: %v", acceptErr)
		}
	case <-time.After(time.Second):
		t.Fatal("proxy sentinel did not resolve")
	}
}

func TestHTTPExchangeNeverFollowsRedirectLocationMutationGuard(t *testing.T) {
	redirectTarget, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = redirectTarget.Close() })
	if err := redirectTarget.SetDeadline(time.Now().Add(750 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	redirectAccepted := make(chan error, 1)
	go func() {
		connection, acceptErr := redirectTarget.AcceptTCP()
		if acceptErr == nil {
			_ = connection.Close()
		}
		redirectAccepted <- acceptErr
	}()

	response := []byte("HTTP/1.1 302 Found\r\nlocation: http://" + redirectTarget.Addr().String() + "/second\r\ncontent-length: 0\r\n\r\n")
	firstEndpoint, firstDone := startHTTPTestPeer(t, func(connection net.Conn) error {
		if _, readErr := io.ReadAll(connection); readErr != nil {
			return readErr
		}
		return writeHTTPTestPeerAll(connection, response)
	})
	request := []byte("GET /first HTTP/1.1\r\nhost: " + firstEndpoint + "\r\nconnection: close\r\n\r\n")
	result := performOneHTTPExchange(context.Background(), firstEndpoint, request, 4096, time.Second)
	awaitHTTPTestPeer(t, firstDone)
	if result.control != "" || !result.complete || !bytes.Equal(result.wire, response) {
		t.Fatalf("direct exchange did not retain the first complete redirect response: %+v", result)
	}
	policy, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 256, HeaderBytes: 1024, HeaderCount: 16, BodyBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := httpmodel.ParseResponse(result.wire, policy)
	if err != nil || parsed.Status() != 302 {
		t.Fatalf("first redirect response was not retained as application behavior: status=%d err=%v", parsed.Status(), err)
	}
	select {
	case acceptErr := <-redirectAccepted:
		if acceptErr == nil {
			t.Fatal("Location triggered a forbidden second connection")
		}
		var networkError net.Error
		if !errors.As(acceptErr, &networkError) || !networkError.Timeout() {
			t.Fatalf("redirect sentinel ended without an expected no-connection timeout: %v", acceptErr)
		}
	case <-time.After(time.Second):
		t.Fatal("redirect sentinel did not resolve")
	}
}

func TestHTTPExchangePartialResponseRemainsProtocolFailure(t *testing.T) {
	partial := []byte("HTTP/1.1 200 OK\r\ncontent-length: 4\r\n\r\nab")
	endpoint, peerDone := startHTTPTestPeer(t, func(connection net.Conn) error {
		if _, err := io.ReadAll(connection); err != nil {
			return err
		}
		return writeHTTPTestPeerAll(connection, partial)
	})
	request := []byte("GET / HTTP/1.1\r\nhost: " + endpoint + "\r\nconnection: close\r\n\r\n")
	result := performOneHTTPExchange(context.Background(), endpoint, request, 4096, time.Second)
	awaitHTTPTestPeer(t, peerDone)
	if result.control != "" || !bytes.Equal(result.wire, partial) {
		t.Fatalf("physical partial response was not retained for strict parsing: %+v", result)
	}
	policy, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 256, HeaderBytes: 1024, HeaderCount: 16, BodyBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, parseErr := httpmodel.ParseResponse(result.wire, policy)
	control, diagnostic := classifyHTTPResponseParseError(parseErr)
	if parseErr == nil || control != domain.ControlProbeTransportError || diagnostic != httpmodel.CodeResponseLength {
		t.Fatalf("partial response parse classification = err=%v control=%s diagnostic=%s", parseErr, control, diagnostic)
	}
}

func startHTTPTestPeer(t *testing.T, handler func(net.Conn) error) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		defer listener.Close()
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer connection.Close()
		done <- handler(connection)
	}()
	return listener.Addr().String(), done
}

func awaitHTTPTestRequest(t *testing.T, requestC <-chan []byte) []byte {
	t.Helper()
	select {
	case request := <-requestC:
		return request
	case <-time.After(time.Second):
		t.Fatal("HTTP test peer did not receive one request")
		return nil
	}
}

func awaitHTTPExchangeResult(t *testing.T, resultC <-chan httpExchangeResult, budget time.Duration) httpExchangeResult {
	t.Helper()
	select {
	case result := <-resultC:
		return result
	case <-time.After(budget):
		t.Fatal("HTTP exchange did not respect the test deadline")
		return httpExchangeResult{}
	}
}

func awaitHTTPTestPeer(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP test peer did not terminate")
	}
}

func writeHTTPTestPeerAll(connection net.Conn, payload []byte) error {
	for offset := 0; offset < len(payload); {
		count, err := connection.Write(payload[offset:])
		offset += count
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
