package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxStatusLineLimit = 8 << 10
	maxHeaderByteLimit = 1 << 20
	maxHeaderCount     = 1024
	maxBodyByteLimit   = 16 << 20
)

type HTTPCapturePolicyConfig struct {
	StatusLineBytes int64
	HeaderBytes     int64
	HeaderCount     int
	BodyBytes       int64
}

type HTTPCapturePolicy struct {
	digest          domain.Digest
	canonicalBytes  []byte
	statusLineBytes int64
	headerBytes     int64
	headerCount     int
	bodyBytes       int64
}

func NewHTTPCapturePolicy(config HTTPCapturePolicyConfig) (HTTPCapturePolicy, error) {
	if config.StatusLineBytes < 16 || config.StatusLineBytes > maxStatusLineLimit ||
		config.HeaderBytes < 16 || config.HeaderBytes > maxHeaderByteLimit ||
		config.HeaderCount < 1 || config.HeaderCount > maxHeaderCount ||
		config.BodyBytes < 1 || config.BodyBytes > maxBodyByteLimit {
		return HTTPCapturePolicy{}, refuse(CodeInvalidCapturePolicy, "wire capture limits are outside the closed v1 profile")
	}
	identity := struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Adapter       string `json:"adapter"`
		Version       string `json:"version"`
		StatusLine    int64  `json:"status_line_bytes"`
		HeaderBytes   int64  `json:"header_bytes"`
		HeaderCount   int    `json:"header_count"`
		BodyBytes     int64  `json:"body_bytes"`
		TransferMode  string `json:"transfer_mode"`
		EncodingMode  string `json:"content_encoding_mode"`
	}{domain.SchemaVersion, "HTTPCapturePolicy", string(domain.AdapterHTTP), "http-capture/v1", config.StatusLineBytes, config.HeaderBytes, config.HeaderCount, config.BodyBytes, "CONTENT_LENGTH_ONLY", "IDENTITY_ONLY"}
	digest, canonicalBytes, err := digestTyped("HTTPCapturePolicy", identity)
	if err != nil {
		return HTTPCapturePolicy{}, err
	}
	return HTTPCapturePolicy{
		digest: digest, canonicalBytes: canonicalBytes, statusLineBytes: config.StatusLineBytes,
		headerBytes: config.HeaderBytes, headerCount: config.HeaderCount, bodyBytes: config.BodyBytes,
	}, nil
}

func (p HTTPCapturePolicy) Valid() bool {
	rebuilt, err := NewHTTPCapturePolicy(HTTPCapturePolicyConfig{
		StatusLineBytes: p.statusLineBytes, HeaderBytes: p.headerBytes, HeaderCount: p.headerCount, BodyBytes: p.bodyBytes,
	})
	return err == nil && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonicalBytes, p.canonicalBytes)
}
func (p HTTPCapturePolicy) Digest() domain.Digest  { return p.digest }
func (p HTTPCapturePolicy) CanonicalBytes() []byte { return append([]byte(nil), p.canonicalBytes...) }
func (p HTTPCapturePolicy) StatusLineBytes() int64 { return p.statusLineBytes }
func (p HTTPCapturePolicy) HeaderBytes() int64     { return p.headerBytes }
func (p HTTPCapturePolicy) HeaderCount() int       { return p.headerCount }
func (p HTTPCapturePolicy) BodyBytes() int64       { return p.bodyBytes }

// MaximumResponseWireBytes is the maximum valid response size: bounded status
// bytes, its CRLF, the entire bounded header block and terminator, and bounded
// body. The physical owner reads one additional sentinel byte via
// OwnerResponseReadLimit so overflow is observed rather than silently cut.
func (p HTTPCapturePolicy) MaximumResponseWireBytes() int64 {
	if !p.Valid() {
		return 0
	}
	return p.statusLineBytes + 2 + p.headerBytes + 4 + p.bodyBytes
}

func (p HTTPCapturePolicy) MaxResponseWireBytes() int64          { return p.MaximumResponseWireBytes() }
func (p HTTPCapturePolicy) MaximumValidResponseWireBytes() int64 { return p.MaximumResponseWireBytes() }
func (p HTTPCapturePolicy) OwnerResponseReadLimit() int64 {
	maximum := p.MaximumResponseWireBytes()
	if maximum == 0 {
		return 0
	}
	return maximum + 1
}
