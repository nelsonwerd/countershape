package domain

import (
	"bytes"
	"sort"
	"unicode/utf8"
)

const (
	ProjectionComparatorExact = "EXACT_CANONICAL_V1"
	maxProjectionOperations   = 64
	maxProjectionNameBytes    = 128
)

// ProjectionOperationBinding is one ordered step in a projection pipeline.
// RuleDigest identifies the exact implementation/configuration of that step.
type ProjectionOperationBinding struct {
	Name       string
	RuleDigest Digest
}

// ProjectionDefinitionBindingConfig is the full semantic body from which an
// opaque projection-definition capability is derived. In particular, callers
// cannot turn a bare digest into a binding or relabel a CLI definition as HTTP.
type ProjectionDefinitionBindingConfig struct {
	AdapterDomain        AdapterDomain
	ImplementationDigest Digest
	ConfigurationDigest  Digest
	AcceptedChannels     []string
	Operations           []ProjectionOperationBinding
	Comparator           string
	FieldRegistryDigest  Digest
}

// ProjectionDefinitionBinding retains the complete declaration that produced
// its digest. WorldPlan consumes this capability, not a caller-paired digest
// and adapter label. Its zero value and parsed digest references are unusable.
type ProjectionDefinitionBinding struct {
	digest               Digest
	adapterDomain        AdapterDomain
	implementationDigest Digest
	configurationDigest  Digest
	acceptedChannels     []string
	operations           []ProjectionOperationBinding
	comparator           string
	fieldRegistryDigest  Digest
	canonicalBytes       []byte
}

type projectionOperationIdentity struct {
	Name       string `json:"name"`
	RuleDigest string `json:"rule_digest"`
}

type projectionDefinitionIdentity struct {
	SchemaVersion        string                        `json:"schema_version"`
	Kind                 string                        `json:"kind"`
	AdapterDomain        string                        `json:"adapter_domain"`
	ImplementationDigest string                        `json:"implementation_digest"`
	ConfigurationDigest  string                        `json:"configuration_digest"`
	AcceptedChannels     []string                      `json:"accepted_channels"`
	Operations           []projectionOperationIdentity `json:"operations"`
	Comparator           string                        `json:"comparator"`
	FieldRegistryDigest  string                        `json:"field_registry_digest"`
}

// NewProjectionDefinitionBinding validates the closed v1 declaration and
// derives both identity and adapter-domain authority from it. Operation order
// is preserved because these operations form a semantic pipeline. Channel
// order is normalized because accepted channels form a set.
func NewProjectionDefinitionBinding(config ProjectionDefinitionBindingConfig) (ProjectionDefinitionBinding, error) {
	if config.AdapterDomain != AdapterCLI && config.AdapterDomain != AdapterHTTP {
		return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "unsupported projection adapter domain")
	}
	if !config.ImplementationDigest.Valid() || !config.ConfigurationDigest.Valid() || !config.FieldRegistryDigest.Valid() {
		return ProjectionDefinitionBinding{}, refuse(ErrInvalidDigest, "projection definition")
	}
	if config.Comparator != ProjectionComparatorExact {
		return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "unsupported projection comparator")
	}
	maxChannels := 3
	if len(config.AcceptedChannels) < 1 || len(config.AcceptedChannels) > maxChannels {
		return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "projection channel set outside v1 bound")
	}
	channels := append([]string(nil), config.AcceptedChannels...)
	sort.Strings(channels)
	allowed := map[AdapterDomain]map[string]bool{
		AdapterCLI:  {"exit": true, "stderr": true, "stdout": true},
		AdapterHTTP: {"http.body": true, "http.headers": true, "http.status": true},
	}
	for index, channel := range channels {
		if channel == "" || len(channel) > maxProjectionNameBytes || !utf8.ValidString(channel) ||
			!allowed[config.AdapterDomain][channel] || (index > 0 && channel == channels[index-1]) {
			return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "invalid projection channel set")
		}
	}
	if len(config.Operations) < 1 || len(config.Operations) > maxProjectionOperations {
		return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "projection operation pipeline outside v1 bound")
	}
	operations := append([]ProjectionOperationBinding(nil), config.Operations...)
	seenOperations := make(map[string]struct{}, len(operations))
	operationIdentities := make([]projectionOperationIdentity, len(operations))
	for index, operation := range operations {
		if operation.Name == "" || len(operation.Name) > maxProjectionNameBytes || !utf8.ValidString(operation.Name) || !operation.RuleDigest.Valid() {
			return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "invalid projection operation")
		}
		if _, duplicate := seenOperations[operation.Name]; duplicate {
			return ProjectionDefinitionBinding{}, refuse(ErrInvalidWorldPlan, "duplicate projection operation name")
		}
		seenOperations[operation.Name] = struct{}{}
		operationIdentities[index] = projectionOperationIdentity{Name: operation.Name, RuleDigest: operation.RuleDigest.String()}
	}
	identity := projectionDefinitionIdentity{
		SchemaVersion: SchemaVersion, Kind: "ProjectionDefinition", AdapterDomain: string(config.AdapterDomain),
		ImplementationDigest: config.ImplementationDigest.String(), ConfigurationDigest: config.ConfigurationDigest.String(),
		AcceptedChannels: channels, Operations: operationIdentities, Comparator: config.Comparator,
		FieldRegistryDigest: config.FieldRegistryDigest.String(),
	}
	digest, canonicalBytes, err := digestTyped("ProjectionDefinition", identity)
	if err != nil {
		return ProjectionDefinitionBinding{}, err
	}
	return ProjectionDefinitionBinding{
		digest: digest, adapterDomain: config.AdapterDomain,
		implementationDigest: config.ImplementationDigest, configurationDigest: config.ConfigurationDigest,
		acceptedChannels: channels, operations: operations, comparator: config.Comparator,
		fieldRegistryDigest: config.FieldRegistryDigest, canonicalBytes: append([]byte(nil), canonicalBytes...),
	}, nil
}

func (b ProjectionDefinitionBinding) Valid() bool {
	if !b.digest.Valid() || len(b.canonicalBytes) == 0 {
		return false
	}
	rebuilt, err := NewProjectionDefinitionBinding(ProjectionDefinitionBindingConfig{
		AdapterDomain: b.adapterDomain, ImplementationDigest: b.implementationDigest,
		ConfigurationDigest: b.configurationDigest, AcceptedChannels: b.acceptedChannels,
		Operations: b.operations, Comparator: b.comparator, FieldRegistryDigest: b.fieldRegistryDigest,
	})
	return err == nil && rebuilt.digest == b.digest && bytes.Equal(rebuilt.canonicalBytes, b.canonicalBytes)
}

func (b ProjectionDefinitionBinding) Digest() Digest { return b.digest }

func (b ProjectionDefinitionBinding) AdapterDomain() AdapterDomain { return b.adapterDomain }

func (b ProjectionDefinitionBinding) FieldRegistryDigest() Digest { return b.fieldRegistryDigest }

func (b ProjectionDefinitionBinding) CanonicalBytes() []byte {
	return append([]byte(nil), b.canonicalBytes...)
}
