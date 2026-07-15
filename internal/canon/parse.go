package canon

import (
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

const maxNestingDepth = 256

// Parse creates an immutable Value only after strict lossless tokenization has
// succeeded. It never calls encoding/json.
func Parse(input []byte) (Value, error) {
	tokens, err := Scan(input)
	if err != nil {
		return Value{}, err
	}
	parser := tokenParser{tokens: tokens}
	value, err := parser.parseValue(0)
	if err != nil {
		return Value{}, err
	}
	parser.skipWhitespace()
	if parser.current().Kind != TokenEOF {
		return Value{}, refusal(CodeTrailingData, parser.current().Offset, "data follows the first complete value")
	}
	return value, nil
}

type tokenParser struct {
	tokens []Token
	index  int
}

func (p *tokenParser) current() Token {
	if p.index >= len(p.tokens) {
		return Token{Kind: TokenEOF}
	}
	return p.tokens[p.index]
}

func (p *tokenParser) skipWhitespace() {
	for p.current().Kind == TokenWhitespace {
		p.index++
	}
}

func (p *tokenParser) take(kind TokenKind) (Token, error) {
	p.skipWhitespace()
	token := p.current()
	if token.Kind != kind {
		return Token{}, refusal(CodeUnexpectedToken, token.Offset, "unexpected token while parsing strict JSON")
	}
	p.index++
	return token, nil
}

func (p *tokenParser) parseValue(depth int) (Value, error) {
	if depth > maxNestingDepth {
		return Value{}, refusal(CodeDepthExceeded, p.current().Offset, "maximum nesting depth exceeded")
	}
	p.skipWhitespace()
	token := p.current()
	switch token.Kind {
	case TokenNull:
		p.index++
		return Null(), nil
	case TokenTrue:
		p.index++
		return Bool(true), nil
	case TokenFalse:
		p.index++
		return Bool(false), nil
	case TokenInteger:
		p.index++
		integer, err := strconv.ParseInt(string(token.Raw), 10, 64)
		if err != nil {
			return Value{}, refusal(CodeUnsafeInteger, token.Offset, "integer exceeds the exact interoperability range")
		}
		return Value{kind: KindInteger, integer: integer}, nil
	case TokenString:
		p.index++
		decoded, err := decodeStringToken(token)
		if err != nil {
			return Value{}, err
		}
		return Value{kind: KindString, text: decoded}, nil
	case TokenArrayOpen:
		return p.parseArray(depth + 1)
	case TokenObjectOpen:
		return p.parseObject(depth + 1)
	case TokenEOF:
		return Value{}, refusal(CodeUnexpectedEOF, token.Offset, "expected a JSON value")
	default:
		return Value{}, refusal(CodeUnexpectedToken, token.Offset, "token cannot begin a JSON value")
	}
}

func (p *tokenParser) parseArray(depth int) (Value, error) {
	if _, err := p.take(TokenArrayOpen); err != nil {
		return Value{}, err
	}
	p.skipWhitespace()
	if p.current().Kind == TokenArrayClose {
		p.index++
		return Array()
	}

	values := make([]Value, 0, 4)
	for {
		if len(values) >= MaxContainerMembers {
			return Value{}, refusal(CodeMemberLimit, p.current().Offset, "array exceeds the v1 member ceiling")
		}
		value, err := p.parseValue(depth)
		if err != nil {
			return Value{}, err
		}
		values = append(values, value)
		p.skipWhitespace()
		switch p.current().Kind {
		case TokenComma:
			p.index++
		case TokenArrayClose:
			p.index++
			return Array(values...)
		default:
			return Value{}, refusal(CodeUnexpectedToken, p.current().Offset, "array requires a comma or closing bracket")
		}
	}
}

func (p *tokenParser) parseObject(depth int) (Value, error) {
	if _, err := p.take(TokenObjectOpen); err != nil {
		return Value{}, err
	}
	p.skipWhitespace()
	if p.current().Kind == TokenObjectClose {
		p.index++
		return Object()
	}

	members := make([]Member, 0, 4)
	seen := make(map[string]bool, 4)
	for {
		if len(members) >= MaxContainerMembers {
			return Value{}, refusal(CodeMemberLimit, p.current().Offset, "object exceeds the v1 member ceiling")
		}
		nameToken, err := p.take(TokenString)
		if err != nil {
			return Value{}, err
		}
		name, err := decodeStringToken(nameToken)
		if err != nil {
			return Value{}, err
		}
		if err := admitObjectName(seen, name, nameToken.Offset); err != nil {
			return Value{}, err
		}
		if _, err := p.take(TokenColon); err != nil {
			return Value{}, err
		}
		value, err := p.parseValue(depth)
		if err != nil {
			return Value{}, err
		}
		members = append(members, Member{Name: name, Value: value})

		p.skipWhitespace()
		switch p.current().Kind {
		case TokenComma:
			p.index++
		case TokenObjectClose:
			p.index++
			return orderedObject(members), nil
		default:
			return Value{}, refusal(CodeUnexpectedToken, p.current().Offset, "object requires a comma or closing brace")
		}
	}
}

func decodeStringToken(token Token) (string, error) {
	raw := token.Raw
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", refusal(CodeInvalidValue, token.Offset, "string token is malformed")
	}

	decoded := make([]byte, 0, len(raw)-2)
	for offset := 1; offset < len(raw)-1; {
		if raw[offset] != '\\' {
			if raw[offset] < utf8.RuneSelf {
				decoded = append(decoded, raw[offset])
				offset++
				continue
			}
			_, size := utf8.DecodeRune(raw[offset:])
			decoded = append(decoded, raw[offset:offset+size]...)
			offset += size
			continue
		}

		escapeOffset := offset
		escape := raw[offset+1]
		switch escape {
		case '"', '\\', '/':
			decoded = append(decoded, escape)
			offset += 2
		case 'b':
			decoded = append(decoded, '\b')
			offset += 2
		case 'f':
			decoded = append(decoded, '\f')
			offset += 2
		case 'n':
			decoded = append(decoded, '\n')
			offset += 2
		case 'r':
			decoded = append(decoded, '\r')
			offset += 2
		case 't':
			decoded = append(decoded, '\t')
			offset += 2
		case 'u':
			first := decodeHexQuad(raw[offset+2 : offset+6])
			offset += 6
			runeValue := rune(first)
			if utf16.IsSurrogate(runeValue) {
				if first < 0xd800 || first > 0xdbff {
					return "", refusal(CodeLoneSurrogate, token.Offset+escapeOffset, "low surrogate has no leading high surrogate")
				}
				if offset+6 > len(raw)-1 || raw[offset] != '\\' || raw[offset+1] != 'u' {
					return "", refusal(CodeLoneSurrogate, token.Offset+escapeOffset, "high surrogate has no trailing low surrogate")
				}
				second := decodeHexQuad(raw[offset+2 : offset+6])
				if second < 0xdc00 || second > 0xdfff {
					return "", refusal(CodeLoneSurrogate, token.Offset+offset, "high surrogate is followed by a non-low surrogate")
				}
				runeValue = utf16.DecodeRune(rune(first), rune(second))
				offset += 6
			}
			decoded = utf8.AppendRune(decoded, runeValue)
		default:
			return "", refusal(CodeInvalidEscape, token.Offset+offset+1, "unsupported string escape")
		}
	}
	return string(decoded), nil
}

func decodeHexQuad(raw []byte) uint16 {
	var value uint16
	for _, b := range raw {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value |= uint16(b - '0')
		case b >= 'a' && b <= 'f':
			value |= uint16(b-'a') + 10
		case b >= 'A' && b <= 'F':
			value |= uint16(b-'A') + 10
		}
	}
	return value
}
