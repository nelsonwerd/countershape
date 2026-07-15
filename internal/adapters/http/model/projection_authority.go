package model

import (
	"bytes"
	"encoding/json"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type resolvedProjectionOperation struct {
	Name       string `json:"name"`
	Semantics  string `json:"semantics"`
	RuleDigest string `json:"rule_digest"`
}

type resolvedProjectionIdentity struct {
	SchemaVersion           string                        `json:"schema_version"`
	Kind                    string                        `json:"kind"`
	Version                 string                        `json:"version"`
	Fields                  []string                      `json:"fields"`
	Operations              []resolvedProjectionOperation `json:"operations"`
	FieldRegistryDigest     string                        `json:"field_registry_digest"`
	ImplementationDigest    string                        `json:"implementation_digest"`
	ConfigurationDigest     string                        `json:"configuration_digest"`
	ProjectionBindingDigest string                        `json:"projection_definition_binding_digest"`
}

type HTTPProjectionAuthority struct {
	digest         domain.Digest
	canonicalBytes []byte
	binding        domain.ProjectionDefinitionBinding
}

func ResolveHTTPProjectionAuthority(
	digest domain.Digest,
	canonicalBytes []byte,
	binding domain.ProjectionDefinitionBinding,
) (HTTPProjectionAuthority, error) {
	if !digest.Valid() || len(canonicalBytes) == 0 || !binding.Valid() || binding.AdapterDomain() != domain.AdapterHTTP {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved HTTP projection authority is incomplete")
	}
	canonical, err := canon.Canonicalize(canonicalBytes)
	if err != nil || !bytes.Equal(canonical, canonicalBytes) {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved HTTP projection definition is not exact canonical bytes")
	}
	rehashed, err := canon.DigestBytes("HTTPProjectionDefinition", canonicalBytes)
	if err != nil || rehashed.String() != digest.String() {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved HTTP projection digest differs from its bytes")
	}
	var identity resolvedProjectionIdentity
	if err := json.Unmarshal(canonicalBytes, &identity); err != nil ||
		identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "HTTPProjectionDefinition" ||
		identity.Version != "http-projection/v1" || identity.ProjectionBindingDigest != binding.Digest().String() ||
		identity.FieldRegistryDigest != binding.FieldRegistryDigest().String() ||
		identity.ImplementationDigest != binding.ImplementationDigest().String() ||
		identity.ConfigurationDigest != binding.ConfigurationDigest().String() ||
		binding.Comparator() != domain.ProjectionComparatorExact ||
		!equalStrings(binding.AcceptedChannels(), []string{"http.body", "http.headers", "http.status"}) {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter definition and generic projection binding are cross-paired")
	}
	if !equalStrings(identity.Fields, []string{
		"http.status", "http.header.content-type", "http.body.kind", "http.body.metadata",
	}) {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP v1 requires the complete fixed four-field registry")
	}
	configurationDigest, _, err := digestTyped("HTTPProjectionConfiguration", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Fields        []string `json:"fields"`
	}{domain.SchemaVersion, "HTTPProjectionConfiguration", "http-projection/v1", identity.Fields})
	if err != nil || configurationDigest != binding.ConfigurationDigest() {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP projection configuration differs from the generic binding")
	}
	implementationDigest, err := canon.DigestBytes(
		"HTTPProjectionImplementation",
		[]byte("http-projection/v1\x00closed-four-field-registry\x00visible-validate-then-omit-operations\x00exact-operation-source-links\x00strict-json-object\x00exact-canonical"),
	)
	if err != nil || implementationDigest.String() != binding.ImplementationDigest().String() {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP projection implementation differs from the generic binding")
	}
	bindingOperations := binding.Operations()
	if len(identity.Operations) != len(bindingOperations) {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP operation set differs from the generic binding")
	}
	for index, operation := range identity.Operations {
		rule, ruleErr := canon.DigestBytes("HTTPProjectionOperationRule", []byte(operation.Name+"\x00"+operation.Semantics))
		if ruleErr != nil || rule.String() != operation.RuleDigest || operation.Name != bindingOperations[index].Name ||
			operation.RuleDigest != bindingOperations[index].RuleDigest.String() {
			return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP projection transcript differs from the generic binding")
		}
	}
	return HTTPProjectionAuthority{digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...), binding: binding}, nil
}

func (a HTTPProjectionAuthority) Valid() bool {
	rebuilt, err := ResolveHTTPProjectionAuthority(a.digest, a.canonicalBytes, a.binding)
	return err == nil && rebuilt.digest == a.digest && bytes.Equal(rebuilt.canonicalBytes, a.canonicalBytes) &&
		rebuilt.binding.Digest() == a.binding.Digest()
}
func (a HTTPProjectionAuthority) Digest() domain.Digest { return a.digest }
func (a HTTPProjectionAuthority) CanonicalBytes() []byte {
	return append([]byte(nil), a.canonicalBytes...)
}
func (a HTTPProjectionAuthority) Binding() domain.ProjectionDefinitionBinding { return a.binding }

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
