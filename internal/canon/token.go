package canon

// TokenKind identifies one lossless lexical token. Whitespace is retained so
// concatenating Raw for all non-EOF tokens reproduces the exact input bytes.
type TokenKind uint8

const (
	TokenWhitespace TokenKind = iota
	TokenObjectOpen
	TokenObjectClose
	TokenArrayOpen
	TokenArrayClose
	TokenColon
	TokenComma
	TokenString
	TokenInteger
	TokenNull
	TokenTrue
	TokenFalse
	TokenEOF
)

func (k TokenKind) String() string {
	switch k {
	case TokenWhitespace:
		return "whitespace"
	case TokenObjectOpen:
		return "object-open"
	case TokenObjectClose:
		return "object-close"
	case TokenArrayOpen:
		return "array-open"
	case TokenArrayClose:
		return "array-close"
	case TokenColon:
		return "colon"
	case TokenComma:
		return "comma"
	case TokenString:
		return "string"
	case TokenInteger:
		return "integer"
	case TokenNull:
		return "null"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenEOF:
		return "eof"
	default:
		return "unknown"
	}
}

// Token retains the exact lexical bytes and their zero-based input offset.
// Raw is an owned copy; mutating the caller's input after Scan cannot alter it.
type Token struct {
	Kind   TokenKind
	Raw    []byte
	Offset int
}
