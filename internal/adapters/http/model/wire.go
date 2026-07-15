package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const HTTPWireProfileV1 = "DIRECT_TCP_HTTP_1_1_CONTENT_LENGTH_CLOSE_V1"

type HTTPRequestWire struct {
	digest         domain.Digest
	canonicalBytes []byte
	bytes          []byte
	rawSHA256      string
	stimulus       HTTPStimulus
	port           int
	target         string
}

func EncodeRequest(stimulus HTTPStimulus, port int) (HTTPRequestWire, error) {
	if !stimulus.Valid() {
		return HTTPRequestWire{}, refuse(CodeWireEncode, "stimulus authority is invalid")
	}
	if port < 1 || port > 65535 {
		return HTTPRequestWire{}, refuse(CodeWireInvalidPort, strconv.Itoa(port))
	}
	target := encodeTarget(stimulus.Path(), stimulus.Query())
	var output bytes.Buffer
	output.WriteString(string(stimulus.Method()))
	output.WriteByte(' ')
	output.WriteString(target)
	output.WriteString(" HTTP/1.1\r\n")
	output.WriteString("host: 127.0.0.1:")
	output.WriteString(strconv.Itoa(port))
	output.WriteString("\r\n")
	output.WriteString("connection: close\r\n")
	for _, header := range stimulus.Headers() {
		output.WriteString(header.Name())
		output.WriteString(": ")
		output.WriteString(header.Value())
		output.WriteString("\r\n")
	}
	body := stimulus.Body()
	// MUTATION_ANCHOR: request-body-absent-distinct-from-present-empty
	if body.Present() {
		output.WriteString("content-length: ")
		output.WriteString(strconv.Itoa(len(body.Bytes())))
		output.WriteString("\r\n")
	}
	output.WriteString("\r\n")
	output.Write(body.Bytes())
	wireBytes := output.Bytes()
	byteDigest, err := digestExactBytes("HTTPRequestWireBytes", wireBytes)
	if err != nil {
		return HTTPRequestWire{}, err
	}
	rawHash := sha256.Sum256(wireBytes)
	rawSHA256 := hex.EncodeToString(rawHash[:])
	identity := struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Profile       string `json:"profile"`
		Stimulus      string `json:"stimulus_digest"`
		Host          string `json:"host"`
		Port          int    `json:"port"`
		Target        string `json:"request_target"`
		ByteLength    int    `json:"byte_length"`
		ByteDigest    string `json:"byte_digest"`
		RawSHA256     string `json:"raw_sha256"`
	}{domain.SchemaVersion, "HTTPRequestWire", HTTPWireProfileV1, stimulus.Digest().String(), "127.0.0.1", port, target, len(wireBytes), byteDigest.String(), rawSHA256}
	digest, canonicalBytes, err := digestTyped("HTTPRequestWire", identity)
	if err != nil {
		return HTTPRequestWire{}, err
	}
	return HTTPRequestWire{
		digest: digest, canonicalBytes: canonicalBytes, bytes: append([]byte(nil), wireBytes...), rawSHA256: rawSHA256,
		stimulus: stimulus, port: port, target: target,
	}, nil
}

func (w HTTPRequestWire) Valid() bool {
	rebuilt, err := EncodeRequest(w.stimulus, w.port)
	return err == nil && rebuilt.digest == w.digest && rebuilt.rawSHA256 == w.rawSHA256 &&
		bytes.Equal(rebuilt.canonicalBytes, w.canonicalBytes) && bytes.Equal(rebuilt.bytes, w.bytes)
}
func (w HTTPRequestWire) Digest() domain.Digest  { return w.digest }
func (w HTTPRequestWire) CanonicalBytes() []byte { return append([]byte(nil), w.canonicalBytes...) }
func (w HTTPRequestWire) Bytes() []byte          { return append([]byte(nil), w.bytes...) }
func (w HTTPRequestWire) ByteLength() int        { return len(w.bytes) }
func (w HTTPRequestWire) RawSHA256() string      { return w.rawSHA256 }
func (w HTTPRequestWire) Port() int              { return w.port }
func (w HTTPRequestWire) Host() string           { return "127.0.0.1" }
func (w HTTPRequestWire) Endpoint() string       { return "127.0.0.1:" + strconv.Itoa(w.port) }
func (w HTTPRequestWire) Target() string         { return w.target }
func (w HTTPRequestWire) Profile() string        { return HTTPWireProfileV1 }

func encodeTarget(path string, query []HTTPQueryEntry) string {
	if len(query) == 0 {
		return path
	}
	var output strings.Builder
	output.WriteString(path)
	output.WriteByte('?')
	for index, entry := range query {
		if index > 0 {
			output.WriteByte('&')
		}
		output.WriteString(percentEncode(entry.Name()))
		if entry.Present() {
			value, _ := entry.Value()
			output.WriteByte('=')
			output.WriteString(percentEncode(value))
		}
	}
	return output.String()
}

func percentEncode(value string) string {
	const hexadecimal = "0123456789ABCDEF"
	var output strings.Builder
	for index := 0; index < len(value); index++ {
		b := value[index]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') ||
			b == '-' || b == '.' || b == '_' || b == '~' {
			output.WriteByte(b)
			continue
		}
		output.WriteByte('%')
		output.WriteByte(hexadecimal[b>>4])
		output.WriteByte(hexadecimal[b&15])
	}
	return output.String()
}

type HTTPResponseHeader struct{ name, value string }

func (h HTTPResponseHeader) Name() string  { return h.name }
func (h HTTPResponseHeader) Value() string { return h.value }

type HTTPResponse struct {
	digest         domain.Digest
	canonicalBytes []byte
	wireDigest     domain.Digest
	wireBytes      []byte
	policy         HTTPCapturePolicy
	status         int
	reason         string
	headers        []HTTPResponseHeader
	body           []byte
	contentLength  int64
}

func ParseResponse(raw []byte, policy HTTPCapturePolicy) (HTTPResponse, error) {
	if !policy.Valid() {
		return HTTPResponse{}, refuse(CodeInvalidCapturePolicy, "capture policy is invalid")
	}
	if int64(len(raw)) > policy.MaximumResponseWireBytes() {
		return HTTPResponse{}, refuse(CodeResponseBodyLimit, "retained response contains the overflow sentinel")
	}
	statusEnd := bytes.Index(raw, []byte("\r\n"))
	if statusEnd < 0 {
		return HTTPResponse{}, refuse(CodeResponseStatus, "status line has no CRLF")
	}
	if int64(statusEnd) > policy.StatusLineBytes() {
		return HTTPResponse{}, refuse(CodeResponseStatusLimit, "status line exceeds declared capture policy")
	}
	status, reason, err := parseStatusLine(raw[:statusEnd])
	if err != nil {
		return HTTPResponse{}, err
	}
	headerEndRelative := bytes.Index(raw[statusEnd+2:], []byte("\r\n\r\n"))
	if headerEndRelative < 0 {
		return HTTPResponse{}, refuse(CodeResponseHeader, "header block has no terminating CRLF")
	}
	headerStart := statusEnd + 2
	headerEnd := headerStart + headerEndRelative
	if int64(headerEnd-headerStart) > policy.HeaderBytes() {
		return HTTPResponse{}, refuse(CodeResponseHeaderLimit, "header bytes exceed declared capture policy")
	}
	headers, contentLength, err := parseResponseHeaders(raw[headerStart:headerEnd], policy.HeaderCount())
	if err != nil {
		return HTTPResponse{}, err
	}
	body := raw[headerEnd+4:]
	if contentLength < 0 {
		return HTTPResponse{}, refuse(CodeResponseLength, "exactly one content-length is required")
	}
	if contentLength > policy.BodyBytes() {
		return HTTPResponse{}, refuse(CodeResponseBodyLimit, "declared body exceeds capture policy")
	}
	if int64(len(body)) != contentLength {
		return HTTPResponse{}, refuse(CodeResponseLength, "received body or trailing bytes differ from content-length")
	}
	wireDigest, err := digestExactBytes("HTTPResponseWireBytes", raw)
	if err != nil {
		return HTTPResponse{}, err
	}
	type headerIdentity struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	headerIDs := make([]headerIdentity, len(headers))
	for index, header := range headers {
		headerIDs[index] = headerIdentity{Name: header.name, Value: header.value}
	}
	bodyDigest, err := digestExactBytes("HTTPResponseBodyBytes", body)
	if err != nil {
		return HTTPResponse{}, err
	}
	identity := struct {
		SchemaVersion string           `json:"schema_version"`
		Kind          string           `json:"kind"`
		Profile       string           `json:"profile"`
		CapturePolicy string           `json:"capture_policy_digest"`
		Status        int              `json:"status"`
		Reason        string           `json:"reason"`
		Headers       []headerIdentity `json:"ordered_response_header_multimap"`
		ContentLength int64            `json:"content_length"`
		BodyDigest    string           `json:"body_digest"`
		WireBytes     int              `json:"wire_bytes"`
		WireDigest    string           `json:"wire_digest"`
	}{domain.SchemaVersion, "HTTPResponse", HTTPWireProfileV1, policy.Digest().String(), status, reason, headerIDs, contentLength, bodyDigest.String(), len(raw), wireDigest.String()}
	digest, canonicalBytes, err := digestTyped("HTTPResponse", identity)
	if err != nil {
		return HTTPResponse{}, err
	}
	return HTTPResponse{digest: digest, canonicalBytes: canonicalBytes, wireDigest: wireDigest, wireBytes: append([]byte(nil), raw...), policy: policy, status: status, reason: reason, headers: headers, body: append([]byte(nil), body...), contentLength: contentLength}, nil
}

func parseStatusLine(line []byte) (int, string, error) {
	if len(line) < 13 || !bytes.HasPrefix(line, []byte("HTTP/1.1 ")) || line[12] != ' ' {
		return 0, "", refuse(CodeResponseStatus, "status line is outside exact HTTP/1.1 final-response grammar")
	}
	codeBytes := line[9:12]
	for _, b := range codeBytes {
		if b < '0' || b > '9' {
			return 0, "", refuse(CodeResponseStatus, "status code is not three decimal digits")
		}
	}
	status, _ := strconv.Atoi(string(codeBytes))
	if status < 200 || status > 599 {
		return 0, "", refuse(CodeResponseStatus, "informational or out-of-range status is not a complete v1 response")
	}
	// MUTATION_ANCHOR: status-500-remains-complete-application-response
	reason := string(line[13:])
	for index := 0; index < len(reason); index++ {
		if reason[index] < 0x20 || reason[index] > 0x7e {
			return 0, "", refuse(CodeResponseStatus, "reason phrase contains unsupported bytes")
		}
	}
	return status, reason, nil
}

func parseResponseHeaders(block []byte, maxCount int) ([]HTTPResponseHeader, int64, error) {
	if len(block) == 0 {
		return nil, -1, nil
	}
	lines := bytes.Split(block, []byte("\r\n"))
	if len(lines) > maxCount {
		return nil, -1, refuse(CodeResponseHeaderLimit, "header count exceeds declared capture policy")
	}
	result := make([]HTTPResponseHeader, 0, len(lines))
	contentLength := int64(-1)
	for _, line := range lines {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
			return nil, -1, refuse(CodeResponseHeader, "empty or folded header line")
		}
		colon := bytes.IndexByte(line, ':')
		if colon < 1 || colon+1 >= len(line) || line[colon+1] != ' ' {
			return nil, -1, refuse(CodeResponseHeader, "header must use exact name-colon-space-value grammar")
		}
		name := strings.ToLower(string(line[:colon]))
		value := string(line[colon+2:])
		if !validWireHeaderName(name) || !validWireHeaderValue(value) {
			return nil, -1, refuse(CodeResponseHeader, "header contains unsupported name or value bytes")
		}
		switch name {
		case "transfer-encoding":
			return nil, -1, refuse(CodeResponseTransferEncoding, value)
		case "content-encoding":
			return nil, -1, refuse(CodeResponseContentEncoding, value)
		case "content-length":
			if contentLength >= 0 || value == "" || (len(value) > 1 && value[0] == '0') {
				return nil, -1, refuse(CodeResponseLength, "duplicate or noncanonical content-length")
			}
			for index := 0; index < len(value); index++ {
				if value[index] < '0' || value[index] > '9' {
					return nil, -1, refuse(CodeResponseLength, "content-length is not canonical decimal")
				}
			}
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, -1, refuse(CodeResponseLength, "content-length exceeds integer profile")
			}
			contentLength = parsed
		}
		result = append(result, HTTPResponseHeader{name: name, value: value})
	}
	return result, contentLength, nil
}

func validWireHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		b := value[index]
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
			continue
		}
		return false
	}
	return true
}
func validWireHeaderValue(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

func (r HTTPResponse) Valid() bool {
	rebuilt, err := ParseResponse(r.wireBytes, r.policy)
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}
func (r HTTPResponse) Digest() domain.Digest     { return r.digest }
func (r HTTPResponse) CanonicalBytes() []byte    { return append([]byte(nil), r.canonicalBytes...) }
func (r HTTPResponse) WireDigest() domain.Digest { return r.wireDigest }
func (r HTTPResponse) WireBytes() []byte         { return append([]byte(nil), r.wireBytes...) }
func (r HTTPResponse) Status() int               { return r.status }
func (r HTTPResponse) Reason() string            { return r.reason }
func (r HTTPResponse) Headers() []HTTPResponseHeader {
	return append([]HTTPResponseHeader(nil), r.headers...)
}
func (r HTTPResponse) Body() []byte         { return append([]byte(nil), r.body...) }
func (r HTTPResponse) ContentLength() int64 { return r.contentLength }
func (r HTTPResponse) HeaderValues(name string) []string {
	name = strings.ToLower(name)
	result := []string{}
	for _, header := range r.headers {
		if header.name == name {
			result = append(result, header.value)
		}
	}
	return result
}
