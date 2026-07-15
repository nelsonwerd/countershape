package http

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const httpProjectionVersion = "http-projection/v1"

type HTTPChannel string

const (
	HTTPChannelStatus  HTTPChannel = "http.status"
	HTTPChannelHeaders HTTPChannel = "http.headers"
	HTTPChannelBody    HTTPChannel = "http.body"
	// HTTPChannelRuntime is derivation-trace provenance, not a selectable
	// projection-output channel. It makes observation-owned facts visible
	// without pretending that they came from the response body.
	HTTPChannelRuntime HTTPChannel = "http.runtime"
)

type HTTPFieldID string

const (
	HTTPFieldStatus       HTTPFieldID = "http.status"
	HTTPFieldContentType  HTTPFieldID = "http.header.content-type"
	HTTPFieldBodyKind     HTTPFieldID = "http.body.kind"
	HTTPFieldBodyMetadata HTTPFieldID = "http.body.metadata"
)

var fixedHTTPFields = []HTTPFieldID{HTTPFieldStatus, HTTPFieldContentType, HTTPFieldBodyKind, HTTPFieldBodyMetadata}

type HTTPFieldDescriptor struct {
	id                       HTTPFieldID
	channel                  HTTPChannel
	path                     []string
	valueKind, missingPolicy string
}

func (d HTTPFieldDescriptor) ID() HTTPFieldID       { return d.id }
func (d HTTPFieldDescriptor) Channel() HTTPChannel  { return d.channel }
func (d HTTPFieldDescriptor) Path() []string        { return append([]string(nil), d.path...) }
func (d HTTPFieldDescriptor) ValueKind() string     { return d.valueKind }
func (d HTTPFieldDescriptor) MissingPolicy() string { return d.missingPolicy }

var httpFieldRegistry = []HTTPFieldDescriptor{
	{id: HTTPFieldStatus, channel: HTTPChannelStatus, path: []string{"status"}, valueKind: "SAFE_INTEGER", missingPolicy: "REJECT_CAPTURE"},
	{id: HTTPFieldContentType, channel: HTTPChannelHeaders, path: []string{"content-type"}, valueKind: "ORDERED_STRING_LIST", missingPolicy: "TAGGED_MISSING"},
	{id: HTTPFieldBodyKind, channel: HTTPChannelBody, path: []string{"strict_json", "kind"}, valueKind: "UTF8_STRING", missingPolicy: "REJECT_BODY"},
	{id: HTTPFieldBodyMetadata, channel: HTTPChannelBody, path: []string{"strict_json", "metadata"}, valueKind: "CANONICAL_JSON_OBJECT", missingPolicy: "REJECT_BODY"},
}

func HTTPFieldRegistry() []HTTPFieldDescriptor {
	result := append([]HTTPFieldDescriptor(nil), httpFieldRegistry...)
	for index := range result {
		result[index].path = append([]string(nil), result[index].path...)
	}
	return result
}

type HTTPProjectionOperation struct {
	name, semantics string
	ruleDigest      domain.Digest
}

func (o HTTPProjectionOperation) Name() string              { return o.name }
func (o HTTPProjectionOperation) Semantics() string         { return o.semantics }
func (o HTTPProjectionOperation) RuleDigest() domain.Digest { return o.ruleDigest }

type HTTPProjectionDefinition struct {
	digest, registryDigest, implementationDigest, configurationDigest domain.Digest
	canonicalBytes                                                    []byte
	operations                                                        []HTTPProjectionOperation
	binding                                                           domain.ProjectionDefinitionBinding
}

type projectionOperationIdentity struct {
	Name       string `json:"name"`
	Semantics  string `json:"semantics"`
	RuleDigest string `json:"rule_digest"`
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
type httpProjectionDefinitionIdentity struct {
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

func NewHTTPProjectionDefinition() (HTTPProjectionDefinition, error) {
	registryIDs := make([]fieldRegistryIdentity, len(httpFieldRegistry))
	for index, field := range httpFieldRegistry {
		registryIDs[index] = fieldRegistryIdentity{string(field.id), string(field.channel), field.path, field.valueKind, field.missingPolicy}
	}
	registryDigest, _, err := digestTyped("HTTPFieldRegistry", struct {
		SchemaVersion string                  `json:"schema_version"`
		Kind          string                  `json:"kind"`
		Version       string                  `json:"version"`
		Fields        []fieldRegistryIdentity `json:"fields"`
	}{domain.SchemaVersion, "HTTPFieldRegistry", httpProjectionVersion, registryIDs})
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	implementationDigest, err := digestBytes("HTTPProjectionImplementation", []byte("http-projection/v1\x00closed-four-field-registry\x00visible-validate-then-omit-operations\x00exact-operation-source-links\x00strict-json-object\x00exact-canonical"))
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	fieldNames := []string{string(HTTPFieldStatus), string(HTTPFieldContentType), string(HTTPFieldBodyKind), string(HTTPFieldBodyMetadata)}
	// MUTATION_ANCHOR: http-projection-mandates-status-field
	if fieldNames[0] != string(HTTPFieldStatus) {
		return HTTPProjectionDefinition{}, refuse(CodeInvalidProjection, "status field is mandatory")
	}
	// MUTATION_ANCHOR: http-projection-mandates-disclosure-metadata-field
	if fieldNames[3] != string(HTTPFieldBodyMetadata) {
		return HTTPProjectionDefinition{}, refuse(CodeInvalidProjection, "disclosure metadata field is mandatory")
	}
	configurationDigest, _, err := digestTyped("HTTPProjectionConfiguration", projectionConfigIdentity{domain.SchemaVersion, "HTTPProjectionConfiguration", httpProjectionVersion, fieldNames})
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	operations, err := fixedProjectionOperations()
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	domainOperations := make([]domain.ProjectionOperationBinding, len(operations))
	operationIDs := make([]projectionOperationIdentity, len(operations))
	for index, operation := range operations {
		domainOperations[index] = domain.ProjectionOperationBinding{Name: operation.name, RuleDigest: operation.ruleDigest}
		operationIDs[index] = projectionOperationIdentity{operation.name, operation.semantics, operation.ruleDigest.String()}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterHTTP, ImplementationDigest: implementationDigest, ConfigurationDigest: configurationDigest,
		AcceptedChannels: []string{string(HTTPChannelBody), string(HTTPChannelHeaders), string(HTTPChannelStatus)},
		Operations:       domainOperations, Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	identity := httpProjectionDefinitionIdentity{
		domain.SchemaVersion, "HTTPProjectionDefinition", httpProjectionVersion, fieldNames, operationIDs, registryDigest.String(), implementationDigest.String(), configurationDigest.String(), binding.Digest().String(),
	}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionDefinition", identity)
	if err != nil {
		return HTTPProjectionDefinition{}, err
	}
	return HTTPProjectionDefinition{digest: digest, canonicalBytes: canonicalBytes, operations: operations, registryDigest: registryDigest, implementationDigest: implementationDigest, configurationDigest: configurationDigest, binding: binding}, nil
}

func fixedProjectionOperations() ([]HTTPProjectionOperation, error) {
	definitions := [][2]string{
		{"http.require-complete-response/v1", "require no controls, accepted fixture readiness, one complete request, one complete parsed response, and clean teardown"},
		{"http.select-status/v1", "select the complete application status including 500"},
		{"http.select-content-type-multimap/v1", "preserve ordered duplicate content-type values and tagged missing versus present-empty"},
		{"http.parse-body-strict-json-object/v1", "parse one strict JSON object with duplicate rejection and no trailing data"},
		{"http.validate-then-omit-request-id/v1", "validate the top-level request_id value as a string, then omit that member from projection output"},
		{"http.validate-then-omit-scratch-root/v1", "validate the top-level scratch_root value as a string exactly equal to the captured runtime scratch-root authority, then omit that member from projection output"},
		{"http.select-body-kind/v1", "select the required top-level kind string"},
		{"http.select-body-metadata/v1", "select the required disclosure-bearing metadata object without deleting its members"},
		{"http.encode-four-fields-canonical/v1", "encode the fixed registry-ordered tagged field tuple exactly"},
	}
	result := make([]HTTPProjectionOperation, len(definitions))
	for index, definition := range definitions {
		digest, err := digestBytes("HTTPProjectionOperationRule", []byte(definition[0]+"\x00"+definition[1]))
		if err != nil {
			return nil, err
		}
		result[index] = HTTPProjectionOperation{name: definition[0], semantics: definition[1], ruleDigest: digest}
	}
	return result, nil
}

func (d HTTPProjectionDefinition) Valid() bool {
	rebuilt, err := NewHTTPProjectionDefinition()
	return err == nil && rebuilt.digest == d.digest && bytes.Equal(rebuilt.canonicalBytes, d.canonicalBytes) && rebuilt.binding.Digest() == d.binding.Digest()
}
func (d HTTPProjectionDefinition) Digest() domain.Digest { return d.digest }
func (d HTTPProjectionDefinition) CanonicalBytes() []byte {
	return append([]byte(nil), d.canonicalBytes...)
}
func (d HTTPProjectionDefinition) Fields() []HTTPFieldID {
	return append([]HTTPFieldID(nil), fixedHTTPFields...)
}
func (d HTTPProjectionDefinition) Operations() []HTTPProjectionOperation {
	return append([]HTTPProjectionOperation(nil), d.operations...)
}
func (d HTTPProjectionDefinition) FieldRegistryDigest() domain.Digest          { return d.registryDigest }
func (d HTTPProjectionDefinition) Binding() domain.ProjectionDefinitionBinding { return d.binding }

type HTTPProjectionSourceLink struct {
	channel  HTTPChannel
	selector []string
}

func (l HTTPProjectionSourceLink) Channel() HTTPChannel { return l.channel }
func (l HTTPProjectionSourceLink) Selector() []string   { return append([]string(nil), l.selector...) }

func httpSourceLink(channel HTTPChannel, selector ...string) HTTPProjectionSourceLink {
	return HTTPProjectionSourceLink{channel: channel, selector: append([]string(nil), selector...)}
}

func cloneHTTPSourceLinks(input []HTTPProjectionSourceLink) []HTTPProjectionSourceLink {
	result := make([]HTTPProjectionSourceLink, len(input))
	for index, link := range input {
		result[index] = HTTPProjectionSourceLink{
			channel:  link.channel,
			selector: append([]string(nil), link.selector...),
		}
	}
	return result
}

type HTTPProjectionTraceEntry struct {
	operation   HTTPProjectionOperation
	sourceLinks []HTTPProjectionSourceLink
}

func (e HTTPProjectionTraceEntry) Operation() HTTPProjectionOperation { return e.operation }
func (e HTTPProjectionTraceEntry) SourceLinks() []HTTPProjectionSourceLink {
	return cloneHTTPSourceLinks(e.sourceLinks)
}
func cloneTrace(input []HTTPProjectionTraceEntry) []HTTPProjectionTraceEntry {
	result := make([]HTTPProjectionTraceEntry, len(input))
	for index, entry := range input {
		result[index] = HTTPProjectionTraceEntry{
			operation:   entry.operation,
			sourceLinks: cloneHTTPSourceLinks(entry.sourceLinks),
		}
	}
	return result
}

type HTTPProjectedField struct {
	id                  HTTPFieldID
	tag                 string
	integer             int64
	strings             []string
	text, canonicalJSON string
}

func (f HTTPProjectedField) ID() HTTPFieldID        { return f.id }
func (f HTTPProjectedField) Tag() string            { return f.tag }
func (f HTTPProjectedField) Integer() (int64, bool) { return f.integer, f.tag == "INTEGER" }
func (f HTTPProjectedField) Strings() ([]string, bool) {
	return append([]string(nil), f.strings...), f.tag == "STRING_LIST"
}
func (f HTTPProjectedField) String() (string, bool) { return f.text, f.tag == "STRING" }
func (f HTTPProjectedField) CanonicalJSON() (string, bool) {
	return f.canonicalJSON, f.tag == "CANONICAL_JSON"
}

type HTTPProjectionResult struct {
	digest                              domain.Digest
	canonicalBytes, projectionBytes     []byte
	fields                              []HTTPProjectedField
	transcript                          []HTTPProjectionTraceEntry
	observationDigest, definitionDigest domain.Digest
}

func (r HTTPProjectionResult) Digest() domain.Digest { return r.digest }
func (r HTTPProjectionResult) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPProjectionResult) ProjectionBytes() []byte {
	return append([]byte(nil), r.projectionBytes...)
}
func (r HTTPProjectionResult) Fields() []HTTPProjectedField {
	return append([]HTTPProjectedField(nil), r.fields...)
}
func (r HTTPProjectionResult) Transcript() []HTTPProjectionTraceEntry {
	return cloneTrace(r.transcript)
}

func (r HTTPProjectionResult) Valid() bool {
	operations, err := fixedProjectionOperations()
	if err != nil || !r.digest.Valid() || !r.observationDigest.Valid() || !r.definitionDigest.Valid() || len(r.canonicalBytes) == 0 || len(r.projectionBytes) == 0 || len(r.fields) != 4 || len(r.transcript) != len(operations) {
		return false
	}
	for index := range operations {
		if r.transcript[index].operation.name != operations[index].name || r.transcript[index].operation.semantics != operations[index].semantics || r.transcript[index].operation.ruleDigest != operations[index].ruleDigest {
			return false
		}
	}
	expectedTraceBytes, err := canon.CanonicalizeTyped(traceIdentities(projectionTraceForOperations(operations)))
	if err != nil {
		return false
	}
	actualTraceBytes, err := canon.CanonicalizeTyped(traceIdentities(r.transcript))
	if err != nil || !bytes.Equal(expectedTraceBytes, actualTraceBytes) {
		return false
	}
	projectionByteDigest, err := digestBytes("HTTPProjectionCanonicalBytes", r.projectionBytes)
	if err != nil {
		return false
	}
	identity := struct {
		SchemaVersion, Kind, ObservationDigest, DefinitionDigest, ProjectionByteDigest string
		Transcript                                                                     any
	}{domain.SchemaVersion, "HTTPProjectionDerivation", r.observationDigest.String(), r.definitionDigest.String(), projectionByteDigest.String(), traceIdentities(r.transcript)}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionDerivation", identity)
	return err == nil && digest == r.digest && bytes.Equal(canonicalBytes, r.canonicalBytes)
}

func (d HTTPProjectionDefinition) Project(observation HTTPCapturedObservation) (HTTPProjectionResult, error) {
	if !d.Valid() || !observation.Valid() || observation.projectionDefinitionDigest != d.binding.Digest() || observation.adapterProjectionDefinitionDigest != d.digest {
		return HTTPProjectionResult{}, refuse(CodeCaptureLineageMismatch, "observation and projection definition differ")
	}
	response, present := observation.Response()
	if !present || len(observation.controls) != 0 {
		return HTTPProjectionResult{}, d.reject(observation, 0, CodeProjectionControl, HTTPChannelStatus, HTTPFieldStatus, "capture is controlled or has no complete response", nil)
	}
	projectionBytes, fields, projectionErr := projectParsedResponseCore(response, observation.scratchRoot)
	if projectionErr != nil {
		return HTTPProjectionResult{}, d.reject(observation, projectionErr.step, projectionErr.code, projectionErr.channel, projectionErr.field, projectionErr.message, nil)
	}
	transcript := d.trace()
	// MUTATION_ANCHOR: http-projection-visible-validate-then-omit-transcript
	if transcript[4].operation.name != "http.validate-then-omit-request-id/v1" || transcript[5].operation.name != "http.validate-then-omit-scratch-root/v1" {
		return HTTPProjectionResult{}, refuse(CodeProjectionInternal, "validate-then-omit operations disappeared from transcript")
	}
	projectionByteDigest, err := digestBytes("HTTPProjectionCanonicalBytes", projectionBytes)
	if err != nil {
		return HTTPProjectionResult{}, err
	}
	traceIDs := traceIdentities(transcript)
	identity := struct {
		SchemaVersion, Kind, ObservationDigest, DefinitionDigest, ProjectionByteDigest string
		Transcript                                                                     any
	}{domain.SchemaVersion, "HTTPProjectionDerivation", observation.digest.String(), d.binding.Digest().String(), projectionByteDigest.String(), traceIDs}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionDerivation", identity)
	if err != nil {
		return HTTPProjectionResult{}, err
	}
	return HTTPProjectionResult{digest: digest, canonicalBytes: canonicalBytes, projectionBytes: projectionBytes, fields: fields, transcript: transcript, observationDigest: observation.digest, definitionDigest: d.binding.Digest()}, nil
}

// ProjectParsedResponse applies the production HTTP projection policy to an
// already parsed response without admitting it into comparison. It exists for
// controlled negative fixtures that must derive their comparison material
// through the same fixed field authority as Project. Project remains the only
// lineage- and control-aware admission boundary.
func ProjectParsedResponse(response HTTPResponse, expectedScratchRoot string) ([]byte, error) {
	if !response.Valid() || !validExpectedScratchRoot(expectedScratchRoot) {
		return nil, refuse(CodeCaptureEvidenceInvalid, "parsed response or expected scratch root is invalid")
	}
	projectionBytes, _, projectionErr := projectParsedResponseCore(response, expectedScratchRoot)
	if projectionErr != nil {
		return nil, refuse(projectionErr.code, projectionErr.message)
	}
	return append([]byte(nil), projectionBytes...), nil
}

type parsedResponseProjectionError struct {
	step    int
	code    string
	channel HTTPChannel
	field   HTTPFieldID
	message string
}

func projectParsedResponseCore(response HTTPResponse, expectedScratchRoot string) ([]byte, []HTTPProjectedField, *parsedResponseProjectionError) {
	body, err := canon.Parse(response.Body())
	if err != nil {
		return nil, nil, &parsedResponseProjectionError{3, CodeProjectionJSON, HTTPChannelBody, HTTPFieldBodyKind, err.Error()}
	}
	if body.Kind() != canon.KindObject {
		return nil, nil, &parsedResponseProjectionError{3, CodeProjectionJSONRoot, HTTPChannelBody, HTTPFieldBodyKind, "response body must be one strict JSON object"}
	}
	requestID, ok := body.LookupMember("request_id")
	if !ok {
		return nil, nil, &parsedResponseProjectionError{4, CodeProjectionMetadata, HTTPChannelBody, HTTPFieldBodyMetadata, "request_id is required captured volatility"}
	}
	if _, ok := requestID.Text(); !ok {
		return nil, nil, &parsedResponseProjectionError{4, CodeProjectionMetadata, HTTPChannelBody, HTTPFieldBodyMetadata, "request_id must be a string"}
	}
	scratch, ok := body.LookupMember("scratch_root")
	scratchText, scratchString := scratch.Text()
	if !ok || !scratchString || scratchText != expectedScratchRoot {
		return nil, nil, &parsedResponseProjectionError{5, CodeProjectionMetadata, HTTPChannelBody, HTTPFieldBodyMetadata, "scratch_root must exactly match captured runtime root"}
	}
	kindValue, ok := body.LookupMember("kind")
	kind, kindString := kindValue.Text()
	if !ok || !kindString {
		return nil, nil, &parsedResponseProjectionError{6, CodeProjectionMetadata, HTTPChannelBody, HTTPFieldBodyKind, "kind must be a string"}
	}
	metadata, ok := body.LookupMember("metadata")
	if !ok || metadata.Kind() != canon.KindObject {
		return nil, nil, &parsedResponseProjectionError{7, CodeProjectionMetadata, HTTPChannelBody, HTTPFieldBodyMetadata, "disclosure metadata must be an object"}
	}
	metadataBytes, err := metadata.CanonicalChecked()
	if err != nil {
		return nil, nil, &parsedResponseProjectionError{7, CodeProjectionJSON, HTTPChannelBody, HTTPFieldBodyMetadata, err.Error()}
	}
	contentTypes := response.HeaderValues("content-type")
	contentTag := "MISSING"
	if len(contentTypes) > 0 {
		contentTag = "STRING_LIST"
	}
	fields := []HTTPProjectedField{
		{id: HTTPFieldStatus, tag: "INTEGER", integer: int64(response.Status())},
		{id: HTTPFieldContentType, tag: contentTag, strings: contentTypes},
		{id: HTTPFieldBodyKind, tag: "STRING", text: kind},
		{id: HTTPFieldBodyMetadata, tag: "CANONICAL_JSON", canonicalJSON: string(metadataBytes)},
	}
	type fieldIdentity struct {
		FieldID, Tag          string
		Integer               int64
		Strings               []string
		String, CanonicalJSON string
	}
	identities := make([]fieldIdentity, len(fields))
	for i, field := range fields {
		values := append([]string(nil), field.strings...)
		if values == nil {
			values = []string{}
		}
		identities[i] = fieldIdentity{string(field.id), field.tag, field.integer, values, field.text, field.canonicalJSON}
	}
	projectionValue := struct {
		SchemaVersion, Kind string
		Fields              []fieldIdentity
	}{domain.SchemaVersion, "HTTPProjection", identities}
	projectionBytes, err := canon.CanonicalizeTyped(projectionValue)
	if err != nil {
		return nil, nil, &parsedResponseProjectionError{7, CodeProjectionInternal, HTTPChannelBody, HTTPFieldBodyMetadata, err.Error()}
	}
	return projectionBytes, fields, nil
}

func (d HTTPProjectionDefinition) trace() []HTTPProjectionTraceEntry {
	return projectionTraceForOperations(d.operations)
}

func projectionTraceForOperations(operations []HTTPProjectionOperation) []HTTPProjectionTraceEntry {
	sources := [][]HTTPProjectionSourceLink{
		{
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "controls"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "transport_kind"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "readiness_accepted"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "request_attempted"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "request_complete"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "response_parsed"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "seed_overlay_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "readiness_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "exchange_receipt_presence"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "invocation_receipt_presence"),
		},
		{httpSourceLink(HTTPChannelStatus, "status")},
		{httpSourceLink(HTTPChannelHeaders, "content-type")},
		{httpSourceLink(HTTPChannelBody, "captured_bytes")},
		{httpSourceLink(HTTPChannelBody, "strict_json", "request_id")},
		{
			httpSourceLink(HTTPChannelBody, "strict_json", "scratch_root"),
			httpSourceLink(HTTPChannelRuntime, "captured_observation", "scratch_root"),
		},
		{httpSourceLink(HTTPChannelBody, "strict_json", "kind")},
		{httpSourceLink(HTTPChannelBody, "strict_json", "metadata")},
		{
			httpSourceLink(HTTPChannelStatus, "status"),
			httpSourceLink(HTTPChannelHeaders, "content-type"),
			httpSourceLink(HTTPChannelBody, "strict_json", "kind"),
			httpSourceLink(HTTPChannelBody, "strict_json", "metadata"),
		},
	}
	result := make([]HTTPProjectionTraceEntry, len(operations))
	for i, operation := range operations {
		var links []HTTPProjectionSourceLink
		if i < len(sources) {
			links = sources[i]
		}
		result[i] = HTTPProjectionTraceEntry{operation: operation, sourceLinks: cloneHTTPSourceLinks(links)}
	}
	return result
}
func traceIdentities(trace []HTTPProjectionTraceEntry) any {
	type link struct {
		Channel  string
		Selector []string
	}
	type entry struct {
		OperationName, OperationSemantics, RuleDigest string
		SourceLinks                                   []link
	}
	result := make([]entry, len(trace))
	for i, item := range trace {
		links := make([]link, len(item.sourceLinks))
		for j, source := range item.sourceLinks {
			links[j] = link{string(source.channel), source.selector}
		}
		result[i] = entry{item.operation.name, item.operation.semantics, item.operation.ruleDigest.String(), links}
	}
	return result
}

func (d HTTPProjectionDefinition) reject(observation HTTPCapturedObservation, operation int, code string, channel HTTPChannel, field HTTPFieldID, detail string, transcript []HTTPProjectionTraceEntry) error {
	if transcript == nil {
		transcript = d.trace()[:operation+1]
	}
	type identity struct {
		SchemaVersion, Kind, ObservationDigest, AdapterDefinitionDigest, DefinitionDigest, Code, Operation, Channel, Field, Detail string
		Transcript                                                                                                                 any
	}
	value := identity{domain.SchemaVersion, "HTTPProjectionRejection", observation.digest.String(), d.digest.String(), d.binding.Digest().String(), code, d.operations[operation].name, string(channel), string(field), detail, traceIdentities(transcript)}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionRejection", value)
	if err != nil {
		return err
	}
	return &ProjectionRejection{Code: code, Operation: d.operations[operation].name, Channel: channel, Field: field, Detail: detail, digest: digest, canonicalBytes: canonicalBytes, observationDigest: observation.digest, adapterDefinitionDigest: d.digest, definitionDigest: d.binding.Digest(), transcript: transcript}
}
