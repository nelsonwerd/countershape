package model

import "github.com/nelsonwerd/countershape/internal/domain"

const httpProjectionVersionV1 = "http-projection/v1"

var closedHTTPFieldsV1 = []struct {
	id, channel, valueKind, missingPolicy string
	path                                  []string
}{
	{id: "http.status", channel: "http.status", path: []string{"status"}, valueKind: "SAFE_INTEGER", missingPolicy: "REJECT_CAPTURE"},
	{id: "http.header.content-type", channel: "http.headers", path: []string{"content-type"}, valueKind: "ORDERED_STRING_LIST", missingPolicy: "TAGGED_MISSING"},
	{id: "http.body.kind", channel: "http.body", path: []string{"strict_json", "kind"}, valueKind: "UTF8_STRING", missingPolicy: "REJECT_BODY"},
	{id: "http.body.metadata", channel: "http.body", path: []string{"strict_json", "metadata"}, valueKind: "CANONICAL_JSON_OBJECT", missingPolicy: "REJECT_BODY"},
}

type closedHTTPFieldIdentity struct {
	FieldID       string   `json:"field_id"`
	Channel       string   `json:"channel"`
	Path          []string `json:"path"`
	ValueKind     string   `json:"value_kind"`
	MissingPolicy string   `json:"missing_policy"`
}

type closedHTTPProjectionConfigIdentity struct {
	SchemaVersion string   `json:"schema_version"`
	Kind          string   `json:"kind"`
	Version       string   `json:"version"`
	Fields        []string `json:"fields"`
}

// NewHTTPProjectionAuthority derives the sole fixed v1 HTTP projection
// authority from the closed adapter schema.
func NewHTTPProjectionAuthority() (HTTPProjectionAuthority, error) {
	registry := make([]closedHTTPFieldIdentity, len(closedHTTPFieldsV1))
	fieldIDs := make([]string, len(closedHTTPFieldsV1))
	for index, field := range closedHTTPFieldsV1 {
		registry[index] = closedHTTPFieldIdentity{
			FieldID: field.id, Channel: field.channel, Path: append([]string(nil), field.path...),
			ValueKind: field.valueKind, MissingPolicy: field.missingPolicy,
		}
		fieldIDs[index] = field.id
	}
	registryDigest, _, err := digestTyped("HTTPFieldRegistry", struct {
		SchemaVersion string                    `json:"schema_version"`
		Kind          string                    `json:"kind"`
		Version       string                    `json:"version"`
		Fields        []closedHTTPFieldIdentity `json:"fields"`
	}{domain.SchemaVersion, "HTTPFieldRegistry", httpProjectionVersionV1, registry})
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	implementationDigest, err := digestExactBytes(
		"HTTPProjectionImplementation",
		[]byte("http-projection/v1\x00closed-four-field-registry\x00visible-validate-then-omit-operations\x00exact-operation-source-links\x00strict-json-object\x00exact-canonical"),
	)
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	configurationDigest, _, err := digestTyped("HTTPProjectionConfiguration", closedHTTPProjectionConfigIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPProjectionConfiguration",
		Version: httpProjectionVersionV1, Fields: fieldIDs,
	})
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	operations, err := closedHTTPOperationsV1()
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	bindingOperations := make([]domain.ProjectionOperationBinding, len(operations))
	for index, operation := range operations {
		rule, parseErr := domain.ParseDigest(operation.RuleDigest)
		if parseErr != nil {
			return HTTPProjectionAuthority{}, parseErr
		}
		bindingOperations[index] = domain.ProjectionOperationBinding{Name: operation.Name, RuleDigest: rule}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterHTTP, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest,
		AcceptedChannels:    []string{"http.body", "http.headers", "http.status"},
		Operations:          bindingOperations, Comparator: domain.ProjectionComparatorExact,
		FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	identity := resolvedProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPProjectionDefinition", Version: httpProjectionVersionV1,
		Fields: fieldIDs, Operations: operations, FieldRegistryDigest: registryDigest.String(),
		ImplementationDigest: implementationDigest.String(), ConfigurationDigest: configurationDigest.String(),
		ProjectionBindingDigest: binding.Digest().String(),
	}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionDefinition", identity)
	if err != nil {
		return HTTPProjectionAuthority{}, err
	}
	return HTTPProjectionAuthority{
		digest: digest, canonicalBytes: canonicalBytes, binding: binding, fieldIDs: append([]string(nil), fieldIDs...),
	}, nil
}

func closedHTTPOperationsV1() ([]resolvedProjectionOperation, error) {
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
	result := make([]resolvedProjectionOperation, len(definitions))
	for index, definition := range definitions {
		digest, err := digestExactBytes("HTTPProjectionOperationRule", []byte(definition[0]+"\x00"+definition[1]))
		if err != nil {
			return nil, err
		}
		result[index] = resolvedProjectionOperation{
			Name: definition[0], Semantics: definition[1], RuleDigest: digest.String(),
		}
	}
	return result, nil
}
