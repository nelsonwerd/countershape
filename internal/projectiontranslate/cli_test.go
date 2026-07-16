package projectiontranslate

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

type cliFieldFixture struct {
	fieldID string
	value   map[string]any
}

func TestCLITranslationPreservesEveryHistoricalTagAndProfileOrder(t *testing.T) {
	resolved := resolveCLIFields(t,
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal,
		cli.CLIFieldStdoutBytes, cli.CLIFieldStderrText, cli.CLIFieldStdoutJSONMode, cli.CLIFieldStdoutJSONSource,
	)
	exact := cliProjectionFixture(t, []cliFieldFixture{
		{string(cli.CLIFieldCompletionKind), map[string]any{"tag": "STRING", "value": "EXITED"}},
		{string(cli.CLIFieldExitCode), map[string]any{"canonical": "2", "tag": "INTEGER"}},
		{string(cli.CLIFieldExitSignal), map[string]any{"tag": "MISSING"}},
		{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": base64.StdEncoding.EncodeToString([]byte{0xff, 0x00, 0x80}), "tag": "BYTES"}},
		{string(cli.CLIFieldStderrText), map[string]any{"tag": "STRING", "value": "diagnostic"}},
		{string(cli.CLIFieldStdoutJSONMode), map[string]any{"tag": "STRING", "value": ""}},
		{string(cli.CLIFieldStdoutJSONSource), map[string]any{"tag": "MISSING"}},
	})
	original := append([]byte(nil), exact...)
	tuple, err := resolved.Translate(exact)
	if err != nil {
		t.Fatal(err)
	}
	if !tuple.Valid() || tuple.ProfileDigest() != resolved.Profile().Digest() {
		t.Fatal("translated CLI tuple is not profile-bound and valid")
	}
	fields := tuple.Fields()
	wantTags := []portablevalue.Tag{
		portablevalue.TagString, portablevalue.TagInteger, portablevalue.TagMissing, portablevalue.TagBytes,
		portablevalue.TagString, portablevalue.TagString, portablevalue.TagMissing,
	}
	for index, field := range fields {
		if field.ID() != resolved.Profile().Fields()[index].FieldID || field.Value().Tag() != wantTags[index] {
			t.Fatalf("field %d = (%q,%s), want (%q,%s)", index, field.ID(), field.Value().Tag(), resolved.Profile().Fields()[index].FieldID, wantTags[index])
		}
	}
	bytesValue, ok := fields[3].Value().BytesValue()
	if !ok || !bytes.Equal(bytesValue, []byte{0xff, 0x00, 0x80}) {
		t.Fatalf("translated CLI bytes = (%x,%t)", bytesValue, ok)
	}
	exact[0] ^= 1
	bytesValue[0] = 0
	again, _ := tuple.Fields()[3].Value().BytesValue()
	if !bytes.Equal(again, []byte{0xff, 0x00, 0x80}) || bytes.Equal(exact, original) {
		t.Fatal("CLI translation exposed input or tuple byte storage")
	}
	strict, err := StrictTranslate(resolved.Profile(), original)
	if err != nil || !tuple.Equal(strict) {
		t.Fatalf("StrictTranslate() = (%v,%v)", strict.Fields(), err)
	}
}

func TestCLITranslationFreezesLiteralHistoricalBytes(t *testing.T) {
	resolved := resolveCLIFields(t, cli.CLIFieldStdoutBytes)
	exact := cliProjectionFixture(t, []cliFieldFixture{
		{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": "/wCA", "tag": "BYTES"}},
	})
	want := []byte(`{"fields":[{"field_id":"cli.stdout.bytes","value":{"base64":"/wCA","tag":"BYTES"}}],"kind":"CLIProjection","schema_version":"cli-projection/v1"}`)
	if !bytes.Equal(exact, want) {
		t.Fatalf("historical CLI fixture = %s, want %s", exact, want)
	}
	if _, err := resolved.Translate(exact); err != nil {
		t.Fatal(err)
	}
}

func TestCLITranslatorRejectsMalformedHistoricalMatrix(t *testing.T) {
	bytesResolved := resolveCLIFields(t, cli.CLIFieldStdoutBytes)
	codeResolved := resolveCLIFields(t, cli.CLIFieldExitCode)
	modeResolved := resolveCLIFields(t, cli.CLIFieldStdoutJSONMode)
	validBytes := cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": "AA==", "tag": "BYTES"}}})
	extraRoot := cliProjectionFixtureWithRoot(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": "AA==", "tag": "BYTES"}}}, map[string]any{"extra": "zero"})
	tests := []struct {
		name     string
		resolved Resolved
		exact    []byte
		code     Code
	}{
		{name: "leading whitespace", resolved: bytesResolved, exact: append([]byte(" "), validBytes...), code: CodeNoncanonical},
		{name: "extra root", resolved: bytesResolved, exact: extraRoot, code: CodeInvalidProjection},
		{name: "wrong version", resolved: bytesResolved, exact: bytes.ReplaceAll(validBytes, []byte("cli-projection/v1"), []byte("cli-projection/v2")), code: CodeInvalidProjection},
		{name: "wrong roster", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStderrText), map[string]any{"tag": "STRING", "value": "x"}}}), code: CodeRosterMismatch},
		{name: "extra missing slot", resolved: modeResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutJSONMode), map[string]any{"tag": "MISSING", "value": ""}}}), code: CodeInvalidPayload},
		{name: "missing disallowed", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"tag": "MISSING"}}}), code: CodeInvalidPayload},
		{name: "unsafe integer spelling", resolved: codeResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldExitCode), map[string]any{"canonical": "-0", "tag": "INTEGER"}}}), code: CodeInvalidPayload},
		{name: "unpadded base64", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": "AA", "tag": "BYTES"}}}), code: CodeInvalidPayload},
		{name: "wrong tag for field", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"tag": "STRING", "value": "AA=="}}}), code: CodeInvalidPayload},
		{name: "unknown tag", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"tag": "UNKNOWN"}}}), code: CodeInvalidTag},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.resolved.Translate(test.exact); !IsCode(err, test.code) {
				t.Fatalf("Translate() error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestCLITranslatorRejectsSameTypedFieldReordering(t *testing.T) {
	resolved := resolveCLIFields(t, cli.CLIFieldStderrText, cli.CLIFieldStdoutJSONMode)
	exact := cliProjectionFixture(t, []cliFieldFixture{
		{string(cli.CLIFieldStdoutJSONMode), map[string]any{"tag": "STRING", "value": "mode"}},
		{string(cli.CLIFieldStderrText), map[string]any{"tag": "STRING", "value": "diagnostic"}},
	})
	if _, err := resolved.Translate(exact); !IsCode(err, CodeRosterMismatch) {
		t.Fatalf("Translate(reordered same-type fields) error = %v", err)
	}
}

func TestCLITranslatorRejectsNoncanonicalProjectionBytes(t *testing.T) {
	resolved := resolveCLIFields(t, cli.CLIFieldStdoutBytes)
	valid := cliProjectionFixture(t, []cliFieldFixture{{
		string(cli.CLIFieldStdoutBytes), map[string]any{"base64": "AA==", "tag": "BYTES"},
	}})
	if _, err := resolved.Translate(append([]byte(" "), valid...)); !IsCode(err, CodeNoncanonical) {
		t.Fatalf("Translate(noncanonical projection) error = %v, want %s", err, CodeNoncanonical)
	}
}

func TestCLITranslatorEnforcesPortableResourceCeilings(t *testing.T) {
	stringResolved := resolveCLIFields(t, cli.CLIFieldStdoutJSONMode)
	bytesResolved := resolveCLIFields(t, cli.CLIFieldStdoutBytes)
	fullResolved := resolveCLIFields(t,
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal,
		cli.CLIFieldStdoutBytes, cli.CLIFieldStderrText, cli.CLIFieldStdoutJSONMode, cli.CLIFieldStdoutJSONSource,
	)
	largeString := strings.Repeat("x", portablevalue.MaxStringBytes+1)
	largeBytes := make([]byte, portablevalue.MaxBytesValueBytes+1)
	boundaryString := strings.Repeat("x", portablevalue.MaxStringBytes)
	boundaryBytes := make([]byte, portablevalue.MaxBytesValueBytes)
	aggregate := cliProjectionFixture(t, []cliFieldFixture{
		{string(cli.CLIFieldCompletionKind), map[string]any{"tag": "STRING", "value": boundaryString}},
		{string(cli.CLIFieldExitCode), map[string]any{"canonical": "2", "tag": "INTEGER"}},
		{string(cli.CLIFieldExitSignal), map[string]any{"tag": "STRING", "value": boundaryString}},
		{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": base64.StdEncoding.EncodeToString(boundaryBytes), "tag": "BYTES"}},
		{string(cli.CLIFieldStderrText), map[string]any{"tag": "STRING", "value": boundaryString}},
		{string(cli.CLIFieldStdoutJSONMode), map[string]any{"tag": "STRING", "value": boundaryString}},
		{string(cli.CLIFieldStdoutJSONSource), map[string]any{"tag": "STRING", "value": boundaryString}},
	})
	tests := []struct {
		name     string
		resolved Resolved
		exact    []byte
	}{
		{name: "string bytes", resolved: stringResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutJSONMode), map[string]any{"tag": "STRING", "value": largeString}}})},
		{name: "decoded bytes", resolved: bytesResolved, exact: cliProjectionFixture(t, []cliFieldFixture{{string(cli.CLIFieldStdoutBytes), map[string]any{"base64": base64.StdEncoding.EncodeToString(largeBytes), "tag": "BYTES"}}})},
		{name: "aggregate tuple", resolved: fullResolved, exact: aggregate},
		{name: "outer bytes", resolved: bytesResolved, exact: bytes.Repeat([]byte("x"), canon.MaxInputBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.resolved.Translate(test.exact); !IsCode(err, CodeTranslationLimit) {
				t.Fatalf("Translate() error = %v, want %s", err, CodeTranslationLimit)
			}
		})
	}
}

func resolveCLIFields(t *testing.T, fields ...cli.CLIFieldID) Resolved {
	t.Helper()
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func cliProjectionFixture(t *testing.T, fields []cliFieldFixture) []byte {
	t.Helper()
	return cliProjectionFixtureWithRoot(t, fields, nil)
}

func cliProjectionFixtureWithRoot(t *testing.T, fields []cliFieldFixture, extra map[string]any) []byte {
	t.Helper()
	encodedFields := make([]map[string]any, len(fields))
	for index, field := range fields {
		encodedFields[index] = map[string]any{"field_id": field.fieldID, "value": field.value}
	}
	root := map[string]any{"fields": encodedFields, "kind": "CLIProjection", "schema_version": "cli-projection/v1"}
	for key, value := range extra {
		root[key] = value
	}
	exact, err := canon.CanonicalizeTyped(root)
	if err != nil {
		t.Fatal(err)
	}
	return exact
}
