package projectionprofile

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestProfileBindsExactBindingTranslatorAndOrderedDescriptors(t *testing.T) {
	binding := testBinding(t, domain.AdapterCLI, "base")
	fields := []Descriptor{
		{FieldID: "cli.stdout.bytes", Channel: "stdout", SourcePath: []string{"bytes"}, SourceKind: "BYTES", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagBytes},
		{FieldID: "cli.stderr.text", Channel: "stderr", SourcePath: []string{"utf8_text"}, SourceKind: "UTF8_STRING", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagString},
	}
	profile, err := NewDerived(DerivedConfig{
		TranslatorName: "CLI_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: binding, Fields: fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !profile.Valid() || profile.Digest() == binding.Digest() || profile.Digest() == binding.FieldRegistryDigest() {
		t.Fatal("profile did not establish its distinct exact identity")
	}
	if !bytes.Contains(profile.CanonicalBytes(), []byte(base64.StdEncoding.EncodeToString(binding.CanonicalBytes()))) {
		t.Fatal("profile identity omitted the exact projection binding bytes")
	}
	fields[0].SourcePath[0] = "mutated"
	got := profile.Fields()
	if got[0].SourcePath[0] != "bytes" {
		t.Fatal("profile retained caller-owned path storage")
	}
	got[0].SourcePath[0] = "mutated-again"
	if profile.Fields()[0].SourcePath[0] != "bytes" {
		t.Fatal("profile exposed retained path storage")
	}

	reversed, err := NewDerived(DerivedConfig{
		TranslatorName: "CLI_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: binding,
		Fields: []Descriptor{fields[1], {FieldID: "cli.stdout.bytes", Channel: "stdout", SourcePath: []string{"bytes"}, SourceKind: "BYTES", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagBytes}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reversed.Digest() == profile.Digest() {
		t.Fatal("profile descriptor order did not affect identity")
	}

	changedTranslator, err := NewDerived(DerivedConfig{
		TranslatorName: "CLI_PROJECTION_TO_PORTABLE", TranslatorVersion: "v2", Binding: binding, Fields: profile.Fields(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if changedTranslator.Digest() == profile.Digest() {
		t.Fatal("translator version did not affect profile identity")
	}
}

func TestProfileRefusesInvalidDescriptorsAndMappings(t *testing.T) {
	binding := testBinding(t, domain.AdapterHTTP, "invalid")
	valid := Descriptor{FieldID: "http.status", Channel: "http.status", SourcePath: []string{"status"}, SourceKind: "SAFE_INTEGER", MissingPolicy: "REJECT_CAPTURE", PortableTag: portablevalue.TagInteger}
	tests := []struct {
		name   string
		fields []Descriptor
		code   Code
	}{
		{name: "empty", fields: nil, code: CodeLimitExceeded},
		{name: "duplicate", fields: []Descriptor{valid, valid}, code: CodeDuplicateField},
		{name: "empty path", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourceKind: valid.SourceKind, MissingPolicy: valid.MissingPolicy, PortableTag: valid.PortableTag}}, code: CodeInvalidDescriptor},
		{name: "unaccepted channel", fields: []Descriptor{{FieldID: valid.FieldID, Channel: "http.body", SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: valid.MissingPolicy, PortableTag: valid.PortableTag}}, code: CodeInvalidDescriptor},
		{name: "wrong mapping", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: valid.MissingPolicy, PortableTag: portablevalue.TagString}}, code: CodeUnsupportedMapping},
		{name: "unknown source", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: "UNKNOWN", MissingPolicy: valid.MissingPolicy, PortableTag: portablevalue.TagString}}, code: CodeUnsupportedMapping},
		{name: "unknown missing policy", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: "TOTALLY_UNKNOWN", PortableTag: valid.PortableTag}}, code: CodeUnsupportedMapping},
		{name: "tagged missing normalized false", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: "TAGGED_MISSING", PortableTag: valid.PortableTag}}, code: CodeUnsupportedMapping},
		{name: "null permission unsupported", fields: []Descriptor{{FieldID: valid.FieldID, Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: valid.MissingPolicy, PortableTag: valid.PortableTag, AllowNull: true}}, code: CodeUnsupportedMapping},
		{name: "aliased source", fields: []Descriptor{valid, {FieldID: "http.status.alias", Channel: valid.Channel, SourcePath: valid.SourcePath, SourceKind: valid.SourceKind, MissingPolicy: valid.MissingPolicy, PortableTag: valid.PortableTag}}, code: CodeDuplicateField},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDerived(DerivedConfig{TranslatorName: "HTTP_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: binding, Fields: test.fields})
			if !IsCode(err, test.code) {
				t.Fatalf("NewDerived() error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestProfileRejectsUnknownMissingPolicy(t *testing.T) {
	binding := testBinding(t, domain.AdapterHTTP, "unknown-policy")
	field := Descriptor{
		FieldID: "http.status", Channel: "http.status", SourcePath: []string{"status"},
		SourceKind: "SAFE_INTEGER", MissingPolicy: "TOTALLY_UNKNOWN", PortableTag: portablevalue.TagInteger,
	}
	_, err := NewDerived(DerivedConfig{
		TranslatorName: "HTTP_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: binding, Fields: []Descriptor{field},
	})
	if !IsCode(err, CodeUnsupportedMapping) {
		t.Fatalf("NewDerived(unknown missing policy) error = %v, want %s", err, CodeUnsupportedMapping)
	}
}

func TestProfileBindingDifferenceChangesIdentity(t *testing.T) {
	field := Descriptor{FieldID: "http.status", Channel: "http.status", SourcePath: []string{"status"}, SourceKind: "SAFE_INTEGER", MissingPolicy: "REJECT_CAPTURE", PortableTag: portablevalue.TagInteger}
	left, err := NewDerived(DerivedConfig{TranslatorName: "HTTP_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: testBinding(t, domain.AdapterHTTP, "left"), Fields: []Descriptor{field}})
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewDerived(DerivedConfig{TranslatorName: "HTTP_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: testBinding(t, domain.AdapterHTTP, "right"), Fields: []Descriptor{field}})
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest() == right.Digest() || bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes()) {
		t.Fatal("different exact projection bindings collapsed to one profile")
	}
}

func TestEveryProfileIdentityDimensionChangesTheDigest(t *testing.T) {
	binding := testBinding(t, domain.AdapterCLI, "dimensions")
	baseField := Descriptor{
		FieldID: "cli.stdout.bytes", Channel: "stdout", SourcePath: []string{"bytes"},
		SourceKind: "BYTES", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagBytes,
	}
	base := DerivedConfig{TranslatorName: "CLI_PROJECTION_TO_PORTABLE", TranslatorVersion: "v1", Binding: binding, Fields: []Descriptor{baseField}}
	baseProfile, err := NewDerived(base)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*DerivedConfig){
		"translator name":    func(config *DerivedConfig) { config.TranslatorName = "CLI_PROJECTION_TO_PORTABLE_OTHER" },
		"translator version": func(config *DerivedConfig) { config.TranslatorVersion = "v2" },
		"field id":           func(config *DerivedConfig) { config.Fields[0].FieldID = "cli.stdout.bytes.other" },
		"channel":            func(config *DerivedConfig) { config.Fields[0].Channel = "stderr" },
		"source path":        func(config *DerivedConfig) { config.Fields[0].SourcePath = []string{"bytes", "exact"} },
		"source kind and tag": func(config *DerivedConfig) {
			config.Fields[0].SourceKind = "UTF8_STRING"
			config.Fields[0].PortableTag = portablevalue.TagString
		},
		"missing semantics": func(config *DerivedConfig) {
			config.Fields[0].MissingPolicy = "TAGGED_MISSING"
			config.Fields[0].AllowMissing = true
		},
	} {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			config := DerivedConfig{
				TranslatorName: base.TranslatorName, TranslatorVersion: base.TranslatorVersion,
				Binding: base.Binding, Fields: []Descriptor{baseField},
			}
			mutate(&config)
			changed, err := NewDerived(config)
			if err != nil {
				t.Fatal(err)
			}
			if changed.Digest() == baseProfile.Digest() || bytes.Equal(changed.CanonicalBytes(), baseProfile.CanonicalBytes()) {
				t.Fatal("changed identity dimension collapsed")
			}
		})
	}
	for _, fragment := range [][]byte{
		[]byte(`"present_tag":"BYTES"`), []byte(`"allow_missing":false`), []byte(`"allow_null":false`),
	} {
		if !bytes.Contains(baseProfile.CanonicalBytes(), fragment) {
			t.Fatalf("profile identity omitted %s", fragment)
		}
	}
}

func TestZeroProfileIsInvalid(t *testing.T) {
	var profile Profile
	if profile.Valid() || profile.CanonicalBytes() != nil || profile.Fields() != nil {
		t.Fatal("zero profile acquired authority")
	}
}

func testBinding(t *testing.T, adapter domain.AdapterDomain, seed string) domain.ProjectionDefinitionBinding {
	t.Helper()
	implementation := testDigest(t, "implementation-"+seed)
	configuration := testDigest(t, "configuration-"+seed)
	registry := testDigest(t, "registry-"+seed)
	rule := testDigest(t, "rule-"+seed)
	channels := []string{"exit", "stderr", "stdout"}
	if adapter == domain.AdapterHTTP {
		channels = []string{"http.status"}
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: adapter, ImplementationDigest: implementation, ConfigurationDigest: configuration,
		AcceptedChannels: channels, Operations: []domain.ProjectionOperationBinding{{Name: "operation/v1", RuleDigest: rule}},
		Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func testDigest(t *testing.T, seed string) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes("ProjectionProfileTest", []byte(seed))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
