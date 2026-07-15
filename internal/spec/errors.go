package spec

import "fmt"

const (
	CodeUnparsedSource                 = "UNPARSED_SOURCE"
	CodeSourceTooLarge                 = "SOURCE_SPEC_TOO_LARGE"
	CodeUnknownSourceKey               = "UNKNOWN_SOURCE_KEY"
	CodeMissingSourceField             = "MISSING_SOURCE_FIELD"
	CodeInvalidSourceType              = "INVALID_SOURCE_TYPE"
	CodeInvalidSourceValue             = "INVALID_SOURCE_VALUE"
	CodeUnresolvedProjectionDefinition = "UNRESOLVED_PROJECTION_DEFINITION"
)

// Error is a closed source-grammar refusal. Canonical-token refusals retain
// their own canon.Error type and byte offset; this error adds a typed path for
// failures after lossless tokenization.
type Error struct {
	Code   string
	Path   string
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("%s at %s", e.Code, e.Path)
	}
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Detail)
}

func refuse(code, path, detail string) error {
	return &Error{Code: code, Path: path, Detail: detail}
}
