package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	httpInvocationFilename         = "http-invocation.json"
	maxHTTPInvocationBytes         = int64(4096)
	httpReadinessReceiptAuthority  = "U4_INHERITED_FD_READINESS_RECEIPT_V1"
	httpExchangeReceiptAuthority   = "U4_DIRECT_TCP_EXACTLY_ONE_EXCHANGE_RECEIPT_V1"
	httpInvocationReceiptAuthority = "U4_FIXTURE_WRITTEN_EXACT_INVOCATION_RECEIPT_V1"
)

type HTTPReadinessReceipt struct {
	digest         domain.Digest
	canonicalBytes []byte
	worldDigest    domain.Digest
	attemptDigest  domain.Digest
	bindingDigest  domain.Digest
	endpoint       string
	port           int
	listenerFD     int
	readinessFD    int
	bytesObserved  int64
	observedByte   byte
	eofObserved    bool
	accepted       bool
	diagnosticCode string
}

type httpReadinessReceiptIdentity struct {
	SchemaVersion  string `json:"schema_version"`
	Kind           string `json:"kind"`
	Authority      string `json:"authority"`
	WorldDigest    string `json:"world_instance_digest"`
	AttemptDigest  string `json:"attempt_artifact_digest"`
	BindingDigest  string `json:"http_execution_binding_digest"`
	Endpoint       string `json:"literal_loopback_endpoint"`
	Port           int    `json:"allocated_port"`
	ListenerFD     int    `json:"inherited_listener_fd"`
	ReadinessFD    int    `json:"inherited_readiness_fd"`
	Protocol       string `json:"readiness_protocol"`
	BytesObserved  int64  `json:"readiness_bytes_observed"`
	ObservedByte   int    `json:"readiness_first_byte"`
	EOFObserved    bool   `json:"readiness_eof_observed"`
	Accepted       bool   `json:"readiness_accepted"`
	DiagnosticCode string `json:"diagnostic_code"`
}

func newHTTPReadinessReceipt(
	worldDigest, attemptDigest, bindingDigest domain.Digest,
	physical httpReadinessPhysical,
) (HTTPReadinessReceipt, error) {
	if !worldDigest.Valid() || !attemptDigest.Valid() || !bindingDigest.Valid() ||
		physical.port < 1 || physical.port > 65535 || physical.endpoint != "127.0.0.1:"+strconv.Itoa(physical.port) ||
		physical.listenerFD != httpListenerChildFD || physical.readinessFD != httpReadinessChildFD ||
		physical.bytesObserved < 0 || physical.bytesObserved > 2 ||
		physical.accepted != (physical.bytesObserved == 1 && physical.observedByte == httpmodel.ReadinessSuccessByte &&
			physical.eofObserved && physical.diagnosticCode == "") ||
		(!physical.accepted && physical.diagnosticCode == "") {
		return HTTPReadinessReceipt{}, refuse(CodeHTTPExecutionRejected, "readiness receipt input is incomplete", nil)
	}
	identity := httpReadinessReceiptIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPReadinessReceipt", Authority: httpReadinessReceiptAuthority,
		WorldDigest: worldDigest.String(), AttemptDigest: attemptDigest.String(), BindingDigest: bindingDigest.String(),
		Endpoint: physical.endpoint, Port: physical.port, ListenerFD: physical.listenerFD, ReadinessFD: physical.readinessFD,
		Protocol: httpmodel.ReadinessProtocolV1, BytesObserved: physical.bytesObserved, ObservedByte: int(physical.observedByte),
		EOFObserved: physical.eofObserved, Accepted: physical.accepted, DiagnosticCode: physical.diagnosticCode,
	}
	digest, canonicalBytes, err := canon.DigestTyped("HTTPReadinessReceipt", identity)
	if err != nil {
		return HTTPReadinessReceipt{}, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return HTTPReadinessReceipt{}, err
	}
	return HTTPReadinessReceipt{
		digest: parsed, canonicalBytes: canonicalBytes, worldDigest: worldDigest, attemptDigest: attemptDigest,
		bindingDigest: bindingDigest, endpoint: physical.endpoint, port: physical.port,
		listenerFD: physical.listenerFD, readinessFD: physical.readinessFD, bytesObserved: physical.bytesObserved,
		observedByte: physical.observedByte, eofObserved: physical.eofObserved, accepted: physical.accepted,
		diagnosticCode: physical.diagnosticCode,
	}, nil
}

func (r HTTPReadinessReceipt) Digest() domain.Digest { return r.digest }
func (r HTTPReadinessReceipt) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPReadinessReceipt) WorldDigest() domain.Digest            { return r.worldDigest }
func (r HTTPReadinessReceipt) AttemptArtifactDigest() domain.Digest  { return r.attemptDigest }
func (r HTTPReadinessReceipt) ExecutionBindingDigest() domain.Digest { return r.bindingDigest }
func (r HTTPReadinessReceipt) Endpoint() string                      { return r.endpoint }
func (r HTTPReadinessReceipt) Port() int                             { return r.port }
func (r HTTPReadinessReceipt) ListenerFD() int                       { return r.listenerFD }
func (r HTTPReadinessReceipt) ReadinessFD() int                      { return r.readinessFD }
func (r HTTPReadinessReceipt) Protocol() string                      { return httpmodel.ReadinessProtocolV1 }
func (r HTTPReadinessReceipt) BytesObserved() int64                  { return r.bytesObserved }
func (r HTTPReadinessReceipt) ObservedByte() byte                    { return r.observedByte }
func (r HTTPReadinessReceipt) EOFObserved() bool                     { return r.eofObserved }
func (r HTTPReadinessReceipt) Accepted() bool                        { return r.accepted }
func (r HTTPReadinessReceipt) DiagnosticCode() string                { return r.diagnosticCode }
func (r HTTPReadinessReceipt) Authority() string                     { return httpReadinessReceiptAuthority }
func (r HTTPReadinessReceipt) Valid() bool {
	rebuilt, err := newHTTPReadinessReceipt(r.worldDigest, r.attemptDigest, r.bindingDigest, httpReadinessPhysical{
		listenerFD: r.listenerFD, readinessFD: r.readinessFD, endpoint: r.endpoint, port: r.port,
		bytesObserved: r.bytesObserved, observedByte: r.observedByte, eofObserved: r.eofObserved,
		accepted: r.accepted, diagnosticCode: r.diagnosticCode,
	})
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}

type HTTPExchangeReceipt struct {
	digest               domain.Digest
	canonicalBytes       []byte
	worldDigest          domain.Digest
	attemptDigest        domain.Digest
	bindingDigest        domain.Digest
	endpoint             string
	connectionAttempts   int
	requestWire          []byte
	requestDigest        domain.Digest
	requestBytesDigest   domain.Digest
	requestRawSHA256     string
	requestWritten       int64
	requestComplete      bool
	responseWire         []byte
	responseDigest       domain.Digest
	responseObserved     int64
	responseOverflow     bool
	parsedResponseDigest domain.Digest
	status               int
	responseParsed       bool
	diagnosticCode       string
}

type httpExchangeReceiptIdentity struct {
	SchemaVersion         string `json:"schema_version"`
	Kind                  string `json:"kind"`
	Authority             string `json:"authority"`
	WorldDigest           string `json:"world_instance_digest"`
	AttemptDigest         string `json:"attempt_artifact_digest"`
	BindingDigest         string `json:"http_execution_binding_digest"`
	Endpoint              string `json:"literal_loopback_endpoint"`
	Transport             string `json:"transport"`
	ConnectionAttempts    int    `json:"connection_attempts"`
	RequestDigest         string `json:"request_wire_digest"`
	RequestBytesDigest    string `json:"request_wire_bytes_digest"`
	RequestRawSHA256      string `json:"request_wire_raw_sha256"`
	RequestBytes          int64  `json:"request_wire_bytes"`
	RequestWritten        int64  `json:"request_written_bytes"`
	RequestComplete       bool   `json:"request_write_complete"`
	ResponseDigest        string `json:"response_wire_digest"`
	ResponseRetainedBytes int64  `json:"response_retained_bytes"`
	ResponseObserved      int64  `json:"response_observed_bytes"`
	ResponseOverflow      bool   `json:"response_overflow"`
	ParsedResponseDigest  string `json:"parsed_response_digest"`
	Status                int    `json:"status"`
	ResponseParsed        bool   `json:"response_parsed"`
	Redirects             int    `json:"redirects_followed"`
	Retries               int    `json:"retries"`
	Proxy                 string `json:"proxy"`
	DNS                   string `json:"dns"`
	CookieJar             string `json:"cookie_jar"`
	Compression           string `json:"compression"`
	DiagnosticCode        string `json:"diagnostic_code"`
}

func newHTTPExchangeReceipt(
	worldDigest, attemptDigest, bindingDigest domain.Digest,
	endpoint string,
	physical httpExchangePhysical,
) (HTTPExchangeReceipt, error) {
	if !worldDigest.Valid() || !attemptDigest.Valid() || !bindingDigest.Valid() || endpoint == "" ||
		endpoint != "127.0.0.1:"+strconv.Itoa(physicalEndpointPort(endpoint)) || physicalEndpointPort(endpoint) < 1 ||
		physical.connectionAttempts < 0 || physical.connectionAttempts > 1 || physical.requestWritten < 0 ||
		physical.requestWritten > int64(len(physical.requestWire)) || physical.responseObserved < int64(len(physical.responseWire)) ||
		(physical.connectionAttempts == 0 && (physical.requestWritten != 0 || physical.requestComplete || physical.responseObserved != 0 || physical.responseParsed)) ||
		len(physical.requestWire) == 0 || !physical.requestSemanticDigest.Valid() || physical.requestRawSHA256 == "" ||
		physical.requestRawSHA256 != rawSHA256Hex(physical.requestWire) ||
		(physical.connectionAttempts == 1 && len(physical.requestWire) == 0) ||
		(physical.requestComplete != (physical.requestWritten == int64(len(physical.requestWire)) && len(physical.requestWire) > 0)) ||
		(physical.responseParsed && (!physical.responseSummary.digest.Valid() || !physical.responseSummary.wireDigest.Valid() || physical.responseSummary.status < 200 ||
			physical.responseSummary.status > 599 || physical.responseOverflow || !physical.requestComplete)) ||
		(!physical.responseParsed && (physical.responseSummary.digest.Valid() || physical.responseSummary.wireDigest.Valid() || physical.responseSummary.status != 0)) {
		return HTTPExchangeReceipt{}, refuse(CodeHTTPExecutionRejected, "exchange receipt input is incomplete", nil)
	}
	requestDigestRaw, err := canon.DigestBytes("HTTPRequestWireBytes", physical.requestWire)
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	requestBytesDigest, err := domain.ParseDigest(requestDigestRaw.String())
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	responseDigestRaw, err := canon.DigestBytes("HTTPResponseWireBytes", physical.responseWire)
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	responseDigest, err := domain.ParseDigest(responseDigestRaw.String())
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	if physical.responseParsed && physical.responseSummary.wireDigest != responseDigest {
		return HTTPExchangeReceipt{}, refuse(CodeHTTPExecutionRejected, "parsed response wire digest differs from retained bytes", nil)
	}
	identity := httpExchangeReceiptIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPExchangeReceipt", Authority: httpExchangeReceiptAuthority, WorldDigest: worldDigest.String(),
		AttemptDigest: attemptDigest.String(), BindingDigest: bindingDigest.String(), Endpoint: endpoint,
		Transport: "DIRECT_TCP4_LITERAL_127_0_0_1", ConnectionAttempts: physical.connectionAttempts,
		RequestDigest: physical.requestSemanticDigest.String(), RequestBytesDigest: requestBytesDigest.String(),
		RequestRawSHA256: physical.requestRawSHA256, RequestBytes: int64(len(physical.requestWire)), RequestWritten: physical.requestWritten,
		RequestComplete: physical.requestComplete, ResponseDigest: responseDigest.String(),
		ResponseRetainedBytes: int64(len(physical.responseWire)), ResponseObserved: physical.responseObserved,
		ResponseOverflow: physical.responseOverflow, ParsedResponseDigest: physical.responseSummary.digest.String(),
		Status: physical.responseSummary.status, ResponseParsed: physical.responseParsed,
		Redirects: 0, Retries: 0, Proxy: "NONE", DNS: "NONE_LITERAL_IP", CookieJar: "NONE", Compression: "NONE",
		DiagnosticCode: physical.diagnosticCode,
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("HTTPExchangeReceipt", identity)
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return HTTPExchangeReceipt{}, err
	}
	return HTTPExchangeReceipt{
		digest: digest, canonicalBytes: canonicalBytes, worldDigest: worldDigest, attemptDigest: attemptDigest,
		bindingDigest: bindingDigest, endpoint: endpoint, connectionAttempts: physical.connectionAttempts,
		requestWire: append([]byte(nil), physical.requestWire...), requestDigest: physical.requestSemanticDigest,
		requestBytesDigest: requestBytesDigest, requestRawSHA256: physical.requestRawSHA256,
		requestWritten: physical.requestWritten, requestComplete: physical.requestComplete,
		responseWire: append([]byte(nil), physical.responseWire...), responseDigest: responseDigest,
		responseObserved: physical.responseObserved, responseOverflow: physical.responseOverflow,
		parsedResponseDigest: physical.responseSummary.digest, status: physical.responseSummary.status,
		responseParsed: physical.responseParsed, diagnosticCode: physical.diagnosticCode,
	}, nil
}

func physicalEndpointPort(endpoint string) int {
	const prefix = "127.0.0.1:"
	if !strings.HasPrefix(endpoint, prefix) {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimPrefix(endpoint, prefix))
	if err != nil || port < 1 || port > 65535 || endpoint != prefix+strconv.Itoa(port) {
		return 0
	}
	return port
}

func (r HTTPExchangeReceipt) Digest() domain.Digest                 { return r.digest }
func (r HTTPExchangeReceipt) CanonicalBytes() []byte                { return append([]byte(nil), r.canonicalBytes...) }
func (r HTTPExchangeReceipt) WorldDigest() domain.Digest            { return r.worldDigest }
func (r HTTPExchangeReceipt) AttemptArtifactDigest() domain.Digest  { return r.attemptDigest }
func (r HTTPExchangeReceipt) ExecutionBindingDigest() domain.Digest { return r.bindingDigest }
func (r HTTPExchangeReceipt) Endpoint() string                      { return r.endpoint }
func (r HTTPExchangeReceipt) ConnectionAttempts() int               { return r.connectionAttempts }
func (r HTTPExchangeReceipt) RequestWire() []byte                   { return append([]byte(nil), r.requestWire...) }
func (r HTTPExchangeReceipt) RequestWireDigest() domain.Digest      { return r.requestDigest }
func (r HTTPExchangeReceipt) RequestWireBytesDigest() domain.Digest { return r.requestBytesDigest }
func (r HTTPExchangeReceipt) RequestWireRawSHA256() string          { return r.requestRawSHA256 }
func (r HTTPExchangeReceipt) RequestWrittenBytes() int64            { return r.requestWritten }
func (r HTTPExchangeReceipt) RequestComplete() bool                 { return r.requestComplete }
func (r HTTPExchangeReceipt) ResponseWire() []byte                  { return append([]byte(nil), r.responseWire...) }
func (r HTTPExchangeReceipt) ResponseWireDigest() domain.Digest     { return r.responseDigest }
func (r HTTPExchangeReceipt) ResponseObservedBytes() int64          { return r.responseObserved }
func (r HTTPExchangeReceipt) ResponseOverflow() bool                { return r.responseOverflow }
func (r HTTPExchangeReceipt) ParsedResponseDigest() (domain.Digest, bool) {
	return r.parsedResponseDigest, r.responseParsed
}
func (r HTTPExchangeReceipt) Status() (int, bool)      { return r.status, r.responseParsed }
func (r HTTPExchangeReceipt) ResponseParsed() bool     { return r.responseParsed }
func (r HTTPExchangeReceipt) DiagnosticCode() string   { return r.diagnosticCode }
func (r HTTPExchangeReceipt) Authority() string        { return httpExchangeReceiptAuthority }
func (r HTTPExchangeReceipt) TransportProfile() string { return "DIRECT_TCP4_LITERAL_127_0_0_1" }
func (r HTTPExchangeReceipt) RedirectsFollowed() int   { return 0 }
func (r HTTPExchangeReceipt) Retries() int             { return 0 }
func (r HTTPExchangeReceipt) Proxy() string            { return "NONE" }
func (r HTTPExchangeReceipt) DNS() string              { return "NONE_LITERAL_IP" }
func (r HTTPExchangeReceipt) CookieJar() string        { return "NONE" }
func (r HTTPExchangeReceipt) Compression() string      { return "NONE" }
func (r HTTPExchangeReceipt) Valid() bool {
	responseSummary := httpResponseSummary{}
	if r.responseParsed {
		responseSummary = httpResponseSummary{
			digest: r.parsedResponseDigest, wireDigest: r.responseDigest, status: r.status,
		}
	}
	rebuilt, err := newHTTPExchangeReceipt(r.worldDigest, r.attemptDigest, r.bindingDigest, r.endpoint, httpExchangePhysical{
		connectionAttempts: r.connectionAttempts, requestWire: r.requestWire,
		requestSemanticDigest: r.requestDigest, requestRawSHA256: r.requestRawSHA256, requestWritten: r.requestWritten,
		requestComplete: r.requestComplete, responseWire: r.responseWire, responseObserved: r.responseObserved,
		responseOverflow: r.responseOverflow,
		responseSummary:  responseSummary,
		responseParsed:   r.responseParsed, diagnosticCode: r.diagnosticCode,
	})
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}

type HTTPInvocationEvidenceStatus string

const (
	HTTPInvocationNotInspected     HTTPInvocationEvidenceStatus = "NOT_INSPECTED"
	HTTPInvocationAbsent           HTTPInvocationEvidenceStatus = "ABSENT"
	HTTPInvocationReadFailed       HTTPInvocationEvidenceStatus = "READ_FAILED"
	HTTPInvocationNotRegular       HTTPInvocationEvidenceStatus = "NOT_REGULAR"
	HTTPInvocationTooLarge         HTTPInvocationEvidenceStatus = "TOO_LARGE"
	HTTPInvocationMalformed        HTTPInvocationEvidenceStatus = "MALFORMED"
	HTTPInvocationSchemaMismatch   HTTPInvocationEvidenceStatus = "SCHEMA_MISMATCH"
	HTTPInvocationAttemptMismatch  HTTPInvocationEvidenceStatus = "ATTEMPT_MISMATCH"
	HTTPInvocationStimulusMismatch HTTPInvocationEvidenceStatus = "STIMULUS_MISMATCH"
	HTTPInvocationCountMismatch    HTTPInvocationEvidenceStatus = "INVOCATION_COUNT_MISMATCH"
	HTTPInvocationValidated        HTTPInvocationEvidenceStatus = "VALIDATED"
)

type HTTPInvocationEvidenceReceipt struct {
	digest                 domain.Digest
	canonicalBytes         []byte
	worldDigest            domain.Digest
	attemptDigest          domain.Digest
	bindingDigest          domain.Digest
	path                   string
	status                 HTTPInvocationEvidenceStatus
	fileDigest             domain.Digest
	fileBytes              int64
	expectedAttemptID      string
	expectedStimulusDigest domain.Digest
	expectedRequestBytes   int64
	expectedRequestSHA256  string
	attemptValidated       bool
	stimulusValidated      bool
	requestBytesValidated  bool
	requestDigestValidated bool
	countValidated         bool
}

type httpInvocationReceiptIdentity struct {
	SchemaVersion            string `json:"schema_version"`
	Kind                     string `json:"kind"`
	Authority                string `json:"authority"`
	WorldDigest              string `json:"world_instance_digest"`
	AttemptDigest            string `json:"attempt_artifact_digest"`
	BindingDigest            string `json:"http_execution_binding_digest"`
	Path                     string `json:"path"`
	Status                   string `json:"status"`
	FileDigest               string `json:"file_digest"`
	FileBytes                int64  `json:"file_bytes"`
	ExpectedAttemptID        string `json:"expected_attempt_id"`
	ExpectedStimulusDigest   string `json:"expected_stimulus_digest"`
	ExpectedRequestBytes     int64  `json:"expected_request_bytes"`
	ExpectedRequestSHA256    string `json:"expected_request_sha256"`
	AttemptValidated         bool   `json:"attempt_validated"`
	StimulusValidated        bool   `json:"stimulus_validated"`
	RequestBytesValidated    bool   `json:"request_bytes_validated"`
	RequestDigestValidated   bool   `json:"request_digest_validated"`
	InvocationCountValidated bool   `json:"invocation_count_validated"`
}

func inspectHTTPInvocationEvidence(
	worldDigest, attemptDigest, bindingDigest, stimulusDigest domain.Digest,
	evidenceRoot, attemptID string,
	requestWire []byte,
) (HTTPInvocationEvidenceReceipt, error) {
	if !worldDigest.Valid() || !attemptDigest.Valid() || !bindingDigest.Valid() || !stimulusDigest.Valid() || attemptID == "" || !filepath.IsAbs(evidenceRoot) {
		return HTTPInvocationEvidenceReceipt{}, refuse(CodeHTTPExecutionRejected, "invocation receipt input is incomplete", nil)
	}
	path := filepath.Join(evidenceRoot, httpInvocationFilename)
	status := HTTPInvocationAbsent
	var fileDigest domain.Digest
	var fileBytes int64
	var attemptValidated, stimulusValidated, requestBytesValidated, requestDigestValidated, countValidated bool
	expectedRequestSHA256 := rawSHA256(requestWire)
	info, statErr := os.Lstat(path)
	if statErr == nil {
		switch {
		case !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0:
			status = HTTPInvocationNotRegular
		case info.Size() > maxHTTPInvocationBytes:
			status, fileBytes = HTTPInvocationTooLarge, info.Size()
		default:
			status, fileDigest, fileBytes, attemptValidated, stimulusValidated, requestBytesValidated, requestDigestValidated, countValidated =
				readAndValidateHTTPInvocation(path, info.Size(), attemptID, stimulusDigest, requestWire)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		status = HTTPInvocationReadFailed
	}
	receipt := HTTPInvocationEvidenceReceipt{
		worldDigest: worldDigest, attemptDigest: attemptDigest, bindingDigest: bindingDigest,
		path: path, status: status, fileDigest: fileDigest, fileBytes: fileBytes,
		expectedAttemptID: attemptID, expectedStimulusDigest: stimulusDigest,
		expectedRequestBytes: int64(len(requestWire)), expectedRequestSHA256: expectedRequestSHA256,
		attemptValidated: attemptValidated, stimulusValidated: stimulusValidated,
		requestBytesValidated: requestBytesValidated, requestDigestValidated: requestDigestValidated,
		countValidated: countValidated,
	}
	identity := httpInvocationReceiptIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPInvocationEvidenceReceipt", Authority: httpInvocationReceiptAuthority,
		WorldDigest: worldDigest.String(), AttemptDigest: attemptDigest.String(), BindingDigest: bindingDigest.String(),
		Path: path, Status: string(status), FileDigest: fileDigest.String(), FileBytes: fileBytes,
		ExpectedAttemptID: attemptID, ExpectedStimulusDigest: stimulusDigest.String(), AttemptValidated: attemptValidated,
		ExpectedRequestBytes: int64(len(requestWire)), ExpectedRequestSHA256: expectedRequestSHA256,
		StimulusValidated: stimulusValidated, RequestBytesValidated: requestBytesValidated,
		RequestDigestValidated: requestDigestValidated, InvocationCountValidated: countValidated,
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("HTTPInvocationEvidenceReceipt", identity)
	if err != nil {
		return HTTPInvocationEvidenceReceipt{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return HTTPInvocationEvidenceReceipt{}, err
	}
	receipt.digest, receipt.canonicalBytes = digest, canonicalBytes
	if !receipt.Valid() {
		return HTTPInvocationEvidenceReceipt{}, refuse(CodeHTTPExecutionRejected, "constructed invocation evidence receipt is invalid", nil)
	}
	return receipt, nil
}

func readAndValidateHTTPInvocation(
	path string,
	expectedSize int64,
	expectedAttemptID string,
	expectedStimulusDigest domain.Digest,
	expectedRequestWire []byte,
) (HTTPInvocationEvidenceStatus, domain.Digest, int64, bool, bool, bool, bool, bool) {
	handle, err := os.Open(path)
	if err != nil {
		return HTTPInvocationReadFailed, "", 0, false, false, false, false, false
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Size() != expectedSize || opened.Size() > maxHTTPInvocationBytes {
		_ = handle.Close()
		return HTTPInvocationReadFailed, "", 0, false, false, false, false, false
	}
	data, readErr := io.ReadAll(io.LimitReader(handle, maxHTTPInvocationBytes+1))
	closeErr := handle.Close()
	if readErr != nil || closeErr != nil || int64(len(data)) != expectedSize || int64(len(data)) > maxHTTPInvocationBytes {
		return HTTPInvocationReadFailed, "", int64(len(data)), false, false, false, false, false
	}
	digestRaw, err := canon.DigestBytes("HTTPInvocationEvidenceBytes", data)
	if err != nil {
		return HTTPInvocationReadFailed, "", int64(len(data)), false, false, false, false, false
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return HTTPInvocationReadFailed, "", int64(len(data)), false, false, false, false, false
	}
	value, err := canon.Parse(data)
	if err != nil || !bytes.Equal(value.Canonical(), data) {
		return HTTPInvocationMalformed, digest, int64(len(data)), false, false, false, false, false
	}
	members, object := value.Members()
	if !object || len(members) != 7 {
		return HTTPInvocationSchemaMismatch, digest, int64(len(data)), false, false, false, false, false
	}
	attemptValue, hasAttempt := value.LookupMember("attempt_id")
	countValue, hasCount := value.LookupMember("invocation_count")
	kindValue, hasKind := value.LookupMember("kind")
	requestByteCountValue, hasRequestByteCount := value.LookupMember("request_byte_count")
	requestSHA256Value, hasRequestSHA256 := value.LookupMember("request_byte_sha256")
	schemaValue, hasSchema := value.LookupMember("schema_version")
	stimulusValue, hasStimulus := value.LookupMember("stimulus_digest")
	attempt, attemptString := attemptValue.Text()
	count, countInteger := countValue.Int64()
	kind, kindString := kindValue.Text()
	requestByteCount, requestByteCountInteger := requestByteCountValue.Int64()
	requestSHA256, requestSHA256String := requestSHA256Value.Text()
	schema, schemaString := schemaValue.Text()
	stimulus, stimulusString := stimulusValue.Text()
	if !hasAttempt || !hasCount || !hasKind || !hasRequestByteCount || !hasRequestSHA256 || !hasSchema || !hasStimulus ||
		!attemptString || !countInteger || !kindString || !requestByteCountInteger || !requestSHA256String ||
		!schemaString || !stimulusString || kind != "HTTPFixtureInvocationEvidence" || schema != domain.SchemaVersion {
		return HTTPInvocationSchemaMismatch, digest, int64(len(data)), false, false, false, false, false
	}
	attemptMatches := attempt == expectedAttemptID
	stimulusMatches := stimulus == expectedStimulusDigest.String()
	requestByteCountMatches := requestByteCount == int64(len(expectedRequestWire))
	requestSHA256Matches := requestSHA256 == rawSHA256(expectedRequestWire)
	countMatches := count == 1
	if !attemptMatches {
		return HTTPInvocationAttemptMismatch, digest, int64(len(data)), false, stimulusMatches, requestByteCountMatches, requestSHA256Matches, countMatches
	}
	if !stimulusMatches {
		return HTTPInvocationStimulusMismatch, digest, int64(len(data)), true, false, requestByteCountMatches, requestSHA256Matches, countMatches
	}
	if !requestByteCountMatches || !requestSHA256Matches {
		return HTTPInvocationSchemaMismatch, digest, int64(len(data)), true, true, requestByteCountMatches, requestSHA256Matches, countMatches
	}
	if !countMatches {
		return HTTPInvocationCountMismatch, digest, int64(len(data)), true, true, true, true, false
	}
	return HTTPInvocationValidated, digest, int64(len(data)), true, true, true, true, true
}

func rawSHA256(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func rawSHA256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func (r HTTPInvocationEvidenceReceipt) Digest() domain.Digest { return r.digest }
func (r HTTPInvocationEvidenceReceipt) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPInvocationEvidenceReceipt) WorldDigest() domain.Digest            { return r.worldDigest }
func (r HTTPInvocationEvidenceReceipt) AttemptArtifactDigest() domain.Digest  { return r.attemptDigest }
func (r HTTPInvocationEvidenceReceipt) ExecutionBindingDigest() domain.Digest { return r.bindingDigest }
func (r HTTPInvocationEvidenceReceipt) Path() string                          { return r.path }
func (r HTTPInvocationEvidenceReceipt) Status() HTTPInvocationEvidenceStatus  { return r.status }
func (r HTTPInvocationEvidenceReceipt) Authority() string                     { return httpInvocationReceiptAuthority }
func (r HTTPInvocationEvidenceReceipt) FileDigest() (domain.Digest, bool) {
	return r.fileDigest, r.fileDigest.Valid()
}
func (r HTTPInvocationEvidenceReceipt) FileBytes() int64        { return r.fileBytes }
func (r HTTPInvocationEvidenceReceipt) AttemptValidated() bool  { return r.attemptValidated }
func (r HTTPInvocationEvidenceReceipt) StimulusValidated() bool { return r.stimulusValidated }
func (r HTTPInvocationEvidenceReceipt) RequestByteCountValidated() bool {
	return r.requestBytesValidated
}
func (r HTTPInvocationEvidenceReceipt) RequestDigestValidated() bool   { return r.requestDigestValidated }
func (r HTTPInvocationEvidenceReceipt) InvocationCountValidated() bool { return r.countValidated }
func (r HTTPInvocationEvidenceReceipt) Validated() bool {
	return r.status == HTTPInvocationValidated && r.attemptValidated && r.stimulusValidated &&
		r.requestBytesValidated && r.requestDigestValidated && r.countValidated
}

func (r HTTPInvocationEvidenceReceipt) Valid() bool {
	if !r.digest.Valid() || !r.worldDigest.Valid() || !r.attemptDigest.Valid() || !r.bindingDigest.Valid() || !r.expectedStimulusDigest.Valid() ||
		r.path == "" || r.expectedAttemptID == "" || r.expectedRequestBytes < 0 || r.expectedRequestSHA256 == "" ||
		!validHTTPInvocationStatus(r.status) {
		return false
	}
	identity := httpInvocationReceiptIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPInvocationEvidenceReceipt", Authority: httpInvocationReceiptAuthority,
		WorldDigest: r.worldDigest.String(), AttemptDigest: r.attemptDigest.String(),
		BindingDigest: r.bindingDigest.String(), Path: r.path, Status: string(r.status),
		FileDigest: r.fileDigest.String(), FileBytes: r.fileBytes, ExpectedAttemptID: r.expectedAttemptID,
		ExpectedStimulusDigest: r.expectedStimulusDigest.String(), ExpectedRequestBytes: r.expectedRequestBytes,
		ExpectedRequestSHA256: r.expectedRequestSHA256, AttemptValidated: r.attemptValidated,
		StimulusValidated: r.stimulusValidated, RequestBytesValidated: r.requestBytesValidated,
		RequestDigestValidated: r.requestDigestValidated, InvocationCountValidated: r.countValidated,
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("HTTPInvocationEvidenceReceipt", identity)
	return err == nil && digestRaw.String() == r.digest.String() && bytes.Equal(canonicalBytes, r.canonicalBytes) &&
		(r.status != HTTPInvocationValidated || r.Validated())
}

func validHTTPInvocationStatus(status HTTPInvocationEvidenceStatus) bool {
	switch status {
	case HTTPInvocationNotInspected, HTTPInvocationAbsent, HTTPInvocationReadFailed, HTTPInvocationNotRegular,
		HTTPInvocationTooLarge, HTTPInvocationMalformed, HTTPInvocationSchemaMismatch,
		HTTPInvocationAttemptMismatch, HTTPInvocationStimulusMismatch, HTTPInvocationCountMismatch,
		HTTPInvocationValidated:
		return true
	default:
		return false
	}
}
