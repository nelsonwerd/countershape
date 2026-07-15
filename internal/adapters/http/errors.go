// Package http owns Countershape's typed HTTP stimulus, deterministic wire,
// capture, and projection facts. It deliberately uses no net/http client or
// server policy.
package http

import (
	"fmt"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	CodeWireInvalidPort          = "HTTP_WIRE_INVALID_PORT"
	CodeWireEncode               = "HTTP_WIRE_ENCODE"
	CodeResponseStatus           = "HTTP_RESPONSE_BAD_STATUS"
	CodeResponseStatusLimit      = "HTTP_RESPONSE_STATUS_LINE_LIMIT"
	CodeResponseHeader           = "HTTP_RESPONSE_BAD_HEADER"
	CodeResponseHeaderLimit      = "HTTP_RESPONSE_HEADER_LIMIT"
	CodeResponseBodyLimit        = "HTTP_RESPONSE_BODY_LIMIT"
	CodeResponseLength           = "HTTP_RESPONSE_CONTENT_LENGTH_MISMATCH"
	CodeResponseTransferEncoding = "HTTP_RESPONSE_TRANSFER_ENCODING_UNSUPPORTED"
	CodeResponseContentEncoding  = "HTTP_RESPONSE_CONTENT_ENCODING_UNSUPPORTED"
	CodeCaptureLineageMismatch   = "HTTP_CAPTURE_LINEAGE_MISMATCH"
	CodeCaptureEvidenceInvalid   = "HTTP_CAPTURE_EVIDENCE_INVALID"
	CodeInvalidProjection        = "HTTP_INVALID_PROJECTION_DEFINITION"
	CodeProjectionControl        = "HTTP_PROJECTION_CONTROL_INELIGIBLE"
	CodeProjectionChannel        = "HTTP_PROJECTION_CHANNEL_REQUIRED"
	CodeProjectionJSON           = "HTTP_PROJECTION_INVALID_STRICT_JSON"
	CodeProjectionJSONRoot       = "HTTP_PROJECTION_JSON_OBJECT_REQUIRED"
	CodeProjectionMetadata       = "HTTP_PROJECTION_METADATA_INVALID"
	CodeProjectionResourceLimit  = "HTTP_PROJECTION_RESOURCE_LIMIT"
	CodeProjectionInternal       = "HTTP_PROJECTION_INTERNAL_REFUSAL"
)

type Refusal struct {
	Code   string
	Detail string
}

func (r *Refusal) Error() string {
	if r == nil {
		return "<nil>"
	}
	if r.Detail == "" {
		return r.Code
	}
	return fmt.Sprintf("%s: %s", r.Code, r.Detail)
}

func refuse(code, detail string) error { return &Refusal{Code: code, Detail: detail} }

type ProjectionRejection struct {
	Code                    string
	Operation               string
	Channel                 HTTPChannel
	Field                   HTTPFieldID
	Detail                  string
	digest                  domain.Digest
	canonicalBytes          []byte
	observationDigest       domain.Digest
	adapterDefinitionDigest domain.Digest
	definitionDigest        domain.Digest
	transcript              []HTTPProjectionTraceEntry
}

func (r *ProjectionRejection) Error() string {
	if r == nil {
		return "<nil>"
	}
	location := r.Operation
	if r.Field != "" {
		location += "/" + string(r.Field)
	}
	if r.Channel != "" {
		location += "/" + string(r.Channel)
	}
	if r.Detail == "" {
		return fmt.Sprintf("%s: %s", r.Code, location)
	}
	return fmt.Sprintf("%s: %s: %s", r.Code, location, r.Detail)
}

func (r *ProjectionRejection) EvidenceDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.digest
}
func (r *ProjectionRejection) CanonicalBytes() []byte {
	if r == nil {
		return nil
	}
	return append([]byte(nil), r.canonicalBytes...)
}
func (r *ProjectionRejection) ObservationDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.observationDigest
}
func (r *ProjectionRejection) DefinitionDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.definitionDigest
}
func (r *ProjectionRejection) Transcript() []HTTPProjectionTraceEntry {
	if r == nil {
		return nil
	}
	return cloneTrace(r.transcript)
}
