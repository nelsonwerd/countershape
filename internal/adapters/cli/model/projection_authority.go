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
	rehashed, err := canon.DigestBytes("CLIProjectionDefinition", canonicalBytes)
	if err != nil || rehashed.String() != digest.String() {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "resolved CLI projection definition digest differs from its bytes")
	}
	var identity resolvedProjectionIdentity
	if err := json.Unmarshal(canonicalBytes, &identity); err != nil ||
		identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "CLIProjectionDefinition" ||
		identity.Version != "cli-projection/v1" || identity.ProjectionBindingDigest != binding.Digest().String() ||
		identity.FieldRegistryDigest != binding.FieldRegistryDigest().String() ||
		identity.ImplementationDigest != binding.ImplementationDigest().String() ||
		identity.ConfigurationDigest != binding.ConfigurationDigest().String() ||
		binding.Comparator() != domain.ProjectionComparatorExact ||
		!equalStrings(binding.AcceptedChannels(), []string{"exit", "stderr", "stdout"}) {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter definition and generic projection binding are cross-paired")
	}
	configurationDigest, _, err := digestTyped("CLIProjectionConfiguration", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Fields        []string `json:"fields"`
	}{domain.SchemaVersion, "CLIProjectionConfiguration", "cli-projection/v1", identity.Fields})
	if err != nil || configurationDigest != binding.ConfigurationDigest() {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter fields do not resolve to the generic configuration digest")
	}
	implementationDigest, err := canon.DigestBytes(
		"CLIProjectionImplementation",
		[]byte("cli-projection/v1\x00closed-field-registry\x00visible-pure-operations\x00exact-canonical"),
	)
	if err != nil || implementationDigest.String() != binding.ImplementationDigest().String() {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter implementation does not resolve to the generic binding")
	}
	bindingOperations := binding.Operations()
	if len(identity.Operations) != len(bindingOperations) {
		return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter operation set differs from the generic binding")
	}
	for index, operation := range identity.Operations {
		rule, ruleErr := canon.DigestBytes(
			"CLIProjectionOperationRule", []byte(operation.Name+"\x00"+operation.Semantics),
		)
		if ruleErr != nil || rule.String() != operation.RuleDigest ||
			operation.Name != bindingOperations[index].Name ||
			operation.RuleDigest != bindingOperations[index].RuleDigest.String() {
			return CLIProjectionAuthority{}, refuse(CodeInvalidBinding, "adapter operation transcript differs from the generic binding")
		}
	}
	return CLIProjectionAuthority{
		digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...), binding: binding,
	}, nil
}

func (a CLIProjectionAuthority) Valid() bool {
	rebuilt, err := ResolveCLIProjectionAuthority(a.digest, a.canonicalBytes, a.binding)
	return err == nil && rebuilt.digest == a.digest &&
		bytes.Equal(rebuilt.canonicalBytes, a.canonicalBytes) && rebuilt.binding.Digest() == a.binding.Digest()
}

func (a CLIProjectionAuthority) Digest() domain.Digest { return a.digest }
func (a CLIProjectionAuthority) CanonicalBytes() []byte {
	return append([]byte(nil), a.canonicalBytes...)
}
func (a CLIProjectionAuthority) Binding() domain.ProjectionDefinitionBinding { return a.binding }
