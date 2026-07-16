package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestCLIProjectionAuthorityRejectsSelfConsistentInventedSchemas(t *testing.T) {
	closed, err := NewCLIProjectionAuthority([]string{"cli.completion.kind", "cli.stdout.bytes"})
	if err != nil || !closed.Valid() {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		fields    []string
		registry  domain.Digest
		semantics bool
		unknown   bool
	}{
		{name: "arbitrary-registry", fields: closed.FieldIDs(), registry: testDigest("f")},
		{name: "arbitrary-operation-semantics", fields: closed.FieldIDs(), semantics: true},
		{name: "out-of-registry-order", fields: []string{"cli.stdout.bytes", "cli.completion.kind"}},
		{name: "unknown-definition-member", fields: closed.FieldIDs(), unknown: true},
	} {
		digest, exact, binding := forgedCLIProjection(t, test.fields, test.registry, test.semantics, test.unknown)
		if _, err := ResolveCLIProjectionAuthority(digest, exact, binding); err == nil {
			t.Fatalf("%s: self-consistent invented CLI projection minted closed authority", test.name)
		}
	}
}

func forgedCLIProjection(
	t *testing.T,
	fields []string,
	registryOverride domain.Digest,
	mutateSemantics, unknown bool,
) (domain.Digest, []byte, domain.ProjectionDefinitionBinding) {
	t.Helper()
	registrySource, err := NewCLIProjectionAuthority([]string{"cli.stdout.bytes"})
	if err != nil {
		t.Fatal(err)
	}
	registryDigest := registrySource.Binding().FieldRegistryDigest()
	if registryOverride.Valid() {
		registryDigest = registryOverride
	}
	implementationDigest, err := closedProjectionDigestBytes(
		"CLIProjectionImplementation",
		[]byte(cliProjectionVersionV1+"\x00closed-field-registry\x00visible-pure-operations\x00exact-canonical"),
	)
	if err != nil {
		t.Fatal(err)
	}
	configurationDigest, _, err := digestTyped("CLIProjectionConfiguration", closedCLIProjectionConfigIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionConfiguration",
		Version: cliProjectionVersionV1, Fields: fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	operations, err := closedCLIOperations(fields)
	if err != nil {
		t.Fatal(err)
	}
	if mutateSemantics {
		operations[0].Semantics = "invented-visible-semantics"
		rule, digestErr := closedProjectionDigestBytes(
			"CLIProjectionOperationRule", []byte(operations[0].Name+"\x00"+operations[0].Semantics),
		)
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		operations[0].RuleDigest = rule.String()
	}
	bindingOperations := make([]domain.ProjectionOperationBinding, len(operations))
	for index, operation := range operations {
		bindingOperations[index] = domain.ProjectionOperationBinding{
			Name: operation.Name, RuleDigest: domain.MustDigest(operation.RuleDigest),
		}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest, AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations: bindingOperations, Comparator: domain.ProjectionComparatorExact,
		FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := resolvedProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionDefinition", Version: cliProjectionVersionV1,
		Fields: fields, Operations: operations, FieldRegistryDigest: registryDigest.String(),
		ImplementationDigest: implementationDigest.String(), ConfigurationDigest: configurationDigest.String(),
		ProjectionBindingDigest: binding.Digest().String(),
	}
	exact, err := canon.CanonicalizeTyped(identity)
	if err != nil {
		t.Fatal(err)
	}
	if unknown {
		var object map[string]any
		if err := json.Unmarshal(exact, &object); err != nil {
			t.Fatal(err)
		}
		object["unknown_closed_claim"] = strings.Repeat("x", 1)
		exact, err = canon.CanonicalizeTyped(object)
		if err != nil {
			t.Fatal(err)
		}
	}
	digestRaw, err := canon.DigestBytes("CLIProjectionDefinition", exact)
	if err != nil {
		t.Fatal(err)
	}
	return domain.MustDigest(digestRaw.String()), exact, binding
}
