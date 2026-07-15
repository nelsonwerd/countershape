// Package model owns the cycle-free HTTP execution input authority. The
// world edge may import this package without importing HTTP capture or
// projection implementations.
package model

import "fmt"

const (
	CodeInvalidStimulus          = "HTTP_INVALID_STIMULUS"
	CodeInvalidMethod            = "HTTP_INVALID_METHOD"
	CodeInvalidPath              = "HTTP_INVALID_PATH"
	CodeInvalidQuery             = "HTTP_INVALID_QUERY"
	CodeInvalidHeader            = "HTTP_INVALID_HEADER"
	CodeForbiddenHeader          = "HTTP_FORBIDDEN_HEADER"
	CodeInvalidBody              = "HTTP_INVALID_BODY"
	CodeInvalidSeed              = "HTTP_INVALID_SEED"
	CodeSeedOrder                = "HTTP_SEED_ORDER_INVALID"
	CodeSeedCollision            = "HTTP_SEED_PATH_COLLISION"
	CodeInputLimit               = "HTTP_INPUT_LIMIT"
	CodeInvalidStartSpec         = "HTTP_INVALID_START_SPEC"
	CodeInvalidFixtureRecipe     = "HTTP_INVALID_FIXTURE_RECIPE"
	CodeInvalidReadiness         = "HTTP_INVALID_READINESS_CONTRACT"
	CodeInvalidCapturePolicy     = "HTTP_INVALID_CAPTURE_POLICY"
	CodeInvalidBinding           = "HTTP_INVALID_EXECUTION_BINDING"
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

func RefusalCodeOf(err error) (string, bool) {
	refusal, ok := err.(*Refusal)
	if !ok || refusal == nil {
		return "", false
	}
	return refusal.Code, true
}
