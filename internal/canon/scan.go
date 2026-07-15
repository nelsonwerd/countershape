package canon

import (
	"bytes"
	"unicode/utf8"
)

// MaxSafeInteger and MinSafeInteger define the closed v1 integer range. The
// range is deliberately shared with exact JavaScript integers because the U6
// standalone profile must not coerce identity-bearing numbers.
const (
	MaxSafeInteger int64 = 1<<53 - 1
	MinSafeInteger int64 = -MaxSafeInteger
)

var maxSafeIntegerDigits = []byte("9007199254740991")

// Scan tokenizes strict Countershape JSON without performing any ordinary JSON
// decoding. Every non-EOF token owns its exact source bytes.
func Scan(input []byte) ([]Token, error) {
	if len(input) > MaxInputBytes {
		return nil, refusal(CodeInputLimit, MaxInputBytes, "input exceeds the v1 byte ceiling")
	}
	if offset := firstInvalidUTF8(input); offset >= 0 {
		return nil, refusal(CodeInvalidUTF8, offset, "input is not valid UTF-8")
	}

	tokens := make([]Token, 0, 16)
	for offset := 0; offset < len(input); {
		start := offset
		// Reserve one slot for EOF so the published ceiling describes the
		// complete lossless token stream, not an internal implementation detail.
		if len(tokens) >= MaxTokenCount-1 {
			return nil, refusal(CodeTokenLimit, start, "input exceeds the v1 token ceiling")
		}
		switch input[offset] {
		case ' ', '\t', '\n', '\r':
			offset++
			for offset < len(input) && isWhitespace(input[offset]) {
				offset++
			}
			tokens = append(tokens, ownedToken(TokenWhitespace, input[start:offset], start))
		case '{':
			offset++
			tokens = append(tokens, ownedToken(TokenObjectOpen, input[start:offset], start))
		case '}':
			offset++
			tokens = append(tokens, ownedToken(TokenObjectClose, input[start:offset], start))
		case '[':
			offset++
			tokens = append(tokens, ownedToken(TokenArrayOpen, input[start:offset], start))
		case ']':
			offset++
			tokens = append(tokens, ownedToken(TokenArrayClose, input[start:offset], start))
		case ':':
			offset++
			tokens = append(tokens, ownedToken(TokenColon, input[start:offset], start))
		case ',':
			offset++
			tokens = append(tokens, ownedToken(TokenComma, input[start:offset], start))
		case '"':
			end, err := scanString(input, start)
			if err != nil {
				return nil, err
			}
			offset = end
			tokens = append(tokens, ownedToken(TokenString, input[start:offset], start))
		case '-', '+', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			offset = scalarEnd(input, start)
			raw := input[start:offset]
			if err := validateIntegerLexeme(raw, start); err != nil {
				return nil, err
			}
			tokens = append(tokens, ownedToken(TokenInteger, raw, start))
		case 'n':
			end := scalarEnd(input, start)
			if !bytes.Equal(input[start:end], []byte("null")) {
				return nil, refusal(CodeInvalidLiteral, start, "unsupported literal")
			}
			offset = end
			tokens = append(tokens, ownedToken(TokenNull, input[start:offset], start))
		case 't':
			end := scalarEnd(input, start)
			if !bytes.Equal(input[start:end], []byte("true")) {
				return nil, refusal(CodeInvalidLiteral, start, "unsupported literal")
			}
			offset = end
			tokens = append(tokens, ownedToken(TokenTrue, input[start:offset], start))
		case 'f':
			end := scalarEnd(input, start)
			if !bytes.Equal(input[start:end], []byte("false")) {
				return nil, refusal(CodeInvalidLiteral, start, "unsupported literal")
			}
			offset = end
			tokens = append(tokens, ownedToken(TokenFalse, input[start:offset], start))
		case 'N', 'I':
			return nil, refusal(CodeUnsupportedNumber, start, "non-finite numbers are unsupported")
		default:
			return nil, refusal(CodeUnexpectedByte, start, "byte cannot begin a strict JSON token")
		}
	}

	tokens = append(tokens, Token{Kind: TokenEOF, Offset: len(input)})
	return tokens, nil
}

func ownedToken(kind TokenKind, raw []byte, offset int) Token {
	return Token{Kind: kind, Raw: append([]byte(nil), raw...), Offset: offset}
}

func firstInvalidUTF8(input []byte) int {
	for offset := 0; offset < len(input); {
		r, size := utf8.DecodeRune(input[offset:])
		if r == utf8.RuneError && size == 1 {
			return offset
		}
		offset += size
	}
	return -1
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isDelimiter(b byte) bool {
	return isWhitespace(b) || b == '{' || b == '}' || b == '[' || b == ']' || b == ':' || b == ','
}

func scalarEnd(input []byte, start int) int {
	offset := start
	for offset < len(input) && !isDelimiter(input[offset]) {
		offset++
	}
	return offset
}

func scanString(input []byte, start int) (int, error) {
	for offset := start + 1; offset < len(input); {
		b := input[offset]
		switch {
		case b == '"':
			return offset + 1, nil
		case b < 0x20:
			return 0, refusal(CodeControlCharacter, offset, "unescaped control character in string")
		case b == '\\':
			if offset+1 >= len(input) {
				return 0, refusal(CodeUnexpectedEOF, offset, "truncated string escape")
			}
			escape := input[offset+1]
			switch escape {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				offset += 2
			case 'u':
				if offset+6 > len(input) {
					return 0, refusal(CodeUnexpectedEOF, offset, "truncated Unicode escape")
				}
				for index := offset + 2; index < offset+6; index++ {
					if !isHex(input[index]) {
						return 0, refusal(CodeInvalidEscape, index, "Unicode escape contains a non-hex byte")
					}
				}
				first := decodeHexQuad(input[offset+2 : offset+6])
				switch {
				case first >= 0xd800 && first <= 0xdbff:
					secondOffset := offset + 6
					if secondOffset+6 > len(input) || input[secondOffset] != '\\' || input[secondOffset+1] != 'u' {
						return 0, refusal(CodeLoneSurrogate, offset, "high surrogate has no trailing low surrogate")
					}
					for index := secondOffset + 2; index < secondOffset+6; index++ {
						if !isHex(input[index]) {
							return 0, refusal(CodeInvalidEscape, index, "Unicode escape contains a non-hex byte")
						}
					}
					second := decodeHexQuad(input[secondOffset+2 : secondOffset+6])
					if second < 0xdc00 || second > 0xdfff {
						return 0, refusal(CodeLoneSurrogate, secondOffset, "high surrogate is followed by a non-low surrogate")
					}
					offset += 12
				case first >= 0xdc00 && first <= 0xdfff:
					return 0, refusal(CodeLoneSurrogate, offset, "low surrogate has no leading high surrogate")
				default:
					offset += 6
				}
			default:
				return 0, refusal(CodeInvalidEscape, offset+1, "unsupported string escape")
			}
		case b < utf8.RuneSelf:
			offset++
		default:
			_, size := utf8.DecodeRune(input[offset:])
			offset += size
		}
	}
	return 0, refusal(CodeUnexpectedEOF, len(input), "unterminated string")
}

func isHex(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

func validateIntegerLexeme(raw []byte, inputOffset int) error {
	if len(raw) == 0 {
		return refusal(CodeUnsupportedNumber, inputOffset, "empty number")
	}
	if bytes.Equal(raw, []byte("-0")) { // MUTANT_U1_CANON_ACCEPT_NEGATIVE_ZERO
		return refusal(CodeNegativeZero, inputOffset, "negative zero is not identity-safe")
	}

	digitStart := 0
	if raw[0] == '-' {
		digitStart = 1
	} else if raw[0] == '+' {
		return refusal(CodeUnsupportedNumber, inputOffset, "leading plus is unsupported")
	}
	if digitStart == len(raw) {
		return refusal(CodeUnsupportedNumber, inputOffset, "sign is not an integer")
	}
	if raw[digitStart] == '0' && len(raw)-digitStart > 1 {
		return refusal(CodeUnsupportedNumber, inputOffset+digitStart+1, "leading zero is unsupported")
	}
	for index := digitStart; index < len(raw); index++ {
		if raw[index] < '0' || raw[index] > '9' {
			return refusal(CodeUnsupportedNumber, inputOffset+index, "only canonical base-10 integers are supported")
		}
	}

	digits := raw[digitStart:]
	if len(digits) > len(maxSafeIntegerDigits) ||
		(len(digits) == len(maxSafeIntegerDigits) && bytes.Compare(digits, maxSafeIntegerDigits) > 0) {
		return refusal(CodeUnsafeInteger, inputOffset, "integer exceeds the exact interoperability range")
	}
	return nil
}
