package model

import (
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const cliProjectionVersionV1 = "cli-projection/v1"

type closedCLIFieldDescriptor struct {
	id, channel, valueKind, missingPolicy string
	path                                  []string
}

var closedCLIFieldRegistryV1 = []closedCLIFieldDescriptor{
	{id: "cli.completion.kind", channel: "exit", path: []string{"completion", "kind"}, valueKind: "UTF8_STRING", missingPolicy: "REJECT_CAPTURE"},
	{id: "cli.exit.code", channel: "exit", path: []string{"completion", "code"}, valueKind: "SAFE_INTEGER", missingPolicy: "TAGGED_MISSING_FOR_SIGNAL"},
	{id: "cli.exit.signal", channel: "exit", path: []string{"completion", "signal"}, valueKind: "UTF8_STRING", missingPolicy: "TAGGED_MISSING_FOR_EXIT"},
	{id: "cli.stdout.bytes", channel: "stdout", path: []string{"bytes"}, valueKind: "BYTES", missingPolicy: "REJECT_CHANNEL"},
	{id: "cli.stderr.text", channel: "stderr", path: []string{"utf8_text"}, valueKind: "UTF8_STRING", missingPolicy: "REJECT_CHANNEL"},
	{id: "cli.stdout.json.mode", channel: "stdout", path: []string{"strict_json", "mode"}, valueKind: "UTF8_STRING", missingPolicy: "TAGGED_MISSING"},
	{id: "cli.stdout.json.source", channel: "stdout", path: []string{"strict_json", "source"}, valueKind: "UTF8_STRING", missingPolicy: "TAGGED_MISSING"},
}

type closedCLIFieldIdentity struct {
	FieldID       string   `json:"field_id"`
	Channel       string   `json:"channel"`
	Path          []string `json:"path"`
	ValueKind     string   `json:"value_kind"`
	MissingPolicy string   `json:"missing_policy"`
}

type closedCLIProjectionConfigIdentity struct {
	SchemaVersion string   `json:"schema_version"`
	Kind          string   `json:"kind"`
	Version       string   `json:"version"`
	Fields        []string `json:"fields"`
}

// NewCLIProjectionAuthority derives the only closed v1 CLI adapter
// projection authority accepted by the cycle-free model edge. It is the
// security boundary behind ResolveCLIProjectionAuthority, not a digest parser.
func NewCLIProjectionAuthority(input []string) (CLIProjectionAuthority, error) {
	fields, err := normalizeClosedCLIFields(input)
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	registry := make([]closedCLIFieldIdentity, len(closedCLIFieldRegistryV1))
	for index, field := range closedCLIFieldRegistryV1 {
		registry[index] = closedCLIFieldIdentity{
			FieldID: field.id, Channel: field.channel, Path: append([]string(nil), field.path...),
			ValueKind: field.valueKind, MissingPolicy: field.missingPolicy,
		}
	}
	registryDigest, _, err := digestTyped("CLIFieldRegistry", struct {
		SchemaVersion string                   `json:"schema_version"`
		Kind          string                   `json:"kind"`
		Version       string                   `json:"version"`
		Fields        []closedCLIFieldIdentity `json:"fields"`
	}{domain.SchemaVersion, "CLIFieldRegistry", cliProjectionVersionV1, registry})
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	implementationDigest, err := closedProjectionDigestBytes(
		"CLIProjectionImplementation",
		[]byte(cliProjectionVersionV1+"\x00closed-field-registry\x00visible-pure-operations\x00exact-canonical"),
	)
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	configurationDigest, _, err := digestTyped("CLIProjectionConfiguration", closedCLIProjectionConfigIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionConfiguration",
		Version: cliProjectionVersionV1, Fields: fields,
	})
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	operations, err := closedCLIOperations(fields)
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	bindingOperations := make([]domain.ProjectionOperationBinding, len(operations))
	for index, operation := range operations {
		rule, parseErr := domain.ParseDigest(operation.RuleDigest)
		if parseErr != nil {
			return CLIProjectionAuthority{}, parseErr
		}
		bindingOperations[index] = domain.ProjectionOperationBinding{Name: operation.Name, RuleDigest: rule}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest, AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations: bindingOperations, Comparator: domain.ProjectionComparatorExact,
		FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	identity := resolvedProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionDefinition", Version: cliProjectionVersionV1,
		Fields: fields, Operations: operations, FieldRegistryDigest: registryDigest.String(),
		ImplementationDigest: implementationDigest.String(), ConfigurationDigest: configurationDigest.String(),
		ProjectionBindingDigest: binding.Digest().String(),
	}
	digest, canonicalBytes, err := digestTyped("CLIProjectionDefinition", identity)
	if err != nil {
		return CLIProjectionAuthority{}, err
	}
	return CLIProjectionAuthority{
		digest: digest, canonicalBytes: canonicalBytes, binding: binding, fieldIDs: append([]string(nil), fields...),
	}, nil
}

func normalizeClosedCLIFields(input []string) ([]string, error) {
	if len(input) == 0 || len(input) > len(closedCLIFieldRegistryV1) {
		return nil, refuse(CodeInvalidBinding, "CLI projection requires a nonempty closed field set")
	}
	selected := make(map[string]bool, len(input))
	for _, field := range input {
		if selected[field] || closedCLIField(field) == nil {
			return nil, refuse(CodeInvalidBinding, "CLI projection contains an unsupported or duplicate field")
		}
		selected[field] = true
	}
	result := make([]string, 0, len(input))
	for _, field := range closedCLIFieldRegistryV1 {
		if selected[field.id] {
			result = append(result, field.id)
		}
	}
	return result, nil
}

func closedCLIField(id string) *closedCLIFieldDescriptor {
	for index := range closedCLIFieldRegistryV1 {
		if closedCLIFieldRegistryV1[index].id == id {
			return &closedCLIFieldRegistryV1[index]
		}
	}
	return nil
}

func closedCLIOperations(fields []string) ([]resolvedProjectionOperation, error) {
	result := make([]resolvedProjectionOperation, 0, 10+len(fields))
	add := func(name, semantics string) error {
		digest, err := closedProjectionDigestBytes("CLIProjectionOperationRule", []byte(name+"\x00"+semantics))
		if err != nil {
			return err
		}
		result = append(result, resolvedProjectionOperation{Name: name, Semantics: semantics, RuleDigest: digest.String()})
		return nil
	}
	if err := add("cli.require-eligible-capture/v1", "require no controls, present completion, complete stdout and stderr, present fixture overlay receipt, and validated fixture invocation receipt"); err != nil {
		return nil, err
	}
	if anyClosedCLIChannel(fields, "exit") {
		if err := add("cli.require-completion/v1", "require the disjoint EXITED-or-SIGNALED completion sum"); err != nil {
			return nil, err
		}
	}
	if anyClosedCLIChannel(fields, "stdout") {
		if err := add("cli.require-stdout/v1", "require complete stdout while preserving present-empty"); err != nil {
			return nil, err
		}
	}
	if anyClosedCLIChannel(fields, "stderr") {
		if err := add("cli.require-stderr/v1", "require complete stderr while preserving present-empty"); err != nil {
			return nil, err
		}
	}
	if containsClosedCLIField(fields, "cli.stderr.text") {
		if err := add("cli.validate-stderr-utf8/v1", "validate stderr bytes as strict UTF-8 without trimming or replacement"); err != nil {
			return nil, err
		}
	}
	if containsClosedCLIField(fields, "cli.stdout.json.mode") || containsClosedCLIField(fields, "cli.stdout.json.source") {
		if err := add("cli.validate-stdout-utf8/v1", "validate stdout bytes as strict UTF-8 without trimming or replacement"); err != nil {
			return nil, err
		}
		if err := add("cli.parse-stdout-strict-json/v1", "parse one strict canonical-profile JSON object; reject duplicates, coercion, and trailing data"); err != nil {
			return nil, err
		}
	}
	for _, field := range fields {
		if err := add("cli.select-"+closedCLIOperationSuffix(field)+"/v1", "select the one closed typed field and retain tagged missing/present-empty semantics"); err != nil {
			return nil, err
		}
	}
	if err := add("cli.encode-tagged-fields-canonical/v1", "encode registry-ordered field IDs and exact tagged values with the Countershape canonical profile"); err != nil {
		return nil, err
	}
	return result, nil
}

func anyClosedCLIChannel(fields []string, channel string) bool {
	for _, field := range fields {
		if descriptor := closedCLIField(field); descriptor != nil && descriptor.channel == channel {
			return true
		}
	}
	return false
}

func containsClosedCLIField(fields []string, target string) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}

func closedCLIOperationSuffix(field string) string {
	switch field {
	case "cli.completion.kind":
		return "completion-kind"
	case "cli.exit.code":
		return "exit-code"
	case "cli.exit.signal":
		return "exit-signal"
	case "cli.stdout.bytes":
		return "stdout-bytes"
	case "cli.stderr.text":
		return "stderr-text"
	case "cli.stdout.json.mode":
		return "stdout-json-mode"
	default:
		return "stdout-json-source"
	}
}

func closedProjectionDigestBytes(kind string, value []byte) (domain.Digest, error) {
	digest, err := canon.DigestBytes(kind, value)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}
