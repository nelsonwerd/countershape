package parity

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	emittermodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
)

const (
	maxCorpusBytes   = 1 << 20
	maxCorpusVectors = 512
)

var corpusOperationRoster = map[Operation]struct{}{
	CanonicalizeJSON: {}, ParseCanonicalJSON: {}, ParseManifestEnvelope: {}, ParseReadyFrame: {},
	ParseHTTPResponse: {}, ProjectCLIObservation: {}, ProjectHTTPObservation: {}, EvaluateExactPredicate: {},
	SelectDirectResult: {}, SelectOwnerEligibility: {},
}

var corpusOperationCounts = map[Operation]int{
	CanonicalizeJSON: 72, ParseCanonicalJSON: 48, ParseManifestEnvelope: 32, ParseReadyFrame: 24,
	ParseHTTPResponse: 96, ProjectCLIObservation: 48, ProjectHTTPObservation: 48, EvaluateExactPredicate: 64,
	SelectDirectResult: 48, SelectOwnerEligibility: 32,
}

var corpusRefusalRoster = map[Operation]map[string]struct{}{
	CanonicalizeJSON:       refusalSet("INVALID_BASE64", "INVALID_JSON", "INVALID_OPERATION_INPUT"),
	ParseCanonicalJSON:     refusalSet("INVALID_BASE64", "INVALID_JSON", "INVALID_OPERATION_INPUT", "NONCANONICAL_JSON"),
	ParseManifestEnvelope:  refusalSet("INVALID_BASE64", "INVALID_MANIFEST", "INVALID_OPERATION_INPUT"),
	ParseReadyFrame:        refusalSet("INVALID_BASE64", "INVALID_OPERATION_INPUT", "READINESS_FAILED"),
	ParseHTTPResponse:      refusalSet("INVALID_BASE64", "INVALID_OPERATION_INPUT", "OUTPUT_LIMIT", "RESPONSE_PARSE_FAILED"),
	ProjectCLIObservation:  refusalSet("INVALID_BASE64", "INVALID_OPERATION_INPUT", "PROJECTION_FAILED"),
	ProjectHTTPObservation: refusalSet("INVALID_BASE64", "INVALID_OPERATION_INPUT", "PROJECTION_FAILED"),
	EvaluateExactPredicate: refusalSet("INVALID_OPERATION_INPUT", "INVALID_PREDICATE"),
	SelectDirectResult:     refusalSet("INVALID_RESULT_FACTS"),
	SelectOwnerEligibility: refusalSet("INVALID_OWNER_FACTS"),
}

var directReasonRoster = refusalSet(
	"ORPHAN_RISK", "CLEANUP_FAILED", "TEARDOWN_FAILED", "OUTPUT_LIMIT", "TIMEOUT", "TRANSPORT_FAILED",
	"CAPTURE_FAILED", "RESPONSE_PARSE_FAILED", "PROJECTION_FAILED", "READINESS_FAILED", "START_FAILED",
	"ENVIRONMENT_INVALID", "FIXTURE_OVERLAY_FAILED", "SOURCE_COPY_FAILED", "EXECUTION_ROOT_FAILED",
	"SOURCE_INVENTORY_INVALID",
)

var ownerReasonRoster = refusalSet(
	"MATERIALIZATION_ERROR", "SETUP_ERROR", "START_ERROR", "READINESS_ERROR", "PROBE_TRANSPORT_ERROR",
	"TIMEOUT", "CANCELLED", "OUTPUT_LIMIT", "PROJECTION_REJECTED", "ORPHAN_RISK", "TEARDOWN_ERROR",
	"UNSUPPORTED_GIT_MODE", "MISSING_OBJECT", "BUDGET_EXHAUSTED",
)

type corpusVector struct {
	id          string
	description string
	tags        []string
	operation   Operation
	input       canon.Value
	expected    []byte
	line        []byte
}

func refusalSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func corpusPath(t testing.TB) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate the parity package")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "../../../../spec/vectors/v1/contract-parity.jsonl"))
}

func loadCorpus(t testing.TB) []corpusVector {
	t.Helper()
	exact, err := os.ReadFile(corpusPath(t))
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := parseCorpus(exact)
	if err != nil {
		t.Fatal(err)
	}
	return vectors
}

func parseCorpus(exact []byte) ([]corpusVector, error) {
	if len(exact) == 0 || len(exact) > maxCorpusBytes || exact[len(exact)-1] != '\n' || bytes.ContainsRune(exact, '\r') {
		return nil, errors.New("corpus must be nonempty, bounded, LF-only, and final-LF terminated")
	}
	lines := bytes.Split(exact[:len(exact)-1], []byte{'\n'})
	if len(lines) != maxCorpusVectors {
		return nil, fmt.Errorf("corpus has %d vectors; want exactly %d", len(lines), maxCorpusVectors)
	}
	result := make([]corpusVector, len(lines))
	ids := make(map[string]struct{}, len(lines))
	for index, line := range lines {
		if len(line) == 0 || len(line)+1 > MaxFrameBytes {
			return nil, fmt.Errorf("corpus line %d is blank or exceeds the whole-wire cap", index+1)
		}
		vector, err := parseCorpusLine(line)
		if err != nil {
			return nil, fmt.Errorf("corpus line %d: %w", index+1, err)
		}
		if _, duplicate := ids[vector.id]; duplicate {
			return nil, fmt.Errorf("corpus line %d repeats vector ID %q", index+1, vector.id)
		}
		ids[vector.id] = struct{}{}
		result[index] = vector
	}
	return result, nil
}

func parseCorpusLine(line []byte) (corpusVector, error) {
	if len(line) == 0 || len(line) > MaxFrameBodyBytes || bytes.ContainsAny(line, "\r\n") {
		return corpusVector{}, errors.New("line framing is invalid")
	}
	value, err := canon.Parse(line)
	if err != nil {
		return corpusVector{}, fmt.Errorf("line JSON: %w", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, line) {
		return corpusVector{}, errors.New("line is not exact canonical JSON")
	}
	fields, ok := exactObject(value, "description", "expected", "id", "input", "operation", "tags")
	if !ok || fields[3].Kind() != canon.KindObject {
		return corpusVector{}, errors.New("vector root roster or input object is invalid")
	}
	description, descriptionOK := fields[0].Text()
	id, idOK := fields[2].Text()
	operationText, operationOK := fields[4].Text()
	if !descriptionOK || description == "" || !idOK || !validCorpusID(id) || !operationOK {
		return corpusVector{}, errors.New("vector description, ID, or operation text is invalid")
	}
	operation := Operation(operationText)
	if _, admitted := corpusOperationRoster[operation]; !admitted {
		return corpusVector{}, errors.New("operation is outside the closed roster")
	}
	if !validCorpusInputRoster(operation, fields[3]) {
		return corpusVector{}, errors.New("input is outside the operation-owned root roster")
	}
	tags, err := parseCorpusTags(fields[5])
	if err != nil {
		return corpusVector{}, err
	}
	expected, err := parseCorpusExpected(operation, fields[3], fields[1])
	if err != nil {
		return corpusVector{}, err
	}
	vector := corpusVector{
		id: id, description: description, tags: tags, operation: operation, input: fields[3],
		expected: expected, line: append([]byte(nil), line...),
	}
	if _, err := ParseRequest(vector.requestBytes()); err != nil {
		return corpusVector{}, fmt.Errorf("driver request: %w", err)
	}
	return vector, nil
}

func validCorpusInputRoster(operation Operation, input canon.Value) bool {
	var ok bool
	switch operation {
	case CanonicalizeJSON, ParseCanonicalJSON, ParseManifestEnvelope:
		_, ok = exactObject(input, "bytes_base64")
	case ParseReadyFrame:
		_, ok = exactObject(input, "eof_observed", "frame_base64")
	case ParseHTTPResponse:
		_, ok = exactObject(input, "body_bytes", "header_bytes", "header_count", "response_base64", "status_line_bytes")
	case ProjectCLIObservation:
		_, ok = exactObject(input, "completion", "selected_fields", "stderr_base64", "stdout_base64")
	case ProjectHTTPObservation:
		_, ok = exactObject(input, "body_bytes", "header_bytes", "header_count", "response_base64", "scratch_root", "selected_fields", "status_line_bytes")
	case EvaluateExactPredicate:
		_, ok = exactObject(input, "adapter", "allowed_tuples", "decision_action", "observed_tuple", "profile_fields", "selected_fields", "stimulus_digest")
	case SelectDirectResult:
		_, ok = exactObject(input, "ineligible_reasons", "internal_failure", "predicate_match", "tamper")
	case SelectOwnerEligibility:
		_, ok = exactObject(input, "kind", "reasons")
	}
	return ok
}

func validCorpusID(value string) bool {
	if len(value) < 1 || len(value) > 96 {
		return false
	}
	first := value[0]
	if !((first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')) {
		return false
	}
	for index := 1; index < len(value); index++ {
		b := value[index]
		if b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '.' || b == '_' || b == '-' {
			continue
		}
		return false
	}
	return true
}

func parseCorpusTags(value canon.Value) ([]string, error) {
	items, ok := value.Elements()
	if !ok {
		return nil, errors.New("tags must be an array")
	}
	result := make([]string, len(items))
	for index, item := range items {
		tag, ok := item.Text()
		if !ok || tag == "" || index > 0 && bytes.Compare([]byte(result[index-1]), []byte(tag)) >= 0 {
			return nil, errors.New("tags must be nonempty, unique, and unsigned-UTF-8 sorted")
		}
		result[index] = tag
	}
	return result, nil
}

func parseCorpusExpected(operation Operation, input, expected canon.Value) ([]byte, error) {
	if fields, ok := exactObject(expected, "code", "status"); ok {
		code, codeOK := fields[0].Text()
		status, statusOK := fields[1].Text()
		if !codeOK || !statusOK || status != "REFUSED" {
			return nil, errors.New("refused expected result has invalid scalar fields")
		}
		if _, admitted := corpusRefusalRoster[operation][code]; !admitted {
			return nil, errors.New("refusal code is outside the operation-owned roster")
		}
	} else if fields, ok := exactObject(expected, "status", "value"); ok {
		status, statusOK := fields[0].Text()
		if !statusOK || status != "OK" || !validCorpusOKValue(operation, input, fields[1]) {
			return nil, errors.New("OK expected result has invalid operation-owned shape")
		}
	} else {
		return nil, errors.New("expected result is outside the closed OK/REFUSED roster")
	}
	canonical, err := expected.CanonicalChecked()
	if err != nil {
		return nil, err
	}
	if len(canonical) > MaxFrameBodyBytes {
		return nil, errors.New("corpus expected result cannot be represented by the runner")
	}
	return canonical, nil
}

func validCorpusOKValue(operation Operation, input, value canon.Value) bool {
	switch operation {
	case CanonicalizeJSON, ParseCanonicalJSON:
		fields, ok := exactObject(value, "canonical_base64")
		if !ok {
			return false
		}
		exact, ok := canonicalBase64Bytes(fields[0])
		if !ok {
			return false
		}
		parsed, err := canon.Parse(exact)
		if err != nil {
			return false
		}
		canonical, err := parsed.CanonicalChecked()
		return err == nil && bytes.Equal(canonical, exact)
	case ParseManifestEnvelope:
		fields, ok := exactObject(value, "manifest")
		if !ok || fields[0].Kind() != canon.KindObject {
			return false
		}
		exact, err := fields[0].CanonicalChecked()
		if err != nil {
			return false
		}
		_, err = emittermodel.ParseIntegrityManifestEnvelope(append(exact, '\n'))
		return err == nil
	case ParseReadyFrame:
		fields, ok := exactObject(value, "port")
		port, integer := int64(0), false
		if ok {
			port, integer = fields[0].Int64()
		}
		return integer && port >= 1 && port <= 65535
	case ParseHTTPResponse:
		return validCorpusHTTPResult(value)
	case ProjectCLIObservation, ProjectHTTPObservation:
		fields, ok := exactObject(value, "tuple")
		if !ok {
			return false
		}
		exact, err := fields[0].CanonicalChecked()
		if err != nil {
			return false
		}
		tuple, err := emittermodel.ParseExactTuple(exact)
		if err != nil {
			return false
		}
		return tupleMatchesSelectedInput(tuple, input)
	case EvaluateExactPredicate:
		fields, ok := exactObject(value, "match")
		if !ok {
			return false
		}
		_, boolean := fields[0].Boolean()
		return boolean
	case SelectDirectResult:
		return validCorpusDirectResult(value)
	case SelectOwnerEligibility:
		return validCorpusOwnerResult(value)
	default:
		return false
	}
}

func canonicalBase64Bytes(value canon.Value) ([]byte, bool) {
	text, ok := value.Text()
	if !ok {
		return nil, false
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != text {
		return nil, false
	}
	return decoded, true
}

func canonicalBase64Text(value canon.Value) bool {
	_, ok := canonicalBase64Bytes(value)
	return ok
}

func validCorpusHTTPResult(value canon.Value) bool {
	fields, ok := exactObject(value, "body_base64", "content_length", "headers", "reason", "status")
	body, bodyOK := canonicalBase64Bytes(fields[0])
	if !ok || !bodyOK {
		return false
	}
	contentLength, lengthOK := fields[1].Int64()
	headers, headersOK := fields[2].Elements()
	_, reasonOK := fields[3].Text()
	status, statusOK := fields[4].Int64()
	if !lengthOK || contentLength < 0 || int64(len(body)) != contentLength || !headersOK || !reasonOK || !statusOK || status < 200 || status > 599 {
		return false
	}
	for _, header := range headers {
		parts, exact := exactObject(header, "name", "value")
		if !exact {
			return false
		}
		name, ok := parts[0].Text()
		if !ok || !validCorpusHeaderName(name) {
			return false
		}
		headerValue, ok := parts[1].Text()
		if !ok || !validCorpusHeaderValue(headerValue) {
			return false
		}
	}
	return true
}

func validCorpusHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		b := value[index]
		if b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
			continue
		}
		return false
	}
	return true
}

func validCorpusHeaderValue(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

func tupleMatchesSelectedInput(tuple emittermodel.ExactTuple, input canon.Value) bool {
	selectedValue, present := input.LookupMember("selected_fields")
	if !present {
		return false
	}
	selected, ok := selectedValue.Elements()
	fields := tuple.Fields()
	if !ok || len(selected) != len(fields) {
		return false
	}
	for index, value := range selected {
		fieldID, ok := value.Text()
		if !ok || fieldID != fields[index].FieldID() {
			return false
		}
	}
	return true
}

func validCorpusDirectResult(value canon.Value) bool {
	fields, ok := exactObject(value, "outcome", "reason")
	if !ok {
		return false
	}
	outcome, outcomeOK := fields[0].Text()
	reason, reasonOK := fields[1].Text()
	if !outcomeOK || !reasonOK {
		return false
	}
	switch outcome {
	case "CONFORMS":
		return reason == "NONE"
	case "CONTRADICTS":
		return reason == "PREDICATE_MISMATCH"
	case "TAMPER_DETECTED":
		return reason == "COMPANION_INTEGRITY_MISMATCH"
	case "HARNESS_FAILURE":
		return reason == "INTERNAL_INVARIANT_FAILED"
	case "INELIGIBLE_EXECUTION":
		_, ok := directReasonRoster[reason]
		return ok
	default:
		return false
	}
}

func validCorpusOwnerResult(value canon.Value) bool {
	fields, ok := exactObject(value, "eligibility", "reasons")
	if !ok {
		return false
	}
	eligibility, eligibilityOK := fields[0].Text()
	reasons, reasonsOK := fields[1].Elements()
	if !eligibilityOK || !reasonsOK {
		return false
	}
	if eligibility == "ELIGIBLE" {
		return len(reasons) == 0
	}
	if eligibility != "INELIGIBLE" || len(reasons) < 1 || len(reasons) > 3 {
		return false
	}
	seen := make(map[string]struct{}, len(reasons))
	for index, item := range reasons {
		reason, ok := item.Text()
		if !ok {
			return false
		}
		if _, admitted := ownerReasonRoster[reason]; !admitted {
			return false
		}
		if _, duplicate := seen[reason]; duplicate {
			return false
		}
		seen[reason] = struct{}{}
		if index > 0 && reason != "TEARDOWN_ERROR" && reason != "ORPHAN_RISK" {
			return false
		}
	}
	return true
}

func (v corpusVector) requestBytes() []byte {
	operation, _ := canon.String(string(v.operation))
	request, err := canon.Object(
		canon.Member{Name: "input", Value: v.input},
		canon.Member{Name: "operation", Value: operation},
	)
	if err != nil {
		return nil
	}
	exact, _ := request.CanonicalChecked()
	return exact
}

func evaluateCorpusGo(t testing.TB, vector corpusVector) []byte {
	t.Helper()
	request, err := ParseRequest(vector.requestBytes())
	if err != nil {
		t.Fatal(err)
	}
	return Evaluate(request).CanonicalBytes()
}

func evaluateCorpusNode(t *testing.T, vector corpusVector) []byte {
	t.Helper()
	frame := []byte(runNodeParity(t, vector.requestBytes()))
	if len(frame) < 2 || len(frame) > MaxFrameBytes || frame[len(frame)-1] != '\n' {
		t.Fatalf("Node response is not one bounded LF frame: %q", frame)
	}
	body := frame[:len(frame)-1]
	if bytes.ContainsAny(body, "\r\n") {
		t.Fatalf("Node response body contains an embedded frame separator: %q", body)
	}
	value, err := canon.Parse(body)
	if err != nil {
		t.Fatalf("Node response body is not JSON: %v", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, body) {
		t.Fatalf("Node response body is not exact canonical JSON: %v", err)
	}
	return append([]byte(nil), body...)
}

func TestContractParityCorpus(t *testing.T) {
	vectors := loadCorpus(t)
	covered := make(map[Operation]int, len(corpusOperationRoster))
	coveredTags := make(map[string]bool)
	duplicateRequests := 0
	firstExpectedByRequest := make(map[string][]byte)
	for _, vector := range vectors {
		requestKey := string(vector.requestBytes())
		if previous, duplicate := firstExpectedByRequest[requestKey]; duplicate {
			duplicateRequests++
			if !bytes.Equal(previous, vector.expected) {
				t.Fatalf("duplicate exact evaluator input %s has different literal expectations", vector.id)
			}
		} else {
			firstExpectedByRequest[requestKey] = append([]byte(nil), vector.expected...)
		}
	}
	for _, vector := range vectors {
		vector := vector
		t.Run(vector.id, func(t *testing.T) {
			goActual := evaluateCorpusGo(t, vector)
			if !bytes.Equal(goActual, vector.expected) {
				t.Fatalf("Go actual:\n%s\nexpected:\n%s", goActual, vector.expected)
			}
			result := Result{canonical: goActual}
			frame, err := result.FrameBytes()
			if err != nil || !bytes.Equal(frame, append(append([]byte(nil), vector.expected...), '\n')) {
				t.Fatalf("Go frame = %q, %v", frame, err)
			}
			nodeActual := evaluateCorpusNode(t, vector)
			if !bytes.Equal(nodeActual, vector.expected) {
				t.Fatalf("Node actual:\n%s\nexpected:\n%s", nodeActual, vector.expected)
			}
		})
		covered[vector.operation]++
		for _, tag := range vector.tags {
			coveredTags[tag] = true
		}
	}
	for operation, want := range corpusOperationCounts {
		if got := covered[operation]; got != want {
			t.Errorf("operation %s has %d normative vectors; want %d", operation, got, want)
		}
	}
	for _, tag := range []string{
		"boundary", "canonical-json", "cli-projection", "direct-result", "http-projection", "http-wire",
		"manifest", "negative", "oracle-duplicate", "owner-eligibility", "positive", "predicate", "readiness",
	} {
		if !coveredTags[tag] {
			t.Errorf("required coverage tag %q is absent", tag)
		}
	}
	if duplicateRequests == 0 {
		t.Error("corpus has no duplicate exact evaluator input under different private labels")
	}
}

func TestContractParityCorpusIsOrderAndOracleIndependent(t *testing.T) {
	vectors := loadCorpus(t)
	original := make(map[string][]byte, len(vectors))
	for _, vector := range vectors {
		actual := evaluateCorpusGo(t, vector)
		if !bytes.Equal(actual, vector.expected) {
			t.Fatalf("baseline vector %s failed", vector.id)
		}
		original[vector.id] = actual
	}
	shuffled := append([]corpusVector(nil), vectors...)
	sort.Slice(shuffled, func(left, right int) bool { return shuffled[left].id > shuffled[right].id })
	for _, vector := range shuffled {
		if actual := evaluateCorpusGo(t, vector); !bytes.Equal(actual, original[vector.id]) {
			t.Fatalf("Go evaluator changed under shuffled order for %s", vector.id)
		}
		if actual := evaluateCorpusNode(t, vector); !bytes.Equal(actual, original[vector.id]) {
			t.Fatalf("Node evaluator changed under shuffled order for %s", vector.id)
		}
	}

	clone := vectors[0]
	clone.id = "poisoned-private-label"
	clone.description = "driver-only metadata changed"
	clone.tags = []string{"negative", "oracle-poison"}
	clone.expected = poisonedCorpusExpected(clone.expected)
	if !bytes.Equal(clone.requestBytes(), vectors[0].requestBytes()) {
		t.Fatal("driver-only metadata leaked into evaluator input")
	}
	goActual := evaluateCorpusGo(t, clone)
	nodeActual := evaluateCorpusNode(t, clone)
	if !bytes.Equal(goActual, original[vectors[0].id]) || !bytes.Equal(nodeActual, original[vectors[0].id]) {
		t.Fatal("poisoned expected metadata changed evaluator output")
	}
	if bytes.Equal(goActual, clone.expected) || bytes.Equal(nodeActual, clone.expected) {
		t.Fatal("poisoned expected metadata did not make the driver comparison fail")
	}
	if _, answered := expectedEchoOracleMutant(clone.requestBytes()); answered {
		t.Fatal("expected-echo mutant survived the stripped request interface")
	}
	if _, answered := caseIDOracleMutant(clone.requestBytes(), map[string][]byte{clone.id: clone.expected}); answered {
		t.Fatal("case-ID mutant survived the stripped request interface")
	}
}

func TestContractParityOracleLeakMutantsAreKilledByRequestRoster(t *testing.T) {
	vector := loadCorpus(t)[0]
	poisoned := poisonedCorpusExpected(vector.expected)
	wantRefusal := "{\"code\":\"INVALID_OPERATION_INPUT\",\"status\":\"REFUSED\"}\n"

	expectedLeak := corpusRequestWithPrivateLeak(t, vector, "expected", poisoned)
	if actual, answered := expectedEchoOracleMutant(expectedLeak); !answered || !bytes.Equal(actual, poisoned) {
		t.Fatal("expected-echo mutant positive control cannot answer from an expected leak")
	}
	if _, err := ParseRequest(expectedLeak); err == nil {
		t.Fatal("Go request roster admitted the expected-echo mutant's oracle leak")
	}
	if actual := runNodeParity(t, expectedLeak); actual != wantRefusal {
		t.Fatalf("Node request roster returned %q for expected leak; want %q", actual, wantRefusal)
	}

	privateID := "oracle-private-case"
	caseLeak := corpusRequestWithPrivateLeak(t, vector, "id", []byte(privateID))
	answers := map[string][]byte{privateID: poisoned}
	if actual, answered := caseIDOracleMutant(caseLeak, answers); !answered || !bytes.Equal(actual, poisoned) {
		t.Fatal("case-ID mutant positive control cannot answer from a private ID leak")
	}
	if _, err := ParseRequest(caseLeak); err == nil {
		t.Fatal("Go request roster admitted the case-ID mutant's oracle leak")
	}
	if actual := runNodeParity(t, caseLeak); actual != wantRefusal {
		t.Fatalf("Node request roster returned %q for case-ID leak; want %q", actual, wantRefusal)
	}
}

func poisonedCorpusExpected(original []byte) []byte {
	if bytes.Equal(original, []byte(`{"code":"INVALID_OPERATION_INPUT","status":"REFUSED"}`)) {
		return []byte(`{"status":"OK","value":{"match":false}}`)
	}
	return []byte(`{"code":"INVALID_OPERATION_INPUT","status":"REFUSED"}`)
}

func expectedEchoOracleMutant(request []byte) ([]byte, bool) {
	value, err := canon.Parse(request)
	if err != nil {
		return nil, false
	}
	expected, present := value.LookupMember("expected")
	if !present {
		return nil, false
	}
	exact, err := expected.CanonicalChecked()
	return exact, err == nil
}

func caseIDOracleMutant(request []byte, answers map[string][]byte) ([]byte, bool) {
	value, err := canon.Parse(request)
	if err != nil {
		return nil, false
	}
	idValue, present := value.LookupMember("id")
	if !present {
		return nil, false
	}
	id, ok := idValue.Text()
	if !ok {
		return nil, false
	}
	answer, present := answers[id]
	return append([]byte(nil), answer...), present
}

func corpusRequestWithPrivateLeak(t testing.TB, vector corpusVector, name string, exact []byte) []byte {
	t.Helper()
	operation, err := canon.String(string(vector.operation))
	if err != nil {
		t.Fatal(err)
	}
	var leaked canon.Value
	switch name {
	case "expected":
		leaked, err = canon.Parse(exact)
	case "id":
		leaked, err = canon.String(string(exact))
	default:
		t.Fatalf("unsupported private leak %q", name)
	}
	if err != nil {
		t.Fatal(err)
	}
	members := []canon.Member{{Name: name, Value: leaked}, {Name: "input", Value: vector.input}, {Name: "operation", Value: operation}}
	root, err := canon.Object(members...)
	if err != nil {
		t.Fatal(err)
	}
	request, err := root.CanonicalChecked()
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestContractParityCorpusParserRejectsEnvelopeAliases(t *testing.T) {
	vectors := loadCorpus(t)
	line := vectors[0].line
	mutations := map[string][]byte{
		"blank":        nil,
		"bom":          append([]byte{0xef, 0xbb, 0xbf}, line...),
		"crlf":         append(append([]byte(nil), line...), '\r'),
		"noncanonical": append([]byte(" "), line...),
		"unknown root": bytes.Replace(line, []byte(`"tags":`), []byte(`"unknown":0,"tags":`), 1),
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCorpusLine(mutation); err == nil {
				t.Fatal("mutation was accepted")
			}
		})
	}
	if _, err := parseCorpus(append(append([]byte(nil), line...), '\n', '\n')); err == nil {
		t.Fatal("blank corpus line was accepted")
	}
	if _, err := parseCorpus(append([]byte(nil), line...)); err == nil {
		t.Fatal("missing final LF was accepted")
	}
}

func FuzzParseContractParityCorpusLine(f *testing.F) {
	exact, err := os.ReadFile(corpusPath(f))
	if err != nil {
		f.Fatal(err)
	}
	for _, line := range bytes.Split(bytes.TrimSuffix(exact, []byte{'\n'}), []byte{'\n'}) {
		f.Add(append([]byte(nil), line...))
	}
	f.Add([]byte(`{"description":"x","expected":{"code":"INVALID_JSON","status":"REFUSED"},"id":"x","input":{},"operation":"CANONICALIZE_JSON","tags":[]}`))
	f.Fuzz(func(t *testing.T, line []byte) {
		vector, err := parseCorpusLine(line)
		if err != nil {
			return
		}
		if !bytes.Equal(vector.line, line) {
			t.Fatal("accepted line was not retained exactly")
		}
		if _, err := ParseRequest(vector.requestBytes()); err != nil {
			t.Fatalf("accepted vector produced invalid stripped request: %v", err)
		}
	})
}
