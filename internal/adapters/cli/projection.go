package cli

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const cliProjectionVersion = "cli-projection/v1"

type CLIFieldID string

const (
	CLIFieldCompletionKind            CLIFieldID = "cli.completion.kind"
	CLIFieldExitKind                  CLIFieldID = CLIFieldCompletionKind
	CLIFieldExitCode                  CLIFieldID = "cli.exit.code"
	CLIFieldExitSignal                CLIFieldID = "cli.exit.signal"
	CLIFieldStdoutBytes               CLIFieldID = "cli.stdout.bytes"
	CLIFieldStderrText                CLIFieldID = "cli.stderr.text"
	CLIFieldStdoutJSONMode            CLIFieldID = "cli.stdout.json.mode"
	CLIFieldStdoutJSONPrecedenceValue CLIFieldID = CLIFieldStdoutJSONMode
	CLIFieldStdoutJSONSource          CLIFieldID = "cli.stdout.json.source"
)

type CLIFieldValueKind string

const (
	CLIValueString      CLIFieldValueKind = "UTF8_STRING"
	CLIValueSafeInteger CLIFieldValueKind = "SAFE_INTEGER"
	CLIValueBytes       CLIFieldValueKind = "BYTES"
)

type CLIFieldDescriptor struct {
	id            CLIFieldID
	channel       CLIChannel
	path          []string
	valueKind     CLIFieldValueKind
	missingPolicy string
}

func (d CLIFieldDescriptor) ID() CLIFieldID               { return d.id }
func (d CLIFieldDescriptor) Channel() CLIChannel          { return d.channel }
func (d CLIFieldDescriptor) Path() []string               { return append([]string(nil), d.path...) }
func (d CLIFieldDescriptor) ValueKind() CLIFieldValueKind { return d.valueKind }
func (d CLIFieldDescriptor) MissingPolicy() string        { return d.missingPolicy }

var cliFieldRegistry = []CLIFieldDescriptor{
	{id: CLIFieldCompletionKind, channel: CLIChannelExit, path: []string{"completion", "kind"}, valueKind: CLIValueString, missingPolicy: "REJECT_CAPTURE"},
	{id: CLIFieldExitCode, channel: CLIChannelExit, path: []string{"completion", "code"}, valueKind: CLIValueSafeInteger, missingPolicy: "TAGGED_MISSING_FOR_SIGNAL"},
	{id: CLIFieldExitSignal, channel: CLIChannelExit, path: []string{"completion", "signal"}, valueKind: CLIValueString, missingPolicy: "TAGGED_MISSING_FOR_EXIT"},
	{id: CLIFieldStdoutBytes, channel: CLIChannelStdout, path: []string{"bytes"}, valueKind: CLIValueBytes, missingPolicy: "REJECT_CHANNEL"},
	{id: CLIFieldStderrText, channel: CLIChannelStderr, path: []string{"utf8_text"}, valueKind: CLIValueString, missingPolicy: "REJECT_CHANNEL"},
	{id: CLIFieldStdoutJSONMode, channel: CLIChannelStdout, path: []string{"strict_json", "mode"}, valueKind: CLIValueString, missingPolicy: "TAGGED_MISSING"},
	{id: CLIFieldStdoutJSONSource, channel: CLIChannelStdout, path: []string{"strict_json", "source"}, valueKind: CLIValueString, missingPolicy: "TAGGED_MISSING"},
}

func CLIFieldRegistry() []CLIFieldDescriptor {
	result := make([]CLIFieldDescriptor, len(cliFieldRegistry))
	for index, descriptor := range cliFieldRegistry {
		result[index] = descriptor
		result[index].path = append([]string(nil), descriptor.path...)
	}
	return result
}

type CLIProjectionDefinitionConfig struct {
	Fields []CLIFieldID
}

type CLIProjectionOperation struct {
	name       string
	semantics  string
	ruleDigest domain.Digest
}

func (o CLIProjectionOperation) Name() string              { return o.name }
func (o CLIProjectionOperation) Semantics() string         { return o.semantics }
func (o CLIProjectionOperation) RuleDigest() domain.Digest { return o.ruleDigest }

type CLIProjectionDefinition struct {
	digest               domain.Digest
	canonicalBytes       []byte
	fields               []CLIFieldID
	operations           []CLIProjectionOperation
	registryDigest       domain.Digest
	implementationDigest domain.Digest
	configurationDigest  domain.Digest
	binding              domain.ProjectionDefinitionBinding
}

type fieldRegistryIdentity struct {
	FieldID       string   `json:"field_id"`
	Channel       string   `json:"channel"`
	Path          []string `json:"path"`
	ValueKind     string   `json:"value_kind"`
	MissingPolicy string   `json:"missing_policy"`
}

type projectionConfigIdentity struct {
	SchemaVersion string   `json:"schema_version"`
	Kind          string   `json:"kind"`
	Version       string   `json:"version"`
	Fields        []string `json:"fields"`
}

type projectionOperationIdentity struct {
	Name       string `json:"name"`
	Semantics  string `json:"semantics"`
	RuleDigest string `json:"rule_digest"`
}

type cliProjectionDefinitionIdentity struct {
	SchemaVersion           string                        `json:"schema_version"`
	Kind                    string                        `json:"kind"`
	Version                 string                        `json:"version"`
	Fields                  []string                      `json:"fields"`
	Operations              []projectionOperationIdentity `json:"operations"`
	FieldRegistryDigest     string                        `json:"field_registry_digest"`
	ImplementationDigest    string                        `json:"implementation_digest"`
	ConfigurationDigest     string                        `json:"configuration_digest"`
	ProjectionBindingDigest string                        `json:"projection_definition_binding_digest"`
}

func NewCLIProjectionDefinition(config CLIProjectionDefinitionConfig) (CLIProjectionDefinition, error) {
	fields, err := normalizeProjectionFields(config.Fields)
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	registryDigest, err := fieldRegistryDigest()
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	implementationDigest, _, err := cliDigestBytes("CLIProjectionImplementation", []byte(cliProjectionVersion+"\x00closed-field-registry\x00visible-pure-operations\x00exact-canonical"))
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	fieldNames := make([]string, len(fields))
	for index, field := range fields {
		fieldNames[index] = string(field)
	}
	configurationDigest, _, err := cliDigestTyped("CLIProjectionConfiguration", projectionConfigIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionConfiguration", Version: cliProjectionVersion, Fields: fieldNames,
	})
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	operations, err := projectionOperations(fields)
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	acceptedChannels := acceptedProjectionChannels(fields)
	domainOperations := make([]domain.ProjectionOperationBinding, len(operations))
	operationIDs := make([]projectionOperationIdentity, len(operations))
	for index, operation := range operations {
		domainOperations[index] = domain.ProjectionOperationBinding{Name: operation.name, RuleDigest: operation.ruleDigest}
		operationIDs[index] = projectionOperationIdentity{
			Name: operation.name, Semantics: operation.semantics, RuleDigest: operation.ruleDigest.String(),
		}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest, AcceptedChannels: acceptedChannels,
		Operations: domainOperations, Comparator: domain.ProjectionComparatorExact,
		FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	identity := cliProjectionDefinitionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionDefinition", Version: cliProjectionVersion,
		Fields: fieldNames, Operations: operationIDs, FieldRegistryDigest: registryDigest.String(),
		ImplementationDigest: implementationDigest.String(), ConfigurationDigest: configurationDigest.String(),
		ProjectionBindingDigest: binding.Digest().String(),
	}
	digest, canonicalBytes, err := cliDigestTyped("CLIProjectionDefinition", identity)
	if err != nil {
		return CLIProjectionDefinition{}, err
	}
	// The adapter-owned definition and generic capability intentionally have
	// distinct addresses. The latter is the authority consumed by WorldPlan.
	return CLIProjectionDefinition{
		digest: digest, canonicalBytes: canonicalBytes, fields: fields, operations: operations,
		registryDigest: registryDigest, implementationDigest: implementationDigest,
		configurationDigest: configurationDigest, binding: binding,
	}, nil
}

func normalizeProjectionFields(input []CLIFieldID) ([]CLIFieldID, error) {
	if len(input) == 0 || len(input) > len(cliFieldRegistry) {
		return nil, refuse(CodeInvalidProjection, "projection requires a nonempty closed field set")
	}
	selected := make(map[CLIFieldID]bool, len(input))
	for _, field := range input {
		if _, ok := descriptorFor(field); !ok || selected[field] {
			return nil, refuse(CodeInvalidProjection, "projection contains an unsupported or duplicate field")
		}
		selected[field] = true
	}
	result := make([]CLIFieldID, 0, len(input))
	for _, descriptor := range cliFieldRegistry {
		if selected[descriptor.id] {
			result = append(result, descriptor.id)
		}
	}
	return result, nil
}

func descriptorFor(id CLIFieldID) (CLIFieldDescriptor, bool) {
	for _, descriptor := range cliFieldRegistry {
		if descriptor.id == id {
			return descriptor, true
		}
	}
	return CLIFieldDescriptor{}, false
}

func fieldRegistryDigest() (domain.Digest, error) {
	identities := make([]fieldRegistryIdentity, len(cliFieldRegistry))
	for index, descriptor := range cliFieldRegistry {
		identities[index] = fieldRegistryIdentity{
			FieldID: string(descriptor.id), Channel: string(descriptor.channel),
			Path: append([]string(nil), descriptor.path...), ValueKind: string(descriptor.valueKind),
			MissingPolicy: descriptor.missingPolicy,
		}
	}
	digest, _, err := cliDigestTyped("CLIFieldRegistry", struct {
		SchemaVersion string                  `json:"schema_version"`
		Kind          string                  `json:"kind"`
		Version       string                  `json:"version"`
		Fields        []fieldRegistryIdentity `json:"fields"`
	}{domain.SchemaVersion, "CLIFieldRegistry", cliProjectionVersion, identities})
	return digest, err
}

func projectionOperations(fields []CLIFieldID) ([]CLIProjectionOperation, error) {
	result := make([]CLIProjectionOperation, 0, 10+len(fields))
	add := func(name, semantics string) error {
		digest, _, err := cliDigestBytes("CLIProjectionOperationRule", []byte(name+"\x00"+semantics))
		if err != nil {
			return err
		}
		result = append(result, CLIProjectionOperation{name: name, semantics: semantics, ruleDigest: digest})
		return nil
	}
	if err := add("cli.require-eligible-capture/v1", "require no controls, present completion, complete stdout and stderr, present fixture overlay receipt, and validated fixture invocation receipt"); err != nil {
		return nil, err
	}
	if anyExitField(fields) {
		if err := add("cli.require-completion/v1", "require the disjoint EXITED-or-SIGNALED completion sum"); err != nil {
			return nil, err
		}
	}
	if anyChannelField(fields, CLIChannelStdout) {
		if err := add("cli.require-stdout/v1", "require complete stdout while preserving present-empty"); err != nil {
			return nil, err
		}
	}
	if anyChannelField(fields, CLIChannelStderr) {
		if err := add("cli.require-stderr/v1", "require complete stderr while preserving present-empty"); err != nil {
			return nil, err
		}
	}
	if hasField(fields, CLIFieldStderrText) {
		if err := add("cli.validate-stderr-utf8/v1", "validate stderr bytes as strict UTF-8 without trimming or replacement"); err != nil {
			return nil, err
		}
	}
	if anyJSONField(fields) {
		if err := add("cli.validate-stdout-utf8/v1", "validate stdout bytes as strict UTF-8 without trimming or replacement"); err != nil {
			return nil, err
		}
		if err := add("cli.parse-stdout-strict-json/v1", "parse one strict canonical-profile JSON object; reject duplicates, coercion, and trailing data"); err != nil {
			return nil, err
		}
	}
	for _, field := range fields {
		if err := add("cli.select-"+operationSuffix(field)+"/v1", "select the one closed typed field and retain tagged missing/present-empty semantics"); err != nil {
			return nil, err
		}
	}
	if err := add("cli.encode-tagged-fields-canonical/v1", "encode registry-ordered field IDs and exact tagged values with the Countershape canonical profile"); err != nil {
		return nil, err
	}
	return result, nil
}

func operationSuffix(field CLIFieldID) string {
	switch field {
	case CLIFieldCompletionKind:
		return "completion-kind"
	case CLIFieldExitCode:
		return "exit-code"
	case CLIFieldExitSignal:
		return "exit-signal"
	case CLIFieldStdoutBytes:
		return "stdout-bytes"
	case CLIFieldStderrText:
		return "stderr-text"
	case CLIFieldStdoutJSONMode:
		return "stdout-json-mode"
	default:
		return "stdout-json-source"
	}
}

func acceptedProjectionChannels(fields []CLIFieldID) []string {
	_ = fields
	// The first eligibility operation reads all three channels for every field
	// selection, so the generic binding must advertise that complete semantic
	// input set rather than only the fields eventually emitted.
	return []string{string(CLIChannelExit), string(CLIChannelStderr), string(CLIChannelStdout)}
}

func anyExitField(fields []CLIFieldID) bool { return anyChannelField(fields, CLIChannelExit) }
func anyChannelField(fields []CLIFieldID, channel CLIChannel) bool {
	for _, field := range fields {
		descriptor, _ := descriptorFor(field)
		if descriptor.channel == channel {
			return true
		}
	}
	return false
}
func anyJSONField(fields []CLIFieldID) bool {
	return hasField(fields, CLIFieldStdoutJSONMode) || hasField(fields, CLIFieldStdoutJSONSource)
}
func hasField(fields []CLIFieldID, target CLIFieldID) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}

func (d CLIProjectionDefinition) Valid() bool {
	rebuilt, err := NewCLIProjectionDefinition(CLIProjectionDefinitionConfig{Fields: d.fields})
	return err == nil && rebuilt.digest == d.digest && bytes.Equal(rebuilt.canonicalBytes, d.canonicalBytes) &&
		rebuilt.binding.Digest() == d.binding.Digest()
}

func (d CLIProjectionDefinition) Digest() domain.Digest { return d.digest }
func (d CLIProjectionDefinition) CanonicalBytes() []byte {
	return append([]byte(nil), d.canonicalBytes...)
}
func (d CLIProjectionDefinition) Fields() []CLIFieldID { return append([]CLIFieldID(nil), d.fields...) }
func (d CLIProjectionDefinition) Operations() []CLIProjectionOperation {
	return append([]CLIProjectionOperation(nil), d.operations...)
}
func (d CLIProjectionDefinition) FieldRegistryDigest() domain.Digest          { return d.registryDigest }
func (d CLIProjectionDefinition) Binding() domain.ProjectionDefinitionBinding { return d.binding }

type CLIExactValueTag string

const (
	ExactMissing CLIExactValueTag = "MISSING"
	ExactString  CLIExactValueTag = "STRING"
	ExactInteger CLIExactValueTag = "INTEGER"
	ExactBytes   CLIExactValueTag = "BYTES"
)

type CLIExactValue struct {
	tag       CLIExactValueTag
	text      string
	integer   int64
	bytes     []byte
	canonical []byte
}

func (v CLIExactValue) Tag() CLIExactValueTag  { return v.tag }
func (v CLIExactValue) String() (string, bool) { return v.text, v.tag == ExactString }
func (v CLIExactValue) Integer() (int64, bool) { return v.integer, v.tag == ExactInteger }
func (v CLIExactValue) Bytes() ([]byte, bool) {
	return append([]byte(nil), v.bytes...), v.tag == ExactBytes
}
func (v CLIExactValue) CanonicalBytes() []byte { return append([]byte(nil), v.canonical...) }

type CLIProjectedField struct {
	id    CLIFieldID
	value CLIExactValue
}

func (f CLIProjectedField) ID() CLIFieldID       { return f.id }
func (f CLIProjectedField) Value() CLIExactValue { return cloneExactValue(f.value) }

type CLIProjectionSourceLink struct {
	channel  CLIChannel
	selector []string
}

func (l CLIProjectionSourceLink) Channel() CLIChannel { return l.channel }
func (l CLIProjectionSourceLink) Selector() []string  { return append([]string(nil), l.selector...) }

type CLIProjectionTraceEntry struct {
	operation   CLIProjectionOperation
	sourceLinks []CLIProjectionSourceLink
}

func (e CLIProjectionTraceEntry) Operation() CLIProjectionOperation { return e.operation }
func (e CLIProjectionTraceEntry) SourceLinks() []CLIProjectionSourceLink {
	return cloneSourceLinks(e.sourceLinks)
}

type sourceLinkIdentity struct {
	Channel  string   `json:"channel"`
	Selector []string `json:"selector"`
}

type traceEntryIdentity struct {
	OperationName      string               `json:"operation_name"`
	OperationSemantics string               `json:"operation_semantics"`
	RuleDigest         string               `json:"rule_digest"`
	SourceLinks        []sourceLinkIdentity `json:"source_links"`
}

type projectionRejectionEvidenceIdentity struct {
	SchemaVersion           string               `json:"schema_version"`
	Kind                    string               `json:"kind"`
	ObservationDigest       string               `json:"captured_observation_digest"`
	AdapterDefinitionDigest string               `json:"cli_adapter_projection_definition_digest"`
	DefinitionDigest        string               `json:"cli_projection_definition_digest"`
	Code                    string               `json:"code"`
	Operation               string               `json:"operation"`
	Channel                 string               `json:"channel"`
	Field                   string               `json:"field"`
	Transcript              []traceEntryIdentity `json:"transcript"`
}

type CLIProjectionDerivation struct {
	digest                  domain.Digest
	canonicalBytes          []byte
	observationDigest       domain.Digest
	adapterDefinitionDigest domain.Digest
	definitionDigest        domain.Digest
	projectionByteDigest    domain.Digest
	definitionFields        []CLIFieldID
	transcript              []CLIProjectionTraceEntry
}

func (d CLIProjectionDerivation) Valid() bool {
	if !d.digest.Valid() || !d.observationDigest.Valid() || !d.adapterDefinitionDigest.Valid() ||
		!d.definitionDigest.Valid() || !d.projectionByteDigest.Valid() || len(d.canonicalBytes) == 0 {
		return false
	}
	definition, err := NewCLIProjectionDefinition(CLIProjectionDefinitionConfig{Fields: d.definitionFields})
	if err != nil || definition.digest != d.adapterDefinitionDigest ||
		definition.binding.Digest() != d.definitionDigest ||
		!transcriptMatchesDefinition(definition, d.transcript) {
		return false
	}
	digest, canonicalBytes, err := digestDerivation(d.observationDigest, d.adapterDefinitionDigest, d.definitionDigest, d.projectionByteDigest, d.transcript)
	return err == nil && digest == d.digest && bytes.Equal(canonicalBytes, d.canonicalBytes)
}
func (d CLIProjectionDerivation) Digest() domain.Digest { return d.digest }
func (d CLIProjectionDerivation) CanonicalBytes() []byte {
	return append([]byte(nil), d.canonicalBytes...)
}
func (d CLIProjectionDerivation) ObservationDigest() domain.Digest { return d.observationDigest }
func (d CLIProjectionDerivation) DefinitionDigest() domain.Digest  { return d.definitionDigest }
func (d CLIProjectionDerivation) AdapterDefinitionDigest() domain.Digest {
	return d.adapterDefinitionDigest
}
func (d CLIProjectionDerivation) ProjectionByteDigest() domain.Digest { return d.projectionByteDigest }
func (d CLIProjectionDerivation) Transcript() []CLIProjectionTraceEntry {
	return cloneTranscript(d.transcript)
}

type CLIProjectionResult struct {
	projectionBytes []byte
	canonicalBytes  []byte
	fields          []CLIProjectedField
	derivation      CLIProjectionDerivation
}

// ProjectionBytes are the exact behavior bytes consumed by
// ProjectionFingerprint. CanonicalBytes are distinct adapter derivation
// evidence and include the complete ordered operation/source-link transcript.
func (r CLIProjectionResult) ProjectionBytes() []byte {
	return append([]byte(nil), r.projectionBytes...)
}
func (r CLIProjectionResult) CanonicalBytes() []byte      { return append([]byte(nil), r.canonicalBytes...) }
func (r CLIProjectionResult) Fields() []CLIProjectedField { return cloneProjectedFields(r.fields) }
func (r CLIProjectionResult) Derivation() CLIProjectionDerivation {
	return cloneDerivation(r.derivation)
}
func (r CLIProjectionResult) DerivationDigest() domain.Digest { return r.derivation.digest }
func (r CLIProjectionResult) Transcript() []CLIProjectionTraceEntry {
	return cloneTranscript(r.derivation.transcript)
}
func (r CLIProjectionResult) Operations() []CLIProjectionOperation {
	result := make([]CLIProjectionOperation, len(r.derivation.transcript))
	for index, entry := range r.derivation.transcript {
		result[index] = entry.operation
	}
	return result
}
func (r CLIProjectionResult) SourceLinks() []CLIProjectionSourceLink {
	result := []CLIProjectionSourceLink{}
	for _, entry := range r.derivation.transcript {
		result = append(result, cloneSourceLinks(entry.sourceLinks)...)
	}
	return result
}

func (d CLIProjectionDefinition) Project(observation CLICapturedObservation) (CLIProjectionResult, error) {
	if !d.Valid() || !observation.Valid() {
		return CLIProjectionResult{}, refuse(CodeProjectionInternal, "definition or observation authority is invalid")
	}
	if observation.projectionDefinitionDigest != d.binding.Digest() {
		return CLIProjectionResult{}, refuse(CodeProjectionInternal, "capture and projection definition lineage differ")
	}
	if observation.adapterProjectionDefinitionDigest != d.digest {
		return CLIProjectionResult{}, refuse(CodeProjectionInternal, "capture and adapter-owned projection authority differ")
	}
	transcript := make([]CLIProjectionTraceEntry, 0, len(d.operations))
	operationIndex := 0
	appendOperation := func(links ...CLIProjectionSourceLink) CLIProjectionOperation {
		operation := d.operations[operationIndex]
		operationIndex++
		// MUTATION_ANCHOR: hidden-projection-operation
		transcript = append(transcript, CLIProjectionTraceEntry{operation: operation, sourceLinks: cloneSourceLinks(links)})
		return operation
	}

	operation := appendOperation(
		sourceLink("", "controls"),
		sourceLink(CLIChannelExit, "completion"),
		sourceLink(CLIChannelStdout, "state"),
		sourceLink(CLIChannelStderr, "state"),
		sourceLink("", "fixture_overlay_receipt"),
		sourceLink("", "fixture_invocation_receipt"),
	)
	if !observation.projectionEligibilityFactsPresent() {
		return CLIProjectionResult{}, d.rejection(observation, CodeProjectionControl, operation.name, "", "", transcript, "capture is not eligible behavior")
	}
	if anyExitField(d.fields) {
		operation = appendOperation(sourceLink(CLIChannelExit, "completion"))
		if !observation.completionPresent || !observation.completion.valid() {
			return CLIProjectionResult{}, d.rejection(observation, CodeProjectionChannel, operation.name, CLIChannelExit, "", transcript, "completion is absent")
		}
	}
	if anyChannelField(d.fields, CLIChannelStdout) {
		operation = appendOperation(sourceLink(CLIChannelStdout, "state"), sourceLink(CLIChannelStdout, "captured_bytes"))
		if rejection := requirePresentChannel(observation.stdout, operation.name); rejection != nil {
			return CLIProjectionResult{}, d.finishRejection(observation, rejection, transcript)
		}
	}
	if anyChannelField(d.fields, CLIChannelStderr) {
		operation = appendOperation(sourceLink(CLIChannelStderr, "state"), sourceLink(CLIChannelStderr, "captured_bytes"))
		if rejection := requirePresentChannel(observation.stderr, operation.name); rejection != nil {
			return CLIProjectionResult{}, d.finishRejection(observation, rejection, transcript)
		}
	}
	if hasField(d.fields, CLIFieldStderrText) {
		operation = appendOperation(sourceLink(CLIChannelStderr, "captured_bytes"))
		if !utf8.Valid(observation.stderr.bytes) {
			return CLIProjectionResult{}, d.rejection(observation, CodeProjectionUTF8, operation.name, CLIChannelStderr, CLIFieldStderrText, transcript, "stderr is not valid UTF-8")
		}
	}
	var stdoutJSON canon.Value
	if anyJSONField(d.fields) {
		operation = appendOperation(sourceLink(CLIChannelStdout, "captured_bytes"))
		if !utf8.Valid(observation.stdout.bytes) {
			return CLIProjectionResult{}, d.rejection(observation, CodeProjectionUTF8, operation.name, CLIChannelStdout, "", transcript, "stdout is not valid UTF-8")
		}
		operation = appendOperation(sourceLink(CLIChannelStdout, "captured_bytes"))
		parsed, err := canon.Parse(observation.stdout.bytes)
		if err != nil {
			return CLIProjectionResult{}, d.rejection(observation, CodeProjectionJSON, operation.name, CLIChannelStdout, "", transcript, err.Error())
		}
		if parsed.Kind() != canon.KindObject {
			return CLIProjectionResult{}, d.rejection(observation, CodeProjectionJSONRoot, operation.name, CLIChannelStdout, "", transcript, "stdout strict JSON root is not an object")
		}
		stdoutJSON = parsed
	}

	projected := make([]CLIProjectedField, 0, len(d.fields))
	for _, field := range d.fields {
		descriptor, _ := descriptorFor(field)
		links := []CLIProjectionSourceLink{sourceLink(descriptor.channel, descriptor.path...)}
		operation = appendOperation(links...)
		value, rejection := projectField(field, observation, stdoutJSON, operation.name)
		if rejection != nil {
			return CLIProjectionResult{}, d.finishRejection(observation, rejection, transcript)
		}
		projected = append(projected, CLIProjectedField{id: field, value: value})
	}
	operation = appendOperation()
	if !transcriptMatchesDefinition(d, transcript) {
		return CLIProjectionResult{}, d.rejection(observation, CodeProjectionInternal, operation.name, "", "", transcript, "projection transcript differs from the sealed definition")
	}
	canonicalProjection, err := encodeProjection(projected)
	if err != nil {
		return CLIProjectionResult{}, d.rejection(observation, CodeProjectionResourceLimit, operation.name, "", "", transcript, err.Error())
	}
	projectionByteDigest, _, err := cliDigestBytes("CLIProjectionCanonicalBytes", canonicalProjection)
	if err != nil {
		return CLIProjectionResult{}, d.rejection(observation, CodeProjectionInternal, operation.name, "", "", transcript, err.Error())
	}
	derivationDigest, derivationBytes, err := digestDerivation(observation.digest, d.digest, d.binding.Digest(), projectionByteDigest, transcript)
	if err != nil {
		return CLIProjectionResult{}, d.rejection(observation, CodeProjectionInternal, operation.name, "", "", transcript, err.Error())
	}
	derivation := CLIProjectionDerivation{
		digest: derivationDigest, canonicalBytes: derivationBytes, observationDigest: observation.digest,
		adapterDefinitionDigest: d.digest, definitionDigest: d.binding.Digest(),
		projectionByteDigest: projectionByteDigest, definitionFields: append([]CLIFieldID(nil), d.fields...),
		transcript: cloneTranscript(transcript),
	}
	return CLIProjectionResult{
		projectionBytes: canonicalProjection, canonicalBytes: append([]byte(nil), derivationBytes...),
		fields: cloneProjectedFields(projected), derivation: derivation,
	}, nil
}

func requirePresentChannel(channel CLIChannelCapture, operation string) *ProjectionRejection {
	switch channel.state {
	case ChannelPresent:
		return nil
	case ChannelTruncated:
		return &ProjectionRejection{Code: CodeProjectionTruncated, Operation: operation, Channel: channel.name, Detail: "truncated capture can never project"}
	default:
		return &ProjectionRejection{Code: CodeProjectionChannel, Operation: operation, Channel: channel.name, Detail: "required channel is absent"}
	}
}

func projectField(field CLIFieldID, observation CLICapturedObservation, stdoutJSON canon.Value, operation string) (CLIExactValue, *ProjectionRejection) {
	switch field {
	case CLIFieldCompletionKind:
		return exactString(string(observation.completion.kind), operation, CLIChannelExit, field)
	case CLIFieldExitCode:
		if observation.completion.kind != CompletionExited {
			return exactMissing(operation, CLIChannelExit, field)
		}
		return exactInteger(int64(observation.completion.code), operation, CLIChannelExit, field)
	case CLIFieldExitSignal:
		if observation.completion.kind != CompletionSignaled {
			return exactMissing(operation, CLIChannelExit, field)
		}
		return exactString(observation.completion.signal, operation, CLIChannelExit, field)
	case CLIFieldStdoutBytes:
		return exactBytes(observation.stdout.bytes, operation, CLIChannelStdout, field)
	case CLIFieldStderrText:
		return exactString(string(observation.stderr.bytes), operation, CLIChannelStderr, field)
	case CLIFieldStdoutJSONMode, CLIFieldStdoutJSONSource:
		member := "mode"
		if field == CLIFieldStdoutJSONSource {
			member = "source"
		}
		value, present := stdoutJSON.LookupMember(member)
		if !present {
			return exactMissing(operation, CLIChannelStdout, field)
		}
		text, ok := value.Text()
		if !ok {
			return CLIExactValue{}, &ProjectionRejection{
				Code: CodeProjectionJSONFieldType, Operation: operation, Channel: CLIChannelStdout,
				Field: field, Detail: "selected strict-JSON member is not a string",
			}
		}
		return exactString(text, operation, CLIChannelStdout, field)
	default:
		return CLIExactValue{}, &ProjectionRejection{Code: CodeProjectionInternal, Operation: operation, Field: field, Detail: "field escaped the closed registry"}
	}
}

func exactMissing(operation string, channel CLIChannel, field CLIFieldID) (CLIExactValue, *ProjectionRejection) {
	tag, err := canon.String(string(ExactMissing))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	value, err := canon.Object(canon.Member{Name: "tag", Value: tag})
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	return CLIExactValue{tag: ExactMissing, canonical: canonical}, nil
}

func exactString(text, operation string, channel CLIChannel, field CLIFieldID) (CLIExactValue, *ProjectionRejection) {
	tag, err := canon.String(string(ExactString))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	textValue, err := canon.String(text)
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	value, err := canon.Object(
		canon.Member{Name: "tag", Value: tag},
		canon.Member{Name: "value", Value: textValue},
	)
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	return CLIExactValue{tag: ExactString, text: text, canonical: canonical}, nil
}

func exactInteger(integer int64, operation string, channel CLIChannel, field CLIFieldID) (CLIExactValue, *ProjectionRejection) {
	tag, err := canon.String(string(ExactInteger))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	canonicalInteger, err := canon.String(strconv.FormatInt(integer, 10))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	value, err := canon.Object(
		canon.Member{Name: "tag", Value: tag},
		canon.Member{Name: "canonical", Value: canonicalInteger},
	)
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	return CLIExactValue{tag: ExactInteger, integer: integer, canonical: canonical}, nil
}

func exactBytes(valueBytes []byte, operation string, channel CLIChannel, field CLIFieldID) (CLIExactValue, *ProjectionRejection) {
	tag, err := canon.String(string(ExactBytes))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	encoded, err := canon.String(base64.StdEncoding.EncodeToString(valueBytes))
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	value, err := canon.Object(
		canon.Member{Name: "tag", Value: tag},
		canon.Member{Name: "base64", Value: encoded},
	)
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil {
		return exactValueFailure(err, operation, channel, field)
	}
	return CLIExactValue{tag: ExactBytes, bytes: append([]byte(nil), valueBytes...), canonical: canonical}, nil
}

func exactValueFailure(err error, operation string, channel CLIChannel, field CLIFieldID) (CLIExactValue, *ProjectionRejection) {
	return CLIExactValue{}, &ProjectionRejection{
		Code: CodeProjectionResourceLimit, Operation: operation, Channel: channel,
		Field: field, Detail: err.Error(),
	}
}

func encodeProjection(fields []CLIProjectedField) ([]byte, error) {
	fieldValues := make([]canon.Value, len(fields))
	for index, field := range fields {
		value, err := canon.Parse(field.value.canonical)
		if err != nil {
			return nil, err
		}
		fieldID, stringErr := canon.String(string(field.id))
		if stringErr != nil {
			return nil, stringErr
		}
		fieldValues[index], err = canon.Object(
			canon.Member{Name: "field_id", Value: fieldID},
			canon.Member{Name: "value", Value: value},
		)
		if err != nil {
			return nil, err
		}
	}
	array, err := canon.Array(fieldValues...)
	if err != nil {
		return nil, err
	}
	schemaVersion, err := canon.String(cliProjectionVersion)
	if err != nil {
		return nil, err
	}
	kind, err := canon.String("CLIProjection")
	if err != nil {
		return nil, err
	}
	root, err := canon.Object(
		canon.Member{Name: "schema_version", Value: schemaVersion},
		canon.Member{Name: "kind", Value: kind},
		canon.Member{Name: "fields", Value: array},
	)
	if err != nil {
		return nil, err
	}
	return root.CanonicalChecked()
}

func digestDerivation(observation, adapterDefinition, definition, projectionBytes domain.Digest, transcript []CLIProjectionTraceEntry) (domain.Digest, []byte, error) {
	identities := transcriptIdentities(transcript)
	return cliDigestTyped("CLIProjectionDerivation", struct {
		SchemaVersion           string               `json:"schema_version"`
		Kind                    string               `json:"kind"`
		ObservationDigest       string               `json:"captured_observation_digest"`
		AdapterDefinitionDigest string               `json:"cli_adapter_projection_definition_digest"`
		DefinitionDigest        string               `json:"cli_projection_definition_digest"`
		ProjectionByteDigest    string               `json:"projection_byte_digest"`
		Transcript              []traceEntryIdentity `json:"transcript"`
	}{domain.SchemaVersion, "CLIProjectionDerivation", observation.String(), adapterDefinition.String(), definition.String(), projectionBytes.String(), identities})
}

func transcriptIdentities(transcript []CLIProjectionTraceEntry) []traceEntryIdentity {
	result := make([]traceEntryIdentity, len(transcript))
	for index, entry := range transcript {
		links := make([]sourceLinkIdentity, len(entry.sourceLinks))
		for linkIndex, link := range entry.sourceLinks {
			links[linkIndex] = sourceLinkIdentity{Channel: string(link.channel), Selector: append([]string(nil), link.selector...)}
		}
		result[index] = traceEntryIdentity{
			OperationName: entry.operation.name, OperationSemantics: entry.operation.semantics,
			RuleDigest: entry.operation.ruleDigest.String(), SourceLinks: links,
		}
	}
	return result
}

func transcriptMatchesDefinition(definition CLIProjectionDefinition, transcript []CLIProjectionTraceEntry) bool {
	expected, complete := expectedProjectionTranscript(definition)
	if !complete || len(expected) != len(transcript) {
		return false
	}
	for index := range expected {
		if expected[index].operation.name != transcript[index].operation.name ||
			expected[index].operation.semantics != transcript[index].operation.semantics ||
			expected[index].operation.ruleDigest != transcript[index].operation.ruleDigest ||
			!sourceLinksEqual(expected[index].sourceLinks, transcript[index].sourceLinks) {
			return false
		}
	}
	return true
}

func expectedProjectionTranscript(definition CLIProjectionDefinition) ([]CLIProjectionTraceEntry, bool) {
	result := make([]CLIProjectionTraceEntry, 0, len(definition.operations))
	operationIndex := 0
	complete := true
	appendExpected := func(links ...CLIProjectionSourceLink) {
		if !complete || operationIndex >= len(definition.operations) {
			complete = false
			return
		}
		result = append(result, CLIProjectionTraceEntry{
			operation: definition.operations[operationIndex], sourceLinks: cloneSourceLinks(links),
		})
		operationIndex++
	}

	appendExpected(
		sourceLink("", "controls"),
		sourceLink(CLIChannelExit, "completion"),
		sourceLink(CLIChannelStdout, "state"),
		sourceLink(CLIChannelStderr, "state"),
		sourceLink("", "fixture_overlay_receipt"),
		sourceLink("", "fixture_invocation_receipt"),
	)
	if anyExitField(definition.fields) {
		appendExpected(sourceLink(CLIChannelExit, "completion"))
	}
	if anyChannelField(definition.fields, CLIChannelStdout) {
		appendExpected(sourceLink(CLIChannelStdout, "state"), sourceLink(CLIChannelStdout, "captured_bytes"))
	}
	if anyChannelField(definition.fields, CLIChannelStderr) {
		appendExpected(sourceLink(CLIChannelStderr, "state"), sourceLink(CLIChannelStderr, "captured_bytes"))
	}
	if hasField(definition.fields, CLIFieldStderrText) {
		appendExpected(sourceLink(CLIChannelStderr, "captured_bytes"))
	}
	if anyJSONField(definition.fields) {
		appendExpected(sourceLink(CLIChannelStdout, "captured_bytes"))
		appendExpected(sourceLink(CLIChannelStdout, "captured_bytes"))
	}
	for _, field := range definition.fields {
		descriptor, exists := descriptorFor(field)
		if !exists {
			return nil, false
		}
		appendExpected(sourceLink(descriptor.channel, descriptor.path...))
	}
	appendExpected()
	return result, complete && operationIndex == len(definition.operations)
}

func sourceLinksEqual(left, right []CLIProjectionSourceLink) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].channel != right[index].channel || len(left[index].selector) != len(right[index].selector) {
			return false
		}
		for selectorIndex := range left[index].selector {
			if left[index].selector[selectorIndex] != right[index].selector[selectorIndex] {
				return false
			}
		}
	}
	return true
}

func (d CLIProjectionDefinition) rejection(
	observation CLICapturedObservation,
	code, operation string,
	channel CLIChannel,
	field CLIFieldID,
	transcript []CLIProjectionTraceEntry,
	detail string,
) *ProjectionRejection {
	rejection := &ProjectionRejection{
		Code: code, Operation: operation, Channel: channel, Field: field, Detail: detail,
		observationDigest: observation.digest, adapterDefinitionDigest: d.digest,
		definitionDigest: d.binding.Digest(), transcript: cloneTranscript(transcript),
	}
	identity := projectionRejectionEvidenceIdentity{
		domain.SchemaVersion, "CLIProjectionRejectionEvidence", observation.digest.String(), d.digest.String(), d.binding.Digest().String(),
		code, operation, string(channel), string(field), transcriptIdentities(transcript),
	}
	digest, canonicalBytes, err := cliDigestTyped("CLIProjectionRejectionEvidence", identity)
	if err == nil {
		rejection.digest = digest
		rejection.canonicalBytes = canonicalBytes
	}
	return rejection
}

func (r *ProjectionRejection) Valid() bool {
	if r == nil || r.Code == "" || !r.digest.Valid() || !r.observationDigest.Valid() ||
		!r.adapterDefinitionDigest.Valid() || !r.definitionDigest.Valid() || len(r.canonicalBytes) == 0 {
		return false
	}
	identity := projectionRejectionEvidenceIdentity{
		domain.SchemaVersion, "CLIProjectionRejectionEvidence", r.observationDigest.String(),
		r.adapterDefinitionDigest.String(), r.definitionDigest.String(), r.Code, r.Operation,
		string(r.Channel), string(r.Field), transcriptIdentities(r.transcript),
	}
	digest, canonicalBytes, err := cliDigestTyped("CLIProjectionRejectionEvidence", identity)
	return err == nil && digest == r.digest && bytes.Equal(canonicalBytes, r.canonicalBytes)
}

func (d CLIProjectionDefinition) finishRejection(observation CLICapturedObservation, rejection *ProjectionRejection, transcript []CLIProjectionTraceEntry) *ProjectionRejection {
	return d.rejection(observation, rejection.Code, rejection.Operation, rejection.Channel, rejection.Field, transcript, rejection.Detail)
}

func sourceLink(channel CLIChannel, selector ...string) CLIProjectionSourceLink {
	return CLIProjectionSourceLink{channel: channel, selector: append([]string(nil), selector...)}
}

func cloneExactValue(value CLIExactValue) CLIExactValue {
	value.bytes = append([]byte(nil), value.bytes...)
	value.canonical = append([]byte(nil), value.canonical...)
	return value
}

func cloneProjectedFields(input []CLIProjectedField) []CLIProjectedField {
	result := make([]CLIProjectedField, len(input))
	for index, field := range input {
		result[index] = CLIProjectedField{id: field.id, value: cloneExactValue(field.value)}
	}
	return result
}

func cloneSourceLinks(input []CLIProjectionSourceLink) []CLIProjectionSourceLink {
	result := make([]CLIProjectionSourceLink, len(input))
	for index, link := range input {
		result[index] = CLIProjectionSourceLink{channel: link.channel, selector: append([]string(nil), link.selector...)}
	}
	return result
}

func cloneTranscript(input []CLIProjectionTraceEntry) []CLIProjectionTraceEntry {
	result := make([]CLIProjectionTraceEntry, len(input))
	for index, entry := range input {
		result[index] = CLIProjectionTraceEntry{operation: entry.operation, sourceLinks: cloneSourceLinks(entry.sourceLinks)}
	}
	return result
}

func cloneDerivation(input CLIProjectionDerivation) CLIProjectionDerivation {
	input.canonicalBytes = append([]byte(nil), input.canonicalBytes...)
	input.definitionFields = append([]CLIFieldID(nil), input.definitionFields...)
	input.transcript = cloneTranscript(input.transcript)
	return input
}
