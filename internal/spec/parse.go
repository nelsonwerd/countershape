package spec

import (
	"fmt"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const sourceSchemaVersion = "countershape-source/v1"
const maxSourceBytes = 1 << 20

const (
	maxSourceStringBytes = 4096
	maxArgvItems         = 64
	maxEnvironmentItems  = 5
	maxSecretSlotItems   = 16
	maxRequiredToolItems = 16
)

var topLevelFields = fieldSet(
	"schema_version", "kind", "candidate_set_digest", "materialization_policy_digest",
	"comparison_envelope_digest", "adapter", "execution_shape", "start_argv", "setup_argv",
	"environment", "secret_slots", "fixture_recipe_digest", "readiness", "capture_policy_digest",
	"projection_definition_digest", "repeat_schedule", "required_tools", "budgets",
)

// ParseSource is the sole production constructor for ParsedSource. It first
// uses the lossless strict canonical parser, then applies a closed typed
// grammar. Unknown keys are refused at every level.
func ParseSource(exact []byte) (ParsedSource, error) {
	if len(exact) == 0 || len(exact) > maxSourceBytes {
		return ParsedSource{}, refuse(CodeSourceTooLarge, "$", "source must contain 1..1048576 bytes")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return ParsedSource{}, err
	}
	top, err := objectAt(value, "$", topLevelFields, fieldSet(
		"schema_version", "kind", "candidate_set_digest", "materialization_policy_digest",
		"comparison_envelope_digest", "adapter", "execution_shape", "start_argv",
		"fixture_recipe_digest", "capture_policy_digest", "projection_definition_digest",
	))
	if err != nil {
		return ParsedSource{}, err
	}
	if err := exactString(top, "schema_version", sourceSchemaVersion, "$."); err != nil {
		return ParsedSource{}, err
	}
	if err := exactString(top, "kind", "SourceSpec", "$."); err != nil {
		return ParsedSource{}, err
	}
	canonicalDigest, err := canon.DigestValue("SourceSpec", value)
	if err != nil {
		return ParsedSource{}, err
	}
	sourceDigest, err := domain.ParseDigest(canonicalDigest.String())
	if err != nil {
		return ParsedSource{}, err
	}

	candidateSet, err := digestField(top, "candidate_set_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}
	materialization, err := digestField(top, "materialization_policy_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}
	envelope, err := digestField(top, "comparison_envelope_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}
	fixture, err := digestField(top, "fixture_recipe_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}
	capture, err := digestField(top, "capture_policy_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}
	projection, err := digestField(top, "projection_definition_digest", "$.")
	if err != nil {
		return ParsedSource{}, err
	}

	adapter, err := parseAdapter(top["adapter"])
	if err != nil {
		return ParsedSource{}, err
	}
	shapeText, err := textValue(top["execution_shape"], "$.execution_shape")
	if err != nil {
		return ParsedSource{}, err
	}
	shape := domain.ExecutionShape(shapeText)
	start, err := stringArray(top["start_argv"], "$.start_argv")
	if err != nil {
		return ParsedSource{}, err
	}

	setup := []string{}
	if raw, ok := top["setup_argv"]; ok {
		setup, err = stringArray(raw, "$.setup_argv")
		if err != nil {
			return ParsedSource{}, err
		}
	}
	environment := []domain.EnvironmentEntry{}
	if raw, ok := top["environment"]; ok {
		environment, err = parseEnvironment(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}
	secretSlots := []domain.SecretSlot{}
	if raw, ok := top["secret_slots"]; ok {
		secretSlots, err = parseSecretSlots(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}
	readiness := domain.Readiness{Kind: domain.ReadinessNone}
	if raw, ok := top["readiness"]; ok {
		readiness, err = parseReadiness(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}
	repeats := repeatOverrides{}
	if raw, ok := top["repeat_schedule"]; ok {
		repeats, err = parseRepeats(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}
	tools := []domain.RequiredTool{}
	if raw, ok := top["required_tools"]; ok {
		tools, err = parseTools(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}
	budgets := budgetOverrides{}
	if raw, ok := top["budgets"]; ok {
		budgets, err = parseBudgets(raw)
		if err != nil {
			return ParsedSource{}, err
		}
	}

	parsed := ParsedSource{
		valid:                       true,
		digest:                      sourceDigest,
		canonicalBytes:              value.Canonical(),
		candidateSetDigest:          candidateSet,
		materializationPolicyDigest: materialization,
		comparisonEnvelopeDigest:    envelope,
		adapter:                     adapter,
		executionShape:              shape,
		startArgv:                   append([]string(nil), start...),
		setupArgv:                   append([]string(nil), setup...),
		environment:                 append([]domain.EnvironmentEntry(nil), environment...),
		secretSlots:                 append([]domain.SecretSlot(nil), secretSlots...),
		fixtureRecipeDigest:         fixture,
		readiness:                   readiness,
		capturePolicyDigest:         capture,
		projectionDefinitionDigest:  projection,
		repeats:                     repeats,
		requiredTools:               append([]domain.RequiredTool(nil), tools...),
		budgets:                     budgets,
	}
	// Parsing admits not only the token grammar but the complete semantic source
	// grammar. An invalid enum, budget, argv, readiness, or environment profile
	// never receives a usable ParsedSource capability or SourceSpec digest.
	if err := domain.ValidateWorldPlanDeclaration(declarationForSource(parsed)); err != nil {
		return ParsedSource{}, err
	}
	return parsed, nil
}

func parseAdapter(value canon.Value) (domain.Adapter, error) {
	object, err := objectAt(value, "$.adapter", fieldSet("domain", "adapter_version", "runner_digest"), fieldSet("domain", "adapter_version", "runner_digest"))
	if err != nil {
		return domain.Adapter{}, err
	}
	domainText, err := textField(object, "domain", "$.adapter.")
	if err != nil {
		return domain.Adapter{}, err
	}
	version, err := textField(object, "adapter_version", "$.adapter.")
	if err != nil {
		return domain.Adapter{}, err
	}
	runner, err := digestField(object, "runner_digest", "$.adapter.")
	if err != nil {
		return domain.Adapter{}, err
	}
	return domain.Adapter{Domain: domain.AdapterDomain(domainText), AdapterVersion: version, RunnerDigest: runner}, nil
}

func parseEnvironment(value canon.Value) ([]domain.EnvironmentEntry, error) {
	values, err := arrayValue(value, "$.environment")
	if err != nil {
		return nil, err
	}
	if len(values) > maxEnvironmentItems {
		return nil, refuse(CodeInvalidSourceValue, "$.environment", "environment exceeds the v1 item bound")
	}
	result := make([]domain.EnvironmentEntry, 0, len(values))
	for index, item := range values {
		path := fmt.Sprintf("$.environment[%d]", index)
		object, err := objectAt(item, path, fieldSet("name", "value"), fieldSet("name", "value"))
		if err != nil {
			return nil, err
		}
		name, err := textField(object, "name", path+".")
		if err != nil {
			return nil, err
		}
		text, err := textField(object, "value", path+".")
		if err != nil {
			return nil, err
		}
		result = append(result, domain.EnvironmentEntry{Name: name, Value: text})
	}
	return result, nil
}

func parseSecretSlots(value canon.Value) ([]domain.SecretSlot, error) {
	values, err := arrayValue(value, "$.secret_slots")
	if err != nil {
		return nil, err
	}
	if len(values) > maxSecretSlotItems {
		return nil, refuse(CodeInvalidSourceValue, "$.secret_slots", "secret_slots exceeds the v1 item bound")
	}
	result := make([]domain.SecretSlot, 0, len(values))
	for index, item := range values {
		path := fmt.Sprintf("$.secret_slots[%d]", index)
		object, err := objectAt(item, path, fieldSet("name", "presence"), fieldSet("name", "presence"))
		if err != nil {
			return nil, err
		}
		name, err := textField(object, "name", path+".")
		if err != nil {
			return nil, err
		}
		presence, err := textField(object, "presence", path+".")
		if err != nil {
			return nil, err
		}
		result = append(result, domain.SecretSlot{Name: name, Presence: domain.SecretPresence(presence)})
	}
	return result, nil
}

func parseReadiness(value canon.Value) (domain.Readiness, error) {
	members, ok := value.Members()
	if !ok {
		return domain.Readiness{}, refuse(CodeInvalidSourceType, "$.readiness", "expected object")
	}
	pre := make(map[string]canon.Value, len(members))
	for _, member := range members {
		pre[member.Name] = member.Value
	}
	kind, err := textField(pre, "kind", "$.readiness.")
	if err != nil {
		return domain.Readiness{}, err
	}
	allowed := fieldSet("kind")
	required := fieldSet("kind")
	if kind == string(domain.FixtureOwnedReadiness) {
		allowed["signal_name"] = struct{}{}
		required["signal_name"] = struct{}{}
	}
	object, err := objectAt(value, "$.readiness", allowed, required)
	if err != nil {
		return domain.Readiness{}, err
	}
	signal := ""
	if raw, exists := object["signal_name"]; exists {
		signal, err = textValue(raw, "$.readiness.signal_name")
		if err != nil {
			return domain.Readiness{}, err
		}
	}
	return domain.Readiness{Kind: domain.ReadinessKind(kind), SignalName: signal}, nil
}

func parseRepeats(value canon.Value) (repeatOverrides, error) {
	object, err := objectAt(
		value,
		"$.repeat_schedule",
		fieldSet("discovery_repeats", "confirmation_repeats", "concurrency", "rotation"),
		fieldSet(),
	)
	if err != nil {
		return repeatOverrides{}, err
	}
	result := repeatOverrides{}
	if raw, ok := object["discovery_repeats"]; ok {
		parsed, err := intValue(raw, "$.repeat_schedule.discovery_repeats")
		if err != nil {
			return result, err
		}
		result.DiscoveryRepeats = &parsed
	}
	if raw, ok := object["confirmation_repeats"]; ok {
		parsed, err := intValue(raw, "$.repeat_schedule.confirmation_repeats")
		if err != nil {
			return result, err
		}
		result.ConfirmationRepeats = &parsed
	}
	if raw, ok := object["concurrency"]; ok {
		parsed, err := textValue(raw, "$.repeat_schedule.concurrency")
		if err != nil {
			return result, err
		}
		value := domain.ScheduleConcurrency(parsed)
		result.Concurrency = &value
	}
	if raw, ok := object["rotation"]; ok {
		parsed, err := textValue(raw, "$.repeat_schedule.rotation")
		if err != nil {
			return result, err
		}
		value := domain.ScheduleRotation(parsed)
		result.Rotation = &value
	}
	return result, nil
}

func parseTools(value canon.Value) ([]domain.RequiredTool, error) {
	values, err := arrayValue(value, "$.required_tools")
	if err != nil {
		return nil, err
	}
	if len(values) > maxRequiredToolItems {
		return nil, refuse(CodeInvalidSourceValue, "$.required_tools", "required_tools exceeds the v1 item bound")
	}
	result := make([]domain.RequiredTool, 0, len(values))
	for index, item := range values {
		path := fmt.Sprintf("$.required_tools[%d]", index)
		object, err := objectAt(item, path, fieldSet("name", "version_constraint"), fieldSet("name", "version_constraint"))
		if err != nil {
			return nil, err
		}
		name, err := textField(object, "name", path+".")
		if err != nil {
			return nil, err
		}
		constraint, err := textField(object, "version_constraint", path+".")
		if err != nil {
			return nil, err
		}
		result = append(result, domain.RequiredTool{Name: name, VersionConstraint: constraint})
	}
	return result, nil
}

var budgetFields = fieldSet(
	"candidate_count", "materialized_entry_count", "materialized_bytes_per_world", "single_blob_bytes",
	"readiness_ms", "probe_ms", "teardown_ms", "stdout_bytes", "stderr_bytes", "http_body_bytes",
	"proposed_shrink_stimuli", "total_candidate_trials", "shrink_wall_ms",
)

var budgetFieldOrder = []string{
	"candidate_count", "materialized_entry_count", "materialized_bytes_per_world", "single_blob_bytes",
	"readiness_ms", "probe_ms", "teardown_ms", "stdout_bytes", "stderr_bytes", "http_body_bytes",
	"proposed_shrink_stimuli", "total_candidate_trials", "shrink_wall_ms",
}

func parseBudgets(value canon.Value) (budgetOverrides, error) {
	object, err := objectAt(value, "$.budgets", budgetFields, fieldSet())
	if err != nil {
		return budgetOverrides{}, err
	}
	result := budgetOverrides{}
	// A stable field order makes the first refusal reproducible when several
	// overrides are malformed. Map iteration must never select diagnostics.
	for _, name := range budgetFieldOrder {
		raw, exists := object[name]
		if !exists {
			continue
		}
		value, ok := raw.Int64()
		if !ok {
			return result, refuse(CodeInvalidSourceType, "$.budgets."+name, "expected integer")
		}
		switch name {
		case "candidate_count":
			parsed, err := checkedInt(value, name)
			if err != nil {
				return result, err
			}
			result.CandidateCount = &parsed
		case "materialized_entry_count":
			parsed, err := checkedInt(value, name)
			if err != nil {
				return result, err
			}
			result.MaterializedEntryCount = &parsed
		case "materialized_bytes_per_world":
			result.MaterializedBytesPerWorld = &value
		case "single_blob_bytes":
			result.SingleBlobBytes = &value
		case "readiness_ms":
			result.ReadinessMS = &value
		case "probe_ms":
			result.ProbeMS = &value
		case "teardown_ms":
			result.TeardownMS = &value
		case "stdout_bytes":
			result.StdoutBytes = &value
		case "stderr_bytes":
			result.StderrBytes = &value
		case "http_body_bytes":
			result.HTTPBodyBytes = &value
		case "proposed_shrink_stimuli":
			parsed, err := checkedInt(value, name)
			if err != nil {
				return result, err
			}
			result.ProposedShrinkStimuli = &parsed
		case "total_candidate_trials":
			parsed, err := checkedInt(value, name)
			if err != nil {
				return result, err
			}
			result.TotalCandidateTrials = &parsed
		case "shrink_wall_ms":
			result.ShrinkWallMS = &value
		}
	}
	return result, nil
}

func fieldSet(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func objectAt(value canon.Value, path string, allowed, required map[string]struct{}) (map[string]canon.Value, error) {
	members, ok := value.Members()
	if !ok {
		return nil, refuse(CodeInvalidSourceType, path, "expected object")
	}
	result := make(map[string]canon.Value, len(members))
	for _, member := range members {
		if _, ok := allowed[member.Name]; !ok {
			return nil, refuse(CodeUnknownSourceKey, path+"."+member.Name, "field is not in the closed source grammar")
		}
		result[member.Name] = member.Value
	}
	requiredNames := make([]string, 0, len(required))
	for name := range required {
		requiredNames = append(requiredNames, name)
	}
	sort.Strings(requiredNames)
	for _, name := range requiredNames {
		if _, ok := result[name]; !ok {
			return nil, refuse(CodeMissingSourceField, path+"."+name, "required field is absent")
		}
	}
	return result, nil
}

func arrayValue(value canon.Value, path string) ([]canon.Value, error) {
	values, ok := value.Elements()
	if !ok {
		return nil, refuse(CodeInvalidSourceType, path, "expected array")
	}
	return values, nil
}

func stringArray(value canon.Value, path string) ([]string, error) {
	values, err := arrayValue(value, path)
	if err != nil {
		return nil, err
	}
	if len(values) > maxArgvItems {
		return nil, refuse(CodeInvalidSourceValue, path, "argv exceeds the v1 item bound")
	}
	result := make([]string, 0, len(values))
	totalBytes := 0
	for index, item := range values {
		text, err := textValue(item, fmt.Sprintf("%s[%d]", path, index))
		if err != nil {
			return nil, err
		}
		totalBytes += len(text)
		if totalBytes > 65536 {
			return nil, refuse(CodeInvalidSourceValue, path, "argv exceeds the v1 aggregate byte bound")
		}
		result = append(result, text)
	}
	return result, nil
}

func textField(object map[string]canon.Value, name, prefix string) (string, error) {
	value, ok := object[name]
	if !ok {
		return "", refuse(CodeMissingSourceField, prefix+name, "required field is absent")
	}
	return textValue(value, prefix+name)
}

func textValue(value canon.Value, path string) (string, error) {
	text, ok := value.Text()
	if !ok {
		return "", refuse(CodeInvalidSourceType, path, "expected string")
	}
	if len(text) > maxSourceStringBytes {
		return "", refuse(CodeInvalidSourceValue, path, "string exceeds the v1 byte bound")
	}
	return text, nil
}

func exactString(object map[string]canon.Value, name, expected, prefix string) error {
	value, err := textField(object, name, prefix)
	if err != nil {
		return err
	}
	if value != expected {
		return refuse(CodeInvalidSourceValue, prefix+name, "expected "+expected)
	}
	return nil
}

func digestField(object map[string]canon.Value, name, prefix string) (domain.Digest, error) {
	value, err := textField(object, name, prefix)
	if err != nil {
		return "", err
	}
	digest, err := domain.ParseDigest(value)
	if err != nil {
		return "", refuse(CodeInvalidSourceValue, prefix+name, "invalid digest")
	}
	return digest, nil
}

func intValue(value canon.Value, path string) (int, error) {
	integer, ok := value.Int64()
	if !ok {
		return 0, refuse(CodeInvalidSourceType, path, "expected integer")
	}
	return checkedInt(integer, path)
}

func checkedInt(value int64, path string) (int, error) {
	converted := int(value)
	if int64(converted) != value {
		return 0, refuse(CodeInvalidSourceValue, path, "integer is outside native range")
	}
	return converted, nil
}
