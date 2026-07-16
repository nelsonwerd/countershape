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
	fieldIDs       []string
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
	var identity resolvedProjectionIdentity
	if err := json.Unmarshal(canonicalBytes, &identity); err != nil {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved HTTP projection definition typed decode failed")
	}
	expected, err := NewHTTPProjectionAuthority()
	if err != nil || expected.digest != digest || !bytes.Equal(expected.canonicalBytes, canonicalBytes) ||
		expected.binding.Digest() != binding.Digest() ||
		!bytes.Equal(expected.binding.CanonicalBytes(), binding.CanonicalBytes()) {
		return HTTPProjectionAuthority{}, refuse(CodeInvalidBinding, "HTTP projection differs from the closed v1 adapter schema")
	}
	return expected, nil
}

func (a HTTPProjectionAuthority) Valid() bool {
	rebuilt, err := ResolveHTTPProjectionAuthority(a.digest, a.canonicalBytes, a.binding)
	return err == nil && rebuilt.digest == a.digest && bytes.Equal(rebuilt.canonicalBytes, a.canonicalBytes) &&
		rebuilt.binding.Digest() == a.binding.Digest() && equalStrings(rebuilt.fieldIDs, a.fieldIDs)
}
func (a HTTPProjectionAuthority) Digest() domain.Digest { return a.digest }
func (a HTTPProjectionAuthority) CanonicalBytes() []byte {
	return append([]byte(nil), a.canonicalBytes...)
}
func (a HTTPProjectionAuthority) Binding() domain.ProjectionDefinitionBinding { return a.binding }
func (a HTTPProjectionAuthority) FieldIDs() []string                          { return append([]string(nil), a.fieldIDs...) }

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
