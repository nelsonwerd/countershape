package projectiontranslate

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

func TestResolveEveryCLIFieldSubsetUniquelyAndInRegistryOrder(t *testing.T) {
	registry := cli.CLIFieldRegistry()
	seenProfiles := map[string]struct{}{}
	for mask := 1; mask < 1<<len(registry); mask++ {
		selected := make([]cli.CLIFieldID, 0, len(registry))
		wantIDs := make([]string, 0, len(registry))
		for index, descriptor := range registry {
			if mask&(1<<index) != 0 {
				selected = append(selected, descriptor.ID())
				wantIDs = append(wantIDs, string(descriptor.ID()))
			}
		}
		definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: selected})
		if err != nil {
			t.Fatalf("mask %03x definition: %v", mask, err)
		}
		resolved, err := Resolve(definition.Binding())
		if err != nil {
			t.Fatalf("mask %03x resolve: %v", mask, err)
		}
		if !resolved.Valid() || resolved.arm != armCLI {
			t.Fatalf("mask %03x did not produce sealed CLI authority", mask)
		}
		fields := resolved.Profile().Fields()
		if len(fields) != len(wantIDs) {
			t.Fatalf("mask %03x fields = %d, want %d", mask, len(fields), len(wantIDs))
		}
		for index := range fields {
			if fields[index].FieldID != wantIDs[index] {
				t.Fatalf("mask %03x field %d = %q, want %q", mask, index, fields[index].FieldID, wantIDs[index])
			}
		}
		key := resolved.Profile().Digest().String()
		if _, duplicate := seenProfiles[key]; duplicate {
			t.Fatalf("mask %03x collided on profile %s", mask, key)
		}
		seenProfiles[key] = struct{}{}
	}
	if len(seenProfiles) != 127 {
		t.Fatalf("unique CLI profiles = %d, want 127", len(seenProfiles))
	}
}

func TestResolveRequiresDigestAndExactBindingBytes(t *testing.T) {
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStdoutBytes}})
	if err != nil {
		t.Fatal(err)
	}
	digest := definition.Binding().Digest()
	exact := definition.Binding().CanonicalBytes()
	if !exactIdentityMatch(digest, exact, digest, append([]byte(nil), exact...)) {
		t.Fatal("identical digest and bytes did not match")
	}
	mutated := append([]byte(nil), exact...)
	mutated[len(mutated)-1] ^= 1
	if exactIdentityMatch(digest, exact, digest, mutated) {
		t.Fatal("digest-only match ignored exact binding bytes")
	}

	other, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStderrText}})
	if err != nil {
		t.Fatal(err)
	}
	if definition.FieldRegistryDigest() != other.FieldRegistryDigest() {
		t.Fatal("test precondition: CLI subsets should share the adapter registry digest")
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil || resolved.Profile().Fields()[0].FieldID != string(cli.CLIFieldStdoutBytes) {
		t.Fatalf("exact subset resolution = (%v,%v)", resolved.Profile().Fields(), err)
	}
}

func TestResolveFixedHTTPBindingAndRejectNearMatch(t *testing.T) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldContentType),
		string(counterhttp.HTTPFieldBodyKind), string(counterhttp.HTTPFieldBodyMetadata),
	}
	fields := resolved.Profile().Fields()
	if !resolved.Valid() || resolved.arm != armHTTP || len(fields) != len(want) {
		t.Fatalf("fixed HTTP resolution = (%v,%v)", fields, err)
	}
	for index := range want {
		if fields[index].FieldID != want[index] {
			t.Fatalf("HTTP field %d = %q, want %q", index, fields[index].FieldID, want[index])
		}
	}

	near := bindingWithChangedFirstRule(t, definition.Binding())
	if near.AdapterDomain() != domain.AdapterHTTP || near.FieldRegistryDigest() != definition.Binding().FieldRegistryDigest() {
		t.Fatal("near-match binding did not preserve domain and registry preconditions")
	}
	if _, err := Resolve(near); !IsCode(err, CodeProfileNotFound) {
		t.Fatalf("near-match HTTP Resolve() error = %v", err)
	}
}

func TestResolveRejectsEveryCompleteBindingNearMatchDimension(t *testing.T) {
	cliDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{
		Fields: []cli.CLIFieldID{cli.CLIFieldExitCode, cli.CLIFieldStdoutJSONMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	httpDefinition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*domain.ProjectionDefinitionBindingConfig){
		"implementation": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.ImplementationDigest = translateTestDigest(t, "near-implementation")
		},
		"configuration": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.ConfigurationDigest = translateTestDigest(t, "near-configuration")
		},
		"channels": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.AcceptedChannels = config.AcceptedChannels[:1]
		},
		"operation-name": func(config *domain.ProjectionDefinitionBindingConfig) { config.Operations[0].Name += "-near" },
		"operation-rule": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.Operations[0].RuleDigest = translateTestDigest(t, "near-operation")
		},
		"operation-order": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.Operations[0], config.Operations[1] = config.Operations[1], config.Operations[0]
		},
		"registry": func(config *domain.ProjectionDefinitionBindingConfig) {
			config.FieldRegistryDigest = translateTestDigest(t, "near-registry")
		},
	} {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			for _, source := range []domain.ProjectionDefinitionBinding{cliDefinition.Binding(), httpDefinition.Binding()} {
				near := rebuildBinding(t, source, mutate)
				if _, err := Resolve(near); !IsCode(err, CodeProfileNotFound) {
					t.Fatalf("Resolve(%s near %s) error = %v", source.AdapterDomain(), name, err)
				}
			}
		})
	}
}

func TestStrictTranslateReResolvesInertProfile(t *testing.T) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	for name, config := range map[string]projectionprofile.DerivedConfig{
		"version": {
			TranslatorName: resolved.Profile().TranslatorName(), TranslatorVersion: "forged-version",
			Binding: resolved.Profile().Binding(), Fields: resolved.Profile().Fields(),
		},
		"path": forgedProfileConfig(resolved.Profile(), func(fields []projectionprofile.Descriptor) {
			fields[0].SourcePath = []string{"status", "forged"}
		}),
		"policy": forgedProfileConfig(resolved.Profile(), func(fields []projectionprofile.Descriptor) {
			fields[0].MissingPolicy = "TAGGED_MISSING"
			fields[0].AllowMissing = true
		}),
		"roster": forgedProfileConfig(resolved.Profile(), func(fields []projectionprofile.Descriptor) {
			fields[0], fields[1] = fields[1], fields[0]
		}),
	} {
		forged, err := projectionprofile.NewDerived(config)
		if err != nil || !forged.Valid() {
			t.Fatalf("%s inert forged profile test precondition failed: %v", name, err)
		}
		if _, err := StrictTranslate(forged, []byte(`{}`)); !IsCode(err, CodeProfileMismatch) {
			t.Fatalf("StrictTranslate(%s forged) error = %v", name, err)
		}
	}
}

func TestZeroAndUnsupportedBindingsDoNotResolve(t *testing.T) {
	if _, err := Resolve(domain.ProjectionDefinitionBinding{}); !IsCode(err, CodeInvalidBinding) {
		t.Fatalf("Resolve(zero) error = %v", err)
	}
	var zero Resolved
	if zero.Valid() || zero.Profile().Valid() {
		t.Fatal("zero Resolved acquired authority")
	}
}

func bindingWithChangedFirstRule(t *testing.T, source domain.ProjectionDefinitionBinding) domain.ProjectionDefinitionBinding {
	t.Helper()
	operations := source.Operations()
	operations[0].RuleDigest = translateTestDigest(t, "changed-rule")
	rebuilt, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: source.AdapterDomain(), ImplementationDigest: source.ImplementationDigest(),
		ConfigurationDigest: source.ConfigurationDigest(), AcceptedChannels: source.AcceptedChannels(),
		Operations: operations, Comparator: source.Comparator(), FieldRegistryDigest: source.FieldRegistryDigest(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(rebuilt.CanonicalBytes(), source.CanonicalBytes()) || rebuilt.Digest() == source.Digest() {
		t.Fatal("changed binding unexpectedly retained identity")
	}
	return rebuilt
}

func rebuildBinding(t *testing.T, source domain.ProjectionDefinitionBinding, mutate func(*domain.ProjectionDefinitionBindingConfig)) domain.ProjectionDefinitionBinding {
	t.Helper()
	config := domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: source.AdapterDomain(), ImplementationDigest: source.ImplementationDigest(),
		ConfigurationDigest: source.ConfigurationDigest(), AcceptedChannels: source.AcceptedChannels(),
		Operations: source.Operations(), Comparator: source.Comparator(), FieldRegistryDigest: source.FieldRegistryDigest(),
	}
	mutate(&config)
	rebuilt, err := domain.NewProjectionDefinitionBinding(config)
	if err != nil {
		t.Fatal(err)
	}
	return rebuilt
}

func translateTestDigest(t *testing.T, seed string) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes("ProjectionTranslateTest", []byte(fmt.Sprintf("%s", seed)))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func forgedProfileConfig(profile projectionprofile.Profile, mutate func([]projectionprofile.Descriptor)) projectionprofile.DerivedConfig {
	fields := profile.Fields()
	mutate(fields)
	return projectionprofile.DerivedConfig{
		TranslatorName: profile.TranslatorName(), TranslatorVersion: profile.TranslatorVersion(),
		Binding: profile.Binding(), Fields: fields,
	}
}
