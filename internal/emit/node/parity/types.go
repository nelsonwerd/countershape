// Package parity owns the input-only, pure Go side of the shared Go/Node
// semantic corpus. It has no vector labels, expected values, filesystem path,
// process spawning, or execution authority.
package parity

import (
	"bytes"
	"encoding/base64"
	"errors"

	"github.com/nelsonwerd/countershape/internal/canon"
)

const (
	// MaxFrameBytes includes the one required terminal LF. The semantic body is
	// deliberately one byte smaller so framing cannot be confused with an
	// operation-owned output limit.
	MaxFrameBytes     = 64 << 10
	MaxFrameBodyBytes = MaxFrameBytes - 1
)

var errFrameBodySize = errors.New("parity frame body is outside the closed wire limit")

type Operation string

const (
	CanonicalizeJSON       Operation = "CANONICALIZE_JSON"
	ParseCanonicalJSON     Operation = "PARSE_CANONICAL_JSON"
	ParseManifestEnvelope  Operation = "PARSE_MANIFEST_ENVELOPE"
	ParseReadyFrame        Operation = "PARSE_READY_FRAME"
	ParseHTTPResponse      Operation = "PARSE_HTTP_RESPONSE"
	ProjectCLIObservation  Operation = "PROJECT_CLI_OBSERVATION"
	ProjectHTTPObservation Operation = "PROJECT_HTTP_OBSERVATION"
	EvaluateExactPredicate Operation = "EVALUATE_EXACT_PREDICATE"
	SelectDirectResult     Operation = "SELECT_DIRECT_RESULT"
	SelectOwnerEligibility Operation = "SELECT_OWNER_ELIGIBILITY"
)

type Request struct {
	operation Operation
	input     canon.Value
	canonical []byte
}

func ParseRequest(exact []byte) (Request, error) {
	if len(exact) == 0 || len(exact) > MaxFrameBodyBytes {
		return Request{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return Request{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return Request{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	fields, ok := exactObject(value, "input", "operation")
	if !ok || fields[0].Kind() != canon.KindObject {
		return Request{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	operation, ok := fields[1].Text()
	if !ok {
		return Request{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	return Request{
		operation: Operation(operation), input: fields[0], canonical: append([]byte(nil), canonical...),
	}, nil
}

func (r Request) Operation() Operation   { return r.operation }
func (r Request) CanonicalBytes() []byte { return append([]byte(nil), r.canonical...) }
func (r Request) valid() bool {
	reparsed, err := ParseRequest(r.canonical)
	return err == nil && reparsed.operation == r.operation
}

type Result struct {
	canonical []byte
}

func (r Result) CanonicalBytes() []byte { return append([]byte(nil), r.canonical...) }

// FrameBytes applies only the subprocess transport envelope. Evaluate remains
// a pure semantic operation even when its exact result cannot fit on the wire.
func (r Result) FrameBytes() ([]byte, error) {
	if len(r.canonical) == 0 || len(r.canonical) > MaxFrameBodyBytes {
		return nil, errFrameBodySize
	}
	frame := make([]byte, len(r.canonical)+1)
	copy(frame, r.canonical)
	frame[len(r.canonical)] = '\n'
	return frame, nil
}

type evalError struct{ code string }

func (e *evalError) Error() string        { return e.code }
func evaluationRefusal(code string) error { return &evalError{code: code} }

func Evaluate(request Request) Result {
	if !request.valid() {
		return refusedResult("INVALID_OPERATION_INPUT")
	}
	value, err := evaluate(request.operation, request.input)
	if err != nil {
		if refusal, ok := err.(*evalError); ok {
			return refusedResult(refusal.code)
		}
		return refusedResult("INVALID_OPERATION_INPUT")
	}
	return okResult(value)
}

func okResult(value canon.Value) Result {
	status, _ := canon.String("OK")
	root, err := canon.Object(
		canon.Member{Name: "status", Value: status},
		canon.Member{Name: "value", Value: value},
	)
	if err != nil {
		return refusedResult("INVALID_OPERATION_INPUT")
	}
	exact, err := root.CanonicalChecked()
	if err != nil {
		return refusedResult("INVALID_OPERATION_INPUT")
	}
	return Result{canonical: exact}
}

func refusedResult(code string) Result {
	codeValue, _ := canon.String(code)
	status, _ := canon.String("REFUSED")
	root, _ := canon.Object(
		canon.Member{Name: "code", Value: codeValue},
		canon.Member{Name: "status", Value: status},
	)
	exact, _ := root.CanonicalChecked()
	return Result{canonical: exact}
}

func exactObject(value canon.Value, names ...string) ([]canon.Value, bool) {
	members, ok := value.Members()
	if !ok || len(members) != len(names) {
		return nil, false
	}
	values := make([]canon.Value, len(names))
	for index, name := range names {
		if members[index].Name != name {
			return nil, false
		}
		values[index] = members[index].Value
	}
	return values, true
}

func exactText(value canon.Value) (string, bool)         { return value.Text() }
func exactBool(value canon.Value) (bool, bool)           { return value.Boolean() }
func exactInt(value canon.Value) (int64, bool)           { return value.Int64() }
func exactArray(value canon.Value) ([]canon.Value, bool) { return value.Elements() }

func strictBase64Value(value canon.Value) ([]byte, error) {
	encoded, ok := exactText(value)
	if !ok || len(encoded) > MaxFrameBodyBytes {
		return nil, evaluationRefusal("INVALID_BASE64")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != encoded {
		return nil, evaluationRefusal("INVALID_BASE64")
	}
	return decoded, nil
}

func textValue(value string) (canon.Value, error) {
	result, err := canon.String(value)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	return result, nil
}

func integerValue(value int64) (canon.Value, error) {
	result, err := canon.Integer(value)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	return result, nil
}

func objectValue(members ...canon.Member) (canon.Value, error) {
	result, err := canon.Object(members...)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	return result, nil
}

func arrayValue(values ...canon.Value) (canon.Value, error) {
	result, err := canon.Array(values...)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	return result, nil
}
