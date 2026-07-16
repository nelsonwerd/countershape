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

// CLIProjectionAuthority is the cycle-free proof that a generic projection
// binding resolved to one exact adapter-owned definition before execution.
// The top-level CLI package constructs it from its closed registry definition;
// world receives only this immutable capability through CLIExecutionBinding.
type CLIProjectionAuthority struct {
	digest         domain.Digest
	canonicalBytes []byte
	binding        domain.ProjectionDefinitionBinding
	fieldIDs       []string
}

func ResolveCLIProjectionAuthority(
	digest domain.Digest,
	canonicalBytes []byte,
	binding domain.ProjectionDefinitionBinding,
) (CLIProjectionAuthority, error) {
	if !digest.Valid() || len(canonicalBytes) == 0 || !binding.Valid() ||
		binding.AdapterDomain() != domain.AdapterCLI {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved CLI projection authority is incomplete")
	}
	canonical, err := canon.Canonicalize(canonicalBytes)
	if err != nil || !bytes.Equal(canonical, canonicalBytes) {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved CLI projection definition is not exact canonical bytes")
	}
	var identity resolvedProjectionIdentity
	if err := json.Unmarshal(canonicalBytes, &identity); err != nil {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved CLI projection definition typed decode failed")
	}
	expected, err := NewCLIProjectionAuthority(identity.Fields)
	if err != nil || expected.digest != digest || !bytes.Equal(expected.canonicalBytes, canonicalBytes) ||
		expected.binding.Digest() != binding.Digest() ||
		!bytes.Equal(expected.binding.CanonicalBytes(), binding.CanonicalBytes()) {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "CLI projection differs from the closed v1 adapter schema")
	}
	return expected, nil
}

func (a CLIProjectionAuthority) Valid() bool {
	rebuilt, err := ResolveCLIProjectionAuthority(a.digest, a.canonicalBytes, a.binding)
	return err == nil && rebuilt.digest == a.digest &&
		bytes.Equal(rebuilt.canonicalBytes, a.canonicalBytes) && rebuilt.binding.Digest() == a.binding.Digest() &&
		equalStrings(rebuilt.fieldIDs, a.fieldIDs)
}

func (a CLIProjectionAuthority) Digest() domain.Digest { return a.digest }
func (a CLIProjectionAuthority) CanonicalBytes() []byte {
	return append([]byte(nil), a.canonicalBytes...)
}
func (a CLIProjectionAuthority) Binding() domain.ProjectionDefinitionBinding { return a.binding }
func (a CLIProjectionAuthority) FieldIDs() []string                          { return append([]string(nil), a.fieldIDs...) }
