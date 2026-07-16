package contractsource

import (
	"bytes"
	"slices"

	cli "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	cliTranslatorNameV1  = "CLI_PROJECTION_TO_PORTABLE"
	httpTranslatorNameV1 = "HTTP_PROJECTION_TO_PORTABLE"
	portableTranslatorV1 = "v1"
)

var cliProfileRosterV1 = map[string]projectionprofile.Descriptor{
	"cli.completion.kind": {
		FieldID: "cli.completion.kind", Channel: "exit", SourcePath: []string{"completion", "kind"},
		SourceKind: "UTF8_STRING", MissingPolicy: "REJECT_CAPTURE", PortableTag: portablevalue.TagString,
	},
	"cli.exit.code": {
		FieldID: "cli.exit.code", Channel: "exit", SourcePath: []string{"completion", "code"},
		SourceKind: "SAFE_INTEGER", MissingPolicy: "TAGGED_MISSING_FOR_SIGNAL", PortableTag: portablevalue.TagInteger,
		AllowMissing: true,
	},
	"cli.exit.signal": {
		FieldID: "cli.exit.signal", Channel: "exit", SourcePath: []string{"completion", "signal"},
		SourceKind: "UTF8_STRING", MissingPolicy: "TAGGED_MISSING_FOR_EXIT", PortableTag: portablevalue.TagString,
		AllowMissing: true,
	},
	"cli.stdout.bytes": {
		FieldID: "cli.stdout.bytes", Channel: "stdout", SourcePath: []string{"bytes"},
		SourceKind: "BYTES", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagBytes,
	},
	"cli.stderr.text": {
		FieldID: "cli.stderr.text", Channel: "stderr", SourcePath: []string{"utf8_text"},
		SourceKind: "UTF8_STRING", MissingPolicy: "REJECT_CHANNEL", PortableTag: portablevalue.TagString,
	},
	"cli.stdout.json.mode": {
		FieldID: "cli.stdout.json.mode", Channel: "stdout", SourcePath: []string{"strict_json", "mode"},
		SourceKind: "UTF8_STRING", MissingPolicy: "TAGGED_MISSING", PortableTag: portablevalue.TagString,
		AllowMissing: true,
	},
	"cli.stdout.json.source": {
		FieldID: "cli.stdout.json.source", Channel: "stdout", SourcePath: []string{"strict_json", "source"},
		SourceKind: "UTF8_STRING", MissingPolicy: "TAGGED_MISSING", PortableTag: portablevalue.TagString,
		AllowMissing: true,
	},
}

var cliProfileOrderV1 = []string{
	"cli.completion.kind", "cli.exit.code", "cli.exit.signal", "cli.stdout.bytes",
	"cli.stderr.text", "cli.stdout.json.mode", "cli.stdout.json.source",
}

var httpProfileRosterV1 = []projectionprofile.Descriptor{
	{
		FieldID: "http.status", Channel: "http.status", SourcePath: []string{"status"},
		SourceKind: "SAFE_INTEGER", MissingPolicy: "REJECT_CAPTURE", PortableTag: portablevalue.TagInteger,
	},
	{
		FieldID: "http.header.content-type", Channel: "http.headers", SourcePath: []string{"content-type"},
		SourceKind: "ORDERED_STRING_LIST", MissingPolicy: "TAGGED_MISSING", PortableTag: portablevalue.TagOrderedStringList,
		AllowMissing: true,
	},
	{
		FieldID: "http.body.kind", Channel: "http.body", SourcePath: []string{"strict_json", "kind"},
		SourceKind: "UTF8_STRING", MissingPolicy: "REJECT_BODY", PortableTag: portablevalue.TagString,
	},
	{
		FieldID: "http.body.metadata", Channel: "http.body", SourcePath: []string{"strict_json", "metadata"},
		SourceKind: "CANONICAL_JSON_OBJECT", MissingPolicy: "REJECT_BODY", PortableTag: portablevalue.TagCanonicalJSON,
	},
}

func validateCLIProjection(
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection cli.CLIProjectionAuthority,
) error {
	if !exactBinding(plan.ProjectionDefinitionBinding(), projection.Binding()) ||
		!exactBinding(profile.Binding(), projection.Binding()) ||
		profile.AdapterDomain() != domain.AdapterCLI || profile.TranslatorName() != cliTranslatorNameV1 ||
		profile.TranslatorVersion() != portableTranslatorV1 {
		return refuse(CodeSourceAuthorityMismatch, "CLI plan, profile, and projection binding differ", nil)
	}
	fields, projectionFields := profile.Fields(), projection.FieldIDs()
	if len(fields) == 0 || len(fields) != len(projectionFields) {
		return refuse(CodeSourceAuthorityMismatch, "CLI profile roster differs from the adapter projection", nil)
	}
	previousRosterIndex := -1
	for index, field := range fields {
		expected, ok := cliProfileRosterV1[projectionFields[index]]
		rosterIndex := slices.Index(cliProfileOrderV1, projectionFields[index])
		if !ok || rosterIndex <= previousRosterIndex || field.FieldID != projectionFields[index] ||
			!equalDescriptor(field, expected) {
			return refuse(CodeSourceAuthorityMismatch, "CLI profile is outside the closed v1 descriptor roster", nil)
		}
		previousRosterIndex = rosterIndex
	}
	return nil
}

func validateHTTPProjection(
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection counterhttp.HTTPProjectionAuthority,
) error {
	if !exactBinding(plan.ProjectionDefinitionBinding(), projection.Binding()) ||
		!exactBinding(profile.Binding(), projection.Binding()) ||
		profile.AdapterDomain() != domain.AdapterHTTP || profile.TranslatorName() != httpTranslatorNameV1 ||
		profile.TranslatorVersion() != portableTranslatorV1 {
		return refuse(CodeSourceAuthorityMismatch, "HTTP plan, profile, and projection binding differ", nil)
	}
	fields, projectionFields := profile.Fields(), projection.FieldIDs()
	if len(fields) != len(httpProfileRosterV1) || len(projectionFields) != len(httpProfileRosterV1) {
		return refuse(CodeSourceAuthorityMismatch, "HTTP profile roster differs from the fixed adapter projection", nil)
	}
	for index, field := range fields {
		if projectionFields[index] != httpProfileRosterV1[index].FieldID ||
			!equalDescriptor(field, httpProfileRosterV1[index]) {
			return refuse(CodeSourceAuthorityMismatch, "HTTP profile is outside the closed v1 descriptor roster", nil)
		}
	}
	return nil
}

func exactBinding(left, right domain.ProjectionDefinitionBinding) bool {
	return left.Valid() && right.Valid() && left.Digest() == right.Digest() &&
		bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes())
}

func equalDescriptor(left, right projectionprofile.Descriptor) bool {
	return left.FieldID == right.FieldID && left.Channel == right.Channel &&
		slices.Equal(left.SourcePath, right.SourcePath) && left.SourceKind == right.SourceKind &&
		left.MissingPolicy == right.MissingPolicy && left.PortableTag == right.PortableTag &&
		left.AllowMissing == right.AllowMissing && left.AllowNull == right.AllowNull
}
