package parity

import (
	"bytes"
	"encoding/base64"
	"path"
	"strings"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	emittermodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/observe/eligibilitycore"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

var cliFieldIDs = []string{
	string(countercli.CLIFieldCompletionKind),
	string(countercli.CLIFieldExitCode),
	string(countercli.CLIFieldExitSignal),
	string(countercli.CLIFieldStdoutBytes),
	string(countercli.CLIFieldStderrText),
	string(countercli.CLIFieldStdoutJSONMode),
	string(countercli.CLIFieldStdoutJSONSource),
}

var httpFieldIDs = []string{
	string(counterhttp.HTTPFieldStatus),
	string(counterhttp.HTTPFieldContentType),
	string(counterhttp.HTTPFieldBodyKind),
	string(counterhttp.HTTPFieldBodyMetadata),
}

func evaluate(operation Operation, input canon.Value) (canon.Value, error) {
	switch operation {
	case CanonicalizeJSON:
		return evaluateCanonicalize(input, false)
	case ParseCanonicalJSON:
		return evaluateCanonicalize(input, true)
	case ParseManifestEnvelope:
		return evaluateManifest(input)
	case ParseReadyFrame:
		return evaluateReady(input)
	case ParseHTTPResponse:
		return evaluateHTTPParse(input)
	case ProjectCLIObservation:
		return evaluateCLIProjection(input)
	case ProjectHTTPObservation:
		return evaluateHTTPProjection(input)
	case EvaluateExactPredicate:
		return evaluatePredicate(input)
	case SelectDirectResult:
		return evaluateDirectResult(input)
	case SelectOwnerEligibility:
		return evaluateOwnerEligibility(input)
	default:
		return canon.Value{}, evaluationRefusal("UNSUPPORTED_OPERATION")
	}
}

func evaluateCanonicalize(input canon.Value, requireExact bool) (canon.Value, error) {
	fields, ok := exactObject(input, "bytes_base64")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	exact, err := strictBase64Value(fields[0])
	if err != nil {
		return canon.Value{}, err
	}
	canonical, err := canon.Canonicalize(exact)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_JSON")
	}
	if requireExact && !bytes.Equal(canonical, exact) {
		return canon.Value{}, evaluationRefusal("NONCANONICAL_JSON")
	}
	encoded, _ := textValue(base64.StdEncoding.EncodeToString(canonical))
	return objectValue(canon.Member{Name: "canonical_base64", Value: encoded})
}

func evaluateManifest(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "bytes_base64")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	exact, err := strictBase64Value(fields[0])
	if err != nil {
		return canon.Value{}, err
	}
	manifest, err := emittermodel.ParseIntegrityManifestEnvelope(exact)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_MANIFEST")
	}
	return objectValue(canon.Member{Name: "manifest", Value: manifest})
}

func evaluateReady(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "eof_observed", "frame_base64")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	eof, ok := exactBool(fields[0])
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	frameBytes, err := strictBase64Value(fields[1])
	if err != nil {
		return canon.Value{}, err
	}
	if !eof {
		return canon.Value{}, evaluationRefusal("READINESS_FAILED")
	}
	frame, err := httpmodel.ParseHTTPReadyPortFrame(frameBytes)
	if err != nil {
		return canon.Value{}, evaluationRefusal("READINESS_FAILED")
	}
	port, _ := integerValue(int64(frame.Port()))
	return objectValue(canon.Member{Name: "port", Value: port})
}

type httpOperationInput struct {
	policy httpmodel.HTTPCapturePolicy
	raw    []byte
}

func parseHTTPOperationInput(fields []canon.Value) (httpOperationInput, error) {
	body, bodyOK := exactInt(fields[0])
	headerBytes, headerBytesOK := exactInt(fields[1])
	headerCount, headerCountOK := exactInt(fields[2])
	statusLine, statusLineOK := exactInt(fields[4])
	if !bodyOK || !headerBytesOK || !headerCountOK || !statusLineOK || int64(int(headerCount)) != headerCount {
		return httpOperationInput{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	policy, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: statusLine,
		HeaderBytes:     headerBytes,
		HeaderCount:     int(headerCount),
		BodyBytes:       body,
	})
	if err != nil {
		return httpOperationInput{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	raw, err := strictBase64Value(fields[3])
	if err != nil {
		return httpOperationInput{}, err
	}
	return httpOperationInput{policy: policy, raw: raw}, nil
}

func parseHTTPResponse(input httpOperationInput) (httpmodel.HTTPResponse, error) {
	if int64(len(input.raw)) > input.policy.MaximumResponseWireBytes() {
		return httpmodel.HTTPResponse{}, evaluationRefusal("OUTPUT_LIMIT")
	}
	response, err := httpmodel.ParseResponse(input.raw, input.policy)
	if err != nil {
		if code, ok := httpmodel.RefusalCodeOf(err); ok {
			switch code {
			case httpmodel.CodeResponseStatusLimit, httpmodel.CodeResponseHeaderLimit, httpmodel.CodeResponseBodyLimit:
				return httpmodel.HTTPResponse{}, evaluationRefusal("OUTPUT_LIMIT")
			}
		}
		return httpmodel.HTTPResponse{}, evaluationRefusal("RESPONSE_PARSE_FAILED")
	}
	return response, nil
}

func evaluateHTTPParse(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "body_bytes", "header_bytes", "header_count", "response_base64", "status_line_bytes")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	parsedInput, err := parseHTTPOperationInput(fields)
	if err != nil {
		return canon.Value{}, err
	}
	response, err := parseHTTPResponse(parsedInput)
	if err != nil {
		return canon.Value{}, err
	}
	headerValues := make([]canon.Value, 0, len(response.Headers()))
	for _, header := range response.Headers() {
		name, _ := textValue(header.Name())
		value, _ := textValue(header.Value())
		item, itemErr := objectValue(
			canon.Member{Name: "name", Value: name},
			canon.Member{Name: "value", Value: value},
		)
		if itemErr != nil {
			return canon.Value{}, itemErr
		}
		headerValues = append(headerValues, item)
	}
	headers, err := arrayValue(headerValues...)
	if err != nil {
		return canon.Value{}, err
	}
	body, _ := textValue(base64.StdEncoding.EncodeToString(response.Body()))
	contentLength, _ := integerValue(response.ContentLength())
	reason, _ := textValue(response.Reason())
	status, _ := integerValue(int64(response.Status()))
	return objectValue(
		canon.Member{Name: "body_base64", Value: body},
		canon.Member{Name: "content_length", Value: contentLength},
		canon.Member{Name: "headers", Value: headers},
		canon.Member{Name: "reason", Value: reason},
		canon.Member{Name: "status", Value: status},
	)
}

func strictFieldSelection(value canon.Value, profile []string, requireFull bool, code string) ([]string, error) {
	items, ok := exactArray(value)
	if !ok || len(items) == 0 || len(items) > len(profile) {
		return nil, evaluationRefusal(code)
	}
	positions := make(map[string]int, len(profile))
	for index, field := range profile {
		positions[field] = index
	}
	result := make([]string, len(items))
	previous := -1
	for index, item := range items {
		field, ok := exactText(item)
		position, exists := positions[field]
		if !ok || !exists || position <= previous {
			return nil, evaluationRefusal(code)
		}
		previous = position
		result[index] = field
	}
	if requireFull && !equalStrings(result, profile) {
		return nil, evaluationRefusal(code)
	}
	return result, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func parseCLICompletion(value canon.Value) (countercli.CLIProjectionCompletion, error) {
	kindValue, present := value.LookupMember("kind")
	kind, ok := kindValue.Text()
	if !present || !ok {
		return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	if kind == "EXITED" {
		fields, exact := exactObject(value, "code", "kind")
		code, integer := int64(0), false
		if exact {
			code, integer = exactInt(fields[0])
		}
		if !exact || !integer || code < 0 || code > 255 {
			return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
		}
		completion, err := countercli.NewExitedProjectionCompletion(int(code))
		if err != nil {
			return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
		}
		return completion, nil
	}
	if kind == "SIGNALED" {
		fields, exact := exactObject(value, "kind", "signal")
		if !exact {
			return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
		}
		signal, ok := exactText(fields[1])
		if !ok || len(signal) == 0 || len(signal) > 1024 {
			return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
		}
		completion, err := countercli.NewSignaledProjectionCompletion(signal)
		if err != nil {
			return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
		}
		return completion, nil
	}
	return countercli.CLIProjectionCompletion{}, evaluationRefusal("INVALID_OPERATION_INPUT")
}

func evaluateCLIProjection(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "completion", "selected_fields", "stderr_base64", "stdout_base64")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	selected, err := strictFieldSelection(fields[1], cliFieldIDs, false, "INVALID_OPERATION_INPUT")
	if err != nil {
		return canon.Value{}, err
	}
	completion, err := parseCLICompletion(fields[0])
	if err != nil {
		return canon.Value{}, err
	}
	stderr, err := strictBase64Value(fields[2])
	if err != nil {
		return canon.Value{}, err
	}
	stdout, err := strictBase64Value(fields[3])
	if err != nil {
		return canon.Value{}, err
	}
	definitionFields := make([]countercli.CLIFieldID, len(selected))
	for index, field := range selected {
		definitionFields[index] = countercli.CLIFieldID(field)
	}
	definition, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{Fields: definitionFields})
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	projection, err := definition.ProjectInput(countercli.CLIProjectionInput{
		Completion: completion, Stdout: stdout, Stderr: stderr,
	})
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	translated, err := resolved.Translate(projection)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	tuple, err := exactTupleFromTranslation(translated, selected)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	return tupleResultValue(tuple)
}

func evaluateHTTPProjection(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(
		input, "body_bytes", "header_bytes", "header_count", "response_base64", "scratch_root", "selected_fields", "status_line_bytes",
	)
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	selected, err := strictFieldSelection(fields[5], httpFieldIDs, false, "INVALID_OPERATION_INPUT")
	if err != nil {
		return canon.Value{}, err
	}
	scratchRoot, ok := exactText(fields[4])
	if !ok || !strings.HasPrefix(scratchRoot, "/") || path.Clean(scratchRoot) != scratchRoot {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	parsedInput, err := parseHTTPOperationInput([]canon.Value{fields[0], fields[1], fields[2], fields[3], fields[6]})
	if err != nil {
		if refusal, ok := err.(*evalError); ok && refusal.code == "INVALID_BASE64" {
			return canon.Value{}, err
		}
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	response, err := parseHTTPResponse(parsedInput)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	projection, err := counterhttp.ProjectParsedResponse(response, scratchRoot)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	translated, err := resolved.Translate(projection)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	tuple, err := exactTupleFromTranslation(translated, selected)
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	return tupleResultValue(tuple)
}

func exactTupleFromTranslation(translated projectiontranslate.Tuple, selected []string) (emittermodel.ExactTuple, error) {
	if !translated.Valid() {
		return emittermodel.ExactTuple{}, evaluationRefusal("PROJECTION_FAILED")
	}
	fields := make([]emittermodel.ExactField, len(selected))
	for index, fieldID := range selected {
		portable, present := translated.Value(fieldID)
		if !present {
			return emittermodel.ExactTuple{}, evaluationRefusal("PROJECTION_FAILED")
		}
		value, err := emittermodel.NewExactValue(portable)
		if err != nil {
			return emittermodel.ExactTuple{}, err
		}
		fields[index], err = emittermodel.NewExactField(fieldID, value)
		if err != nil {
			return emittermodel.ExactTuple{}, err
		}
	}
	return emittermodel.NewExactTuple(fields)
}

func tupleResultValue(tuple emittermodel.ExactTuple) (canon.Value, error) {
	parsed, err := canon.Parse(tuple.CanonicalBytes())
	if err != nil {
		return canon.Value{}, evaluationRefusal("PROJECTION_FAILED")
	}
	return objectValue(canon.Member{Name: "tuple", Value: parsed})
}

func parityProfile(adapter string, profileValue canon.Value) (projectionprofile.Profile, []string, error) {
	if adapter == "CLI" {
		fields, err := strictFieldSelection(profileValue, cliFieldIDs, false, "INVALID_PREDICATE")
		if err != nil {
			return projectionprofile.Profile{}, nil, err
		}
		definitionFields := make([]countercli.CLIFieldID, len(fields))
		for index, field := range fields {
			definitionFields[index] = countercli.CLIFieldID(field)
		}
		definition, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{Fields: definitionFields})
		if err != nil {
			return projectionprofile.Profile{}, nil, evaluationRefusal("INVALID_PREDICATE")
		}
		resolved, err := projectiontranslate.Resolve(definition.Binding())
		if err != nil {
			return projectionprofile.Profile{}, nil, evaluationRefusal("INVALID_PREDICATE")
		}
		return resolved.Profile(), fields, nil
	}
	if adapter == "HTTP" {
		fields, err := strictFieldSelection(profileValue, httpFieldIDs, true, "INVALID_PREDICATE")
		if err != nil {
			return projectionprofile.Profile{}, nil, err
		}
		definition, err := counterhttp.NewHTTPProjectionDefinition()
		if err != nil {
			return projectionprofile.Profile{}, nil, evaluationRefusal("INVALID_PREDICATE")
		}
		resolved, err := projectiontranslate.Resolve(definition.Binding())
		if err != nil {
			return projectionprofile.Profile{}, nil, evaluationRefusal("INVALID_PREDICATE")
		}
		return resolved.Profile(), fields, nil
	}
	return projectionprofile.Profile{}, nil, evaluationRefusal("INVALID_PREDICATE")
}

func stringArray(values []string) (canon.Value, error) {
	items := make([]canon.Value, len(values))
	for index, value := range values {
		item, err := textValue(value)
		if err != nil {
			return canon.Value{}, err
		}
		items[index] = item
	}
	return arrayValue(items...)
}

func evaluatePredicate(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(
		input, "adapter", "allowed_tuples", "decision_action", "observed_tuple", "profile_fields", "selected_fields", "stimulus_digest",
	)
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OPERATION_INPUT")
	}
	adapter, adapterOK := exactText(fields[0])
	action, actionOK := exactText(fields[2])
	stimulusText, stimulusOK := exactText(fields[6])
	if !adapterOK || !actionOK || !stimulusOK || (action != "ALLOW_OBSERVED" && action != "CUSTOM_EXPECTATION") {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	stimulus, err := domain.ParseDigest(stimulusText)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	profile, profileFields, err := parityProfile(adapter, fields[4])
	if err != nil {
		return canon.Value{}, err
	}
	selected, err := strictFieldSelection(fields[5], profileFields, false, "INVALID_PREDICATE")
	if err != nil {
		return canon.Value{}, err
	}
	allowed, ok := exactArray(fields[1])
	if !ok || len(allowed) == 0 || len(allowed) > 4 || action == "CUSTOM_EXPECTATION" && len(allowed) != 1 {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	kind, _ := textValue(emittermodel.PredicateKindV1)
	scope, _ := textValue(emittermodel.PredicateScopeV1)
	stimulusValue, _ := textValue(stimulus.String())
	profileDigest, _ := textValue(profile.Digest().String())
	selectedValue, err := stringArray(selected)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	predicateValue, err := objectValue(
		canon.Member{Name: "allowed_tuples", Value: fields[1]},
		canon.Member{Name: "kind", Value: kind},
		canon.Member{Name: "portable_profile_digest", Value: profileDigest},
		canon.Member{Name: "scope", Value: scope},
		canon.Member{Name: "selected_fields", Value: selectedValue},
		canon.Member{Name: "stimulus_digest", Value: stimulusValue},
	)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	predicateBytes, err := predicateValue.CanonicalChecked()
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	predicate, err := emittermodel.ParsePredicate(predicateBytes, profile, stimulus)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	observedBytes, err := fields[3].CanonicalChecked()
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	observed, err := emittermodel.ParseExactTuple(observedBytes)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	match, err := predicate.Matches(observed)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_PREDICATE")
	}
	return objectValue(canon.Member{Name: "match", Value: canon.Bool(match)})
}

func evaluateDirectResult(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "ineligible_reasons", "internal_failure", "predicate_match", "tamper")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_RESULT_FACTS")
	}
	reasonValues, reasonsOK := exactArray(fields[0])
	internalFailure, internalOK := exactBool(fields[1])
	predicateMatch, predicateOK := exactBool(fields[2])
	tamper, tamperOK := exactBool(fields[3])
	if !reasonsOK || !internalOK || !predicateOK || !tamperOK {
		return canon.Value{}, evaluationRefusal("INVALID_RESULT_FACTS")
	}
	reasons := make([]emittermodel.DirectReason, len(reasonValues))
	for index, value := range reasonValues {
		text, ok := exactText(value)
		if !ok {
			return canon.Value{}, evaluationRefusal("INVALID_RESULT_FACTS")
		}
		reasons[index] = emittermodel.DirectReason(text)
	}
	result, err := emittermodel.SelectDirectResult(emittermodel.DirectResultFacts{
		IneligibleReasons: reasons,
		InternalFailure:   internalFailure,
		PredicateMatch:    predicateMatch,
		Tamper:            tamper,
	})
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_RESULT_FACTS")
	}
	outcome, _ := textValue(result.Outcome)
	reason, _ := textValue(result.Reason)
	return objectValue(
		canon.Member{Name: "outcome", Value: outcome},
		canon.Member{Name: "reason", Value: reason},
	)
}

func evaluateOwnerEligibility(input canon.Value) (canon.Value, error) {
	fields, ok := exactObject(input, "kind", "reasons")
	if !ok {
		return canon.Value{}, evaluationRefusal("INVALID_OWNER_FACTS")
	}
	kindText, kindOK := exactText(fields[0])
	reasonValues, reasonsOK := exactArray(fields[1])
	if !kindOK || !reasonsOK {
		return canon.Value{}, evaluationRefusal("INVALID_OWNER_FACTS")
	}
	reasons := make([]domain.ControlReason, len(reasonValues))
	for index, value := range reasonValues {
		text, ok := exactText(value)
		if !ok {
			return canon.Value{}, evaluationRefusal("INVALID_OWNER_FACTS")
		}
		reasons[index] = domain.ControlReason(text)
	}
	decision, err := eligibilitycore.Select(eligibilitycore.FactKind(kindText), reasons)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OWNER_FACTS")
	}
	eligibility := "INELIGIBLE"
	if decision.IsEligible() {
		eligibility = "ELIGIBLE"
	}
	eligibilityValue, _ := textValue(eligibility)
	reasonStrings := make([]string, len(decision.Reasons()))
	for index, reason := range decision.Reasons() {
		reasonStrings[index] = string(reason)
	}
	reasonsValue, err := stringArray(reasonStrings)
	if err != nil {
		return canon.Value{}, evaluationRefusal("INVALID_OWNER_FACTS")
	}
	return objectValue(
		canon.Member{Name: "eligibility", Value: eligibilityValue},
		canon.Member{Name: "reasons", Value: reasonsValue},
	)
}
