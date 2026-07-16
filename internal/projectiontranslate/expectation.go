package projectiontranslate

import (
	"bytes"
	"encoding/base64"
	"strconv"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	// PortableChoiceModeV1 is the exact durable Choice interpretation whose
	// runtime compatibility firewalls are derived in this package.
	PortableChoiceModeV1 = "ADAPTER_BOUND_PORTABLE_FIELDS_V1"

	portableExpectationSemanticsV1 = "EXISTS_COMPLETE_ADAPTER_PROJECTION_RESTRICTION_V1"
	cliExpectationValidatorV1      = "CLI_PORTABLE_EXPECTATION_DOMAIN_V1"
	httpExpectationValidatorV1     = "HTTP_PORTABLE_EXPECTATION_DOMAIN_V1"
)

const (
	ruleCLICompletionKind = "CLI_COMPLETION_KIND_EXITED_OR_SIGNALED_V1"
	ruleCLIExitCode       = "CLI_EXIT_CODE_0_255_OR_MISSING_FOR_SIGNAL_V1"
	ruleCLIExitSignal     = "CLI_SIGNAL_NONEMPTY_UTF8_1024_OR_MISSING_FOR_EXIT_V1"
	ruleCLIStdoutBytes    = "CLI_STDOUT_EXACT_BYTES_V1"
	ruleCLIStderrText     = "CLI_STDERR_UTF8_STRING_V1"
	ruleCLIJSONMode       = "CLI_STDOUT_JSON_MODE_MISSING_OR_STRING_V1"
	ruleCLIJSONSource     = "CLI_STDOUT_JSON_SOURCE_MISSING_OR_STRING_V1"
	ruleHTTPStatus        = "HTTP_STATUS_200_599_V1"
	ruleHTTPContentType   = "HTTP_CONTENT_TYPE_MISSING_OR_NONEMPTY_PRINTABLE_ASCII_ORDERED_LIST_V1"
	ruleHTTPBodyKind      = "HTTP_BODY_KIND_UTF8_STRING_V1"
	ruleHTTPBodyMetadata  = "HTTP_BODY_METADATA_CANONICAL_JSON_OBJECT_V1"

	ruleCLICompletionSum   = "CLI_COMPLETION_EXITED_CODE_XOR_SIGNALED_SIGNAL_V1"
	ruleCLIStdoutCoherence = "CLI_SELECTED_STDOUT_BYTES_CONSTRAIN_ACTIVE_JSON_FIELDS_V1"
	ruleIndependentFields  = "INDEPENDENT_PROFILE_FIELDS_V1"
)

type expectationFieldRuleIdentity struct {
	FieldID string `json:"field_id"`
	RuleID  string `json:"rule_id"`
}

type expectationDomainIdentity struct {
	SchemaVersion           string                         `json:"schema_version"`
	Kind                    string                         `json:"kind"`
	Semantics               string                         `json:"semantics"`
	AdapterDomain           string                         `json:"adapter_domain"`
	ProjectionBindingDigest string                         `json:"projection_binding_digest"`
	ProfileDigest           string                         `json:"profile_digest"`
	ProfileBase64           string                         `json:"profile_base64"`
	Validator               string                         `json:"validator"`
	FieldRules              []expectationFieldRuleIdentity `json:"field_rules"`
	TupleRules              []string                       `json:"tuple_rules"`
}

// SelectedValue is inert, exact portable input to an ExpectationDomain. It is
// not a ruling, source, or emission capability.
type SelectedValue struct {
	FieldID string
	Value   portablevalue.Value
}

// ExpectationDomain is the sealed existential projection-model validator for
// one exact resolved profile. It proves model-shape realizability only; it does
// not prove that any current source, candidate, or runtime can emit a value.
type ExpectationDomain struct {
	profile   projectionprofile.Profile
	arm       adapterArm
	digest    domain.Digest
	canonical []byte
	seal      *expectationDomainSeal
}

type expectationDomainSeal struct{}

var expectationDomainAuthority = &expectationDomainSeal{}

// ExpectationDomain returns inert model-shape validation for this exact
// resolved profile. It does not confer Choice, store, source, or emission
// authority.
func (r Resolved) ExpectationDomain() (ExpectationDomain, error) {
	return newExpectationDomain(r)
}

func newExpectationDomain(resolved Resolved) (ExpectationDomain, error) {
	if !resolved.Valid() {
		return ExpectationDomain{}, refuse(CodeProfileMismatch, "", "expectation domain requires one exact resolved profile")
	}
	fieldRules, tupleRules, validator, err := expectationRules(resolved.profile, resolved.arm)
	if err != nil {
		return ExpectationDomain{}, err
	}
	identity := expectationDomainIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "PortableExpectationDomain",
		Semantics: portableExpectationSemanticsV1, AdapterDomain: string(resolved.profile.AdapterDomain()),
		ProjectionBindingDigest: resolved.profile.Binding().Digest().String(),
		ProfileDigest:           resolved.profile.Digest().String(),
		ProfileBase64:           base64.StdEncoding.EncodeToString(resolved.profile.CanonicalBytes()),
		Validator:               validator, FieldRules: fieldRules, TupleRules: tupleRules,
	}
	digestRaw, canonical, err := canon.DigestTyped("PortableExpectationDomain", identity)
	if err != nil || len(canonical) > canon.MaxInputBytes {
		return ExpectationDomain{}, refuse(CodeTranslationLimit, "", "portable expectation-domain identity exceeds the canonical ceiling")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return ExpectationDomain{}, refuse(CodeProfileMismatch, "", "portable expectation-domain digest is invalid")
	}
	return ExpectationDomain{
		profile: resolved.profile, arm: resolved.arm, digest: digest,
		canonical: canonical, seal: expectationDomainAuthority,
	}, nil
}

func expectationRules(profile projectionprofile.Profile, arm adapterArm) ([]expectationFieldRuleIdentity, []string, string, error) {
	fields := profile.Fields()
	rules := make([]expectationFieldRuleIdentity, len(fields))
	tupleRules := []string{portableExpectationSemanticsV1}
	validator := ""
	hasCompletion := false
	hasStdoutBytes := false
	hasJSON := false
	for index, field := range fields {
		rule := ""
		switch arm {
		case armCLI:
			validator = cliExpectationValidatorV1
			switch field.FieldID {
			case string(cli.CLIFieldCompletionKind):
				rule, hasCompletion = ruleCLICompletionKind, true
			case string(cli.CLIFieldExitCode):
				rule, hasCompletion = ruleCLIExitCode, true
			case string(cli.CLIFieldExitSignal):
				rule, hasCompletion = ruleCLIExitSignal, true
			case string(cli.CLIFieldStdoutBytes):
				rule, hasStdoutBytes = ruleCLIStdoutBytes, true
			case string(cli.CLIFieldStderrText):
				rule = ruleCLIStderrText
			case string(cli.CLIFieldStdoutJSONMode):
				rule, hasJSON = ruleCLIJSONMode, true
			case string(cli.CLIFieldStdoutJSONSource):
				rule, hasJSON = ruleCLIJSONSource, true
			}
		case armHTTP:
			validator = httpExpectationValidatorV1
			switch field.FieldID {
			case string(counterhttp.HTTPFieldStatus):
				rule = ruleHTTPStatus
			case string(counterhttp.HTTPFieldContentType):
				rule = ruleHTTPContentType
			case string(counterhttp.HTTPFieldBodyKind):
				rule = ruleHTTPBodyKind
			case string(counterhttp.HTTPFieldBodyMetadata):
				rule = ruleHTTPBodyMetadata
			}
		}
		if rule == "" {
			return nil, nil, "", refuse(CodeProfileMismatch, field.FieldID, "portable expectation domain encountered an unknown profile field")
		}
		rules[index] = expectationFieldRuleIdentity{FieldID: field.FieldID, RuleID: rule}
	}
	if arm == armCLI {
		if hasCompletion {
			tupleRules = append(tupleRules, ruleCLICompletionSum)
		}
		if hasStdoutBytes && hasJSON {
			tupleRules = append(tupleRules, ruleCLIStdoutCoherence)
		}
	} else if arm == armHTTP {
		tupleRules = append(tupleRules, ruleIndependentFields)
	} else {
		return nil, nil, "", refuse(CodeProfileMismatch, "", "portable expectation domain has an unknown adapter arm")
	}
	return rules, tupleRules, validator, nil
}

func (d ExpectationDomain) Valid() bool {
	if d.seal != expectationDomainAuthority || !d.profile.Valid() || !d.digest.Valid() || len(d.canonical) == 0 {
		return false
	}
	rebuilt, err := newExpectationDomain(Resolved{
		profile: d.profile, arm: d.arm,
		stimulusKind: expectationStimulusKind(d.arm), seal: resolvedAuthority,
	})
	return err == nil && rebuilt.digest == d.digest && bytes.Equal(rebuilt.canonical, d.canonical)
}

func expectationStimulusKind(arm adapterArm) string {
	if arm == armCLI {
		return "CLIStimulus"
	}
	if arm == armHTTP {
		return "HTTPStimulus"
	}
	return ""
}

func (d ExpectationDomain) Digest() domain.Digest { return d.digest }

func (d ExpectationDomain) CanonicalBytes() []byte { return append([]byte(nil), d.canonical...) }

func (d ExpectationDomain) ProfileDigest() domain.Digest { return d.profile.Digest() }

func (d ExpectationDomain) ProfileBytes() []byte { return d.profile.CanonicalBytes() }

func (d ExpectationDomain) clone() ExpectationDomain {
	return ExpectationDomain{
		profile: d.profile, arm: d.arm, digest: d.digest,
		canonical: append([]byte(nil), d.canonical...), seal: d.seal,
	}
}

// ValidateSelected accepts exactly those selected values that can be the
// restriction of at least one complete tuple in this exact adapter projection
// model. Unselected values are existential and are never returned or stored.
func (d ExpectationDomain) ValidateSelected(selected []SelectedValue) error {
	if !d.Valid() {
		return refuse(CodeProfileMismatch, "", "portable expectation-domain capability is invalid")
	}
	profileFields := d.profile.Fields()
	if len(selected) == 0 || len(selected) > len(profileFields) {
		return refuse(CodeCustomExpectationNotRealizable, "", "selected expectation is empty or exceeds the exact profile")
	}
	positions := make(map[string]int, len(profileFields))
	descriptors := make(map[string]projectionprofile.Descriptor, len(profileFields))
	for index, field := range profileFields {
		positions[field.FieldID] = index
		descriptors[field.FieldID] = field
	}
	values := make(map[string]portablevalue.Value, len(selected))
	orderedValues := make([]portablevalue.Value, len(selected))
	lastPosition := -1
	for index, field := range selected {
		position, present := positions[field.FieldID]
		descriptor := descriptors[field.FieldID]
		if !present || position <= lastPosition || !field.Value.Valid() {
			return refuse(CodeProfileMismatch, field.FieldID, "selected expectation differs from the exact profile order or value algebra")
		}
		switch field.Value.Tag() {
		case portablevalue.TagMissing:
			if !descriptor.AllowMissing {
				return refuse(CodeCustomExpectationNotRealizable, field.FieldID, "selected expectation uses missing where the adapter requires presence")
			}
		case portablevalue.TagNull:
			if !descriptor.AllowNull {
				return refuse(CodeCustomExpectationNotRealizable, field.FieldID, "selected expectation uses null where the adapter does not emit null")
			}
		default:
			if field.Value.Tag() != descriptor.PortableTag {
				return refuse(CodeProfileMismatch, field.FieldID, "selected expectation tag differs from the exact profile")
			}
		}
		lastPosition = position
		if _, duplicate := values[field.FieldID]; duplicate {
			return refuse(CodeProfileMismatch, field.FieldID, "selected expectation repeats a profile field")
		}
		values[field.FieldID] = field.Value
		orderedValues[index] = field.Value
	}
	if err := portablevalue.ValidateTuple(orderedValues); err != nil {
		return refuse(CodeTranslationLimit, "", "selected expectation exceeds the portable value profile")
	}
	if d.arm == armCLI {
		return validateCLISelectedExpectation(profileFields, values)
	}
	if d.arm == armHTTP {
		return validateHTTPSelectedExpectation(values)
	}
	return refuse(CodeProfileMismatch, "", "selected expectation has an unknown adapter arm")
}

func validateCLISelectedExpectation(profileFields []projectionprofile.Descriptor, values map[string]portablevalue.Value) error {
	const (
		possibleExited = 1 << iota
		possibleSignaled
	)
	possible := possibleExited | possibleSignaled
	if value, present := values[string(cli.CLIFieldCompletionKind)]; present {
		text, ok := value.StringText()
		if !ok || (text != string(cli.CompletionExited) && text != string(cli.CompletionSignaled)) {
			return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldCompletionKind), "CLI completion kind is outside the closed adapter sum")
		}
		if text == string(cli.CompletionExited) {
			possible &= possibleExited
		} else {
			possible &= possibleSignaled
		}
	}
	if value, present := values[string(cli.CLIFieldExitCode)]; present {
		switch value.Tag() {
		case portablevalue.TagMissing:
			possible &= possibleSignaled
		case portablevalue.TagInteger:
			canonical, _ := value.IntegerText()
			code, err := strconv.Atoi(canonical)
			if err != nil || code < 0 || code > 255 {
				return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldExitCode), "CLI exit code is outside 0..255")
			}
			possible &= possibleExited
		default:
			return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldExitCode), "CLI exit code has no adapter completion state")
		}
	}
	if value, present := values[string(cli.CLIFieldExitSignal)]; present {
		switch value.Tag() {
		case portablevalue.TagMissing:
			possible &= possibleExited
		case portablevalue.TagString:
			signal, _ := value.StringText()
			if signal == "" || len(signal) > 1024 {
				return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldExitSignal), "CLI signal is empty or exceeds the adapter text ceiling")
			}
			possible &= possibleSignaled
		default:
			return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldExitSignal), "CLI signal has no adapter completion state")
		}
	}
	if possible == 0 {
		return refuse(CodeCustomExpectationNotRealizable, "", "CLI completion fields cannot coexist in one adapter projection")
	}
	return validateCLIStdoutCoherence(profileFields, values)
}

func validateCLIStdoutCoherence(profileFields []projectionprofile.Descriptor, values map[string]portablevalue.Value) error {
	activeJSON := make(map[string]string, 2)
	for _, field := range profileFields {
		switch field.FieldID {
		case string(cli.CLIFieldStdoutJSONMode):
			activeJSON[field.FieldID] = "mode"
		case string(cli.CLIFieldStdoutJSONSource):
			activeJSON[field.FieldID] = "source"
		}
	}
	stdout, selectedBytes := values[string(cli.CLIFieldStdoutBytes)]
	if !selectedBytes || len(activeJSON) == 0 {
		return nil
	}
	exact, ok := stdout.BytesValue()
	if !ok {
		return refuse(CodeProfileMismatch, string(cli.CLIFieldStdoutBytes), "CLI stdout expectation is not exact bytes")
	}
	root, err := canon.Parse(exact)
	if err != nil || root.Kind() != canon.KindObject {
		return refuse(CodeCustomExpectationNotRealizable, string(cli.CLIFieldStdoutBytes), "selected CLI stdout bytes are not one strict JSON object required by the active profile")
	}
	for fieldID, memberName := range activeJSON {
		member, present := root.LookupMember(memberName)
		memberText := ""
		if present {
			var stringValue bool
			memberText, stringValue = member.Text()
			if !stringValue {
				return refuse(CodeCustomExpectationNotRealizable, fieldID, "selected CLI stdout bytes give an active JSON member the wrong type")
			}
		}
		expected, selectedField := values[fieldID]
		if !selectedField {
			continue
		}
		if expected.Tag() == portablevalue.TagMissing {
			if present {
				return refuse(CodeCustomExpectationNotRealizable, fieldID, "selected missing JSON field is present in selected stdout bytes")
			}
			continue
		}
		text, ok := expected.StringText()
		if !ok || !present || text != memberText {
			return refuse(CodeCustomExpectationNotRealizable, fieldID, "selected JSON field differs from selected stdout bytes")
		}
	}
	return nil
}

func validateHTTPSelectedExpectation(values map[string]portablevalue.Value) error {
	if value, present := values[string(counterhttp.HTTPFieldStatus)]; present {
		canonical, ok := value.IntegerText()
		status, err := strconv.Atoi(canonical)
		if !ok || err != nil || status < 200 || status > 599 {
			return refuse(CodeCustomExpectationNotRealizable, string(counterhttp.HTTPFieldStatus), "HTTP status is outside 200..599")
		}
	}
	if value, present := values[string(counterhttp.HTTPFieldContentType)]; present && value.Tag() != portablevalue.TagMissing {
		members, ok := value.OrderedStrings()
		if !ok || len(members) == 0 {
			return refuse(CodeCustomExpectationNotRealizable, string(counterhttp.HTTPFieldContentType), "present HTTP content-type must contain at least one ordered value")
		}
		for _, member := range members {
			for index := 0; index < len(member); index++ {
				if member[index] < 0x20 || member[index] > 0x7e {
					return refuse(CodeCustomExpectationNotRealizable, string(counterhttp.HTTPFieldContentType), "HTTP content-type contains bytes outside the adapter wire profile")
				}
			}
		}
	}
	if value, present := values[string(counterhttp.HTTPFieldBodyMetadata)]; present {
		exact, ok := value.CanonicalJSONBytes()
		parsed, err := canon.Parse(exact)
		if !ok || err != nil || parsed.Kind() != canon.KindObject {
			return refuse(CodeCustomExpectationNotRealizable, string(counterhttp.HTTPFieldBodyMetadata), "HTTP body metadata must remain one exact canonical JSON object")
		}
	}
	return nil
}
