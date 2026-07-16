package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestHTTPProjectionAuthorityRejectsSelfConsistentInventedSchemas(t *testing.T) {
	closed, err := NewHTTPProjectionAuthority()
	if err != nil || !closed.Valid() {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		registry  domain.Digest
		semantics bool
		unknown   bool
	}{
		{name: "arbitrary-registry", registry: domain.MustDigest("sha256:" + strings.Repeat("f", 64))},
		{name: "arbitrary-operation-semantics", semantics: true},
		{name: "unknown-definition-member", unknown: true},
	} {
		digest, exact, binding := forgedHTTPProjection(t, test.registry, test.semantics, test.unknown)
		if _, err := ResolveHTTPProjectionAuthority(digest, exact, binding); err == nil {
			t.Fatalf("%s: self-consistent invented HTTP projection minted closed authority", test.name)
		}
	}
}

func forgedHTTPProjection(
	t *testing.T,
	registryOverride domain.Digest,
	mutateSemantics, unknown bool,
) (domain.Digest, []byte, domain.ProjectionDefinitionBinding) {
	t.Helper()
	closed, err := NewHTTPProjectionAuthority()
	if err != nil {
		t.Fatal(err)
	}
	registryDigest := closed.Binding().FieldRegistryDigest()
	if registryOverride.Valid() {
		registryDigest = registryOverride
	}
	implementationDigest, err := digestExactBytes(
		"HTTPProjectionImplementation",
		[]byte("http-projection/v1\x00closed-four-field-registry\x00visible-validate-then-omit-operations\x00exact-operation-source-links\x00strict-json-object\x00exact-canonical"),
	)
	if err != nil {
		t.Fatal(err)
	}
	fields := closed.FieldIDs()
	configurationDigest, _, err := digestTyped("HTTPProjectionConfiguration", closedHTTPProjectionConfigIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPProjectionConfiguration",
		Version: httpProjectionVersionV1, Fields: fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	operations, err := closedHTTPOperationsV1()
	if err != nil {
		t.Fatal(err)
	}
	if mutateSemantics {
		operations[0].Semantics = "invented-visible-semantics"
		rule, digestErr := digestExactBytes(
			"HTTPProjectionOperationRule", []byte(operations[0].Name+"\x00"+operations[0].Semantics),
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
		AdapterDomain: domain.AdapterHTTP, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest,
		AcceptedChannels:    []string{"http.body", "http.headers", "http.status"},
		Operations:          bindingOperations, Comparator: domain.ProjectionComparatorExact,
		FieldRegistryDigest: registryDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := resolvedProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPProjectionDefinition", Version: httpProjectionVersionV1,
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
		object["unknown_closed_claim"] = "x"
		exact, err = canon.CanonicalizeTyped(object)
		if err != nil {
			t.Fatal(err)
		}
	}
	digestRaw, err := canon.DigestBytes("HTTPProjectionDefinition", exact)
	if err != nil {
		t.Fatal(err)
	}
	return domain.MustDigest(digestRaw.String()), exact, binding
}
