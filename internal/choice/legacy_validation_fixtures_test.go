package choice

import (
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxProjectionChannels     = 3
	maxProjectionOperations   = 64
	ProjectionComparatorExact = domain.ProjectionComparatorExact
)

// The generic projection builder is deliberately test-only. Production Choice
// derives fresh interpretation from projectiontranslate and retains only a
// private whole-projection path for exact legacy parsing.
type ProjectionOperation struct {
	Name       string
	RuleDigest domain.Digest
}

type ProjectionDefinitionConfig struct {
	AdapterDomain        domain.AdapterDomain
	ImplementationDigest domain.Digest
	ConfigurationDigest  domain.Digest
	AcceptedChannels     []string
	Operations           []ProjectionOperation
	Comparator           string
	Fields               []FieldDefinition
}

type ProjectionDefinition struct {
	digest   domain.Digest
	binding  domain.ProjectionDefinitionBinding
	registry FieldRegistry
}

func NewFieldRegistry(definitions []FieldDefinition) (FieldRegistry, error) {
	return newFieldRegistry(definitions, false)
}

func NewProjectionDefinition(config ProjectionDefinitionConfig) (ProjectionDefinition, error) {
	if (config.AdapterDomain != domain.AdapterCLI && config.AdapterDomain != domain.AdapterHTTP) ||
		!config.ImplementationDigest.Valid() || !config.ConfigurationDigest.Valid() ||
		config.Comparator != ProjectionComparatorExact || len(config.AcceptedChannels) == 0 || len(config.AcceptedChannels) > maxProjectionChannels ||
		len(config.Operations) == 0 || len(config.Operations) > maxProjectionOperations {
		return ProjectionDefinition{}, refusal(CodeInvalidFieldRegistry, "projection definition is incomplete")
	}
	registry, err := NewFieldRegistry(config.Fields)
	if err != nil {
		return ProjectionDefinition{}, err
	}
	operations := make([]domain.ProjectionOperationBinding, len(config.Operations))
	for index, operation := range config.Operations {
		operations[index] = domain.ProjectionOperationBinding{Name: operation.Name, RuleDigest: operation.RuleDigest}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: config.AdapterDomain, ImplementationDigest: config.ImplementationDigest,
		ConfigurationDigest: config.ConfigurationDigest, AcceptedChannels: config.AcceptedChannels,
		Operations: operations, Comparator: config.Comparator, FieldRegistryDigest: registry.fieldRegistryDigest,
	})
	if err != nil {
		return ProjectionDefinition{}, refusal(CodeInvalidFieldRegistry, "projection definition body is outside the closed v1 profile")
	}
	registry.projectionDefinitionDigest = binding.Digest()
	return ProjectionDefinition{digest: binding.Digest(), binding: binding, registry: registry}, nil
}

func (d ProjectionDefinition) Digest() domain.Digest { return d.digest }

func (d ProjectionDefinition) Binding() domain.ProjectionDefinitionBinding { return d.binding }

func (d ProjectionDefinition) Registry() FieldRegistry { return d.registry.clone() }

func NewConfirmedOutcomeSet(
	registry FieldRegistry,
	roster compare.ConfirmedProjectionRoster,
	inputs []ProjectionProofInput,
) (ConfirmedOutcomeSet, error) {
	return newLegacyConfirmedOutcomeSet(registry, roster, inputs)
}
