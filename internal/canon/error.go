package canon

import "fmt"

// ErrorCode is a stable machine-readable refusal code. Callers must branch on
// Code, not on the human-oriented Error string.
type ErrorCode string

const (
	CodeInvalidUTF8       ErrorCode = "CANON_INVALID_UTF8"
	CodeUnexpectedByte    ErrorCode = "CANON_UNEXPECTED_BYTE"
	CodeUnexpectedEOF     ErrorCode = "CANON_UNEXPECTED_EOF"
	CodeInvalidLiteral    ErrorCode = "CANON_INVALID_LITERAL"
	CodeInvalidEscape     ErrorCode = "CANON_INVALID_ESCAPE"
	CodeLoneSurrogate     ErrorCode = "CANON_LONE_SURROGATE"
	CodeControlCharacter  ErrorCode = "CANON_CONTROL_CHARACTER"
	CodeUnsupportedNumber ErrorCode = "CANON_UNSUPPORTED_NUMBER"
	CodeNegativeZero      ErrorCode = "CANON_NEGATIVE_ZERO"
	CodeUnsafeInteger     ErrorCode = "CANON_UNSAFE_INTEGER"
	CodeDuplicateKey      ErrorCode = "CANON_DUPLICATE_KEY"
	CodeUnexpectedToken   ErrorCode = "CANON_UNEXPECTED_TOKEN"
	CodeTrailingData      ErrorCode = "CANON_TRAILING_DATA"
	CodeDepthExceeded     ErrorCode = "CANON_DEPTH_EXCEEDED"
	CodeInputLimit        ErrorCode = "CANON_INPUT_LIMIT"
	CodeTokenLimit        ErrorCode = "CANON_TOKEN_LIMIT"
	CodeMemberLimit       ErrorCode = "CANON_MEMBER_LIMIT"
	CodeTypedNodeLimit    ErrorCode = "CANON_TYPED_NODE_LIMIT"
	CodeTypedByteLimit    ErrorCode = "CANON_TYPED_BYTE_LIMIT"
	CodeInvalidKind       ErrorCode = "CANON_INVALID_KIND"
	CodeInvalidValue      ErrorCode = "CANON_INVALID_VALUE"
)

// UnknownOffset is used for constructor errors that do not originate in an
// input byte stream. Parser and scanner errors always carry a nonnegative,
// zero-based byte offset.
const UnknownOffset = -1

// Error is a typed canonicalization refusal. Offset is a zero-based byte
// offset into the original input, or UnknownOffset for programmatic values.
// Detail is deliberately generic so malformed identity input is not echoed
// into routine logs.
type Error struct {
	Code   ErrorCode
	Offset int
	Detail string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Offset == UnknownOffset {
		return fmt.Sprintf("canon: %s: %s", e.Code, e.Detail)
	}
	return fmt.Sprintf("canon: %s at byte %d: %s", e.Code, e.Offset, e.Detail)
}

func refusal(code ErrorCode, offset int, detail string) *Error {
	return &Error{Code: code, Offset: offset, Detail: detail}
}
