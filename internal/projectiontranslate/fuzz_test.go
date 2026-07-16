package projectiontranslate

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func FuzzStrictProjectionTranslation(f *testing.F) {
	cliDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStdoutBytes}})
	if err != nil {
		f.Fatal(err)
	}
	cliResolved, err := Resolve(cliDefinition.Binding())
	if err != nil {
		f.Fatal(err)
	}
	httpDefinition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		f.Fatal(err)
	}
	httpResolved, err := Resolve(httpDefinition.Binding())
	if err != nil {
		f.Fatal(err)
	}
	cliSeed := []byte(`{"fields":[{"field_id":"cli.stdout.bytes","value":{"base64":"/wCA","tag":"BYTES"}}],"kind":"CLIProjection","schema_version":"cli-projection/v1"}`)
	httpSeed, err := canon.CanonicalizeTyped(map[string]any{
		"Fields": validHTTPFieldFixtures(), "Kind": "HTTPProjection", "SchemaVersion": "countershape/v1",
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(false, cliSeed)
	f.Add(true, httpSeed)
	f.Add(false, []byte(`{}`))
	f.Fuzz(func(t *testing.T, useHTTP bool, exact []byte) {
		resolved := cliResolved
		if useHTTP {
			resolved = httpResolved
		}
		tuple, err := resolved.Translate(exact)
		if err != nil {
			return
		}
		if !tuple.Valid() {
			t.Fatal("successful translation returned an invalid tuple")
		}
		parsed, err := canon.Parse(exact)
		if err != nil {
			t.Fatalf("accepted projection did not parse under canon: %v", err)
		}
		canonical, err := parsed.CanonicalChecked()
		if err != nil || !bytes.Equal(canonical, exact) {
			t.Fatalf("accepted projection was not exact canonical input: %v", err)
		}
		fields := tuple.Fields()
		profileFields := resolved.Profile().Fields()
		if len(fields) != len(profileFields) {
			t.Fatalf("accepted tuple fields = %d, profile fields = %d", len(fields), len(profileFields))
		}
		for index := range fields {
			if fields[index].ID() != profileFields[index].FieldID {
				t.Fatalf("accepted tuple field %d = %q, want %q", index, fields[index].ID(), profileFields[index].FieldID)
			}
		}
		strict, err := StrictTranslate(resolved.Profile(), exact)
		if err != nil || !tuple.Equal(strict) {
			t.Fatalf("accepted translation differs from strict inert-profile translation: %v", err)
		}
		if len(fields) > 0 {
			fields[0] = Field{}
			if tuple.Fields()[0].ID() == "" {
				t.Fatal("Tuple.Fields exposed tuple slice storage")
			}
		}
		again, err := resolved.Translate(exact)
		if err != nil || !tuple.Equal(again) {
			t.Fatalf("successful translation was nondeterministic: %v", err)
		}
	})
}

func FuzzCLIBytesTranslation(f *testing.F) {
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStdoutBytes}})
	if err != nil {
		f.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x00, 0x80})
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > portablevalue.MaxBytesValueBytes {
			return
		}
		exact, err := canon.CanonicalizeTyped(map[string]any{
			"fields": []map[string]any{{
				"field_id": string(cli.CLIFieldStdoutBytes),
				"value":    map[string]any{"base64": base64.StdEncoding.EncodeToString(input), "tag": "BYTES"},
			}},
			"kind": "CLIProjection", "schema_version": "cli-projection/v1",
		})
		if err != nil {
			t.Fatal(err)
		}
		tuple, err := resolved.Translate(exact)
		if err != nil {
			t.Fatal(err)
		}
		value, ok := tuple.Value(string(cli.CLIFieldStdoutBytes))
		got, bytesOK := value.BytesValue()
		if !ok || !bytesOK || !equalTestBytes(got, input) {
			t.Fatalf("translated bytes = (%x,%t,%t), want %x", got, ok, bytesOK, input)
		}
	})
}

func FuzzHTTPOrderedListTranslation(f *testing.F) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		f.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		f.Fatal(err)
	}
	f.Add("", "application/json", "application/json")
	f.Add("a", "b", "a")
	f.Fuzz(func(t *testing.T, first, second, third string) {
		values := []string{first, second, third, first}
		for _, value := range values {
			if !printableASCII(value) {
				return
			}
		}
		retained := 0
		for _, value := range values {
			if len(value) > portablevalue.MaxListMemberBytes {
				return
			}
			retained += len(value)
		}
		canonicalList, err := canon.CanonicalizeTyped(values)
		if err != nil || retained > portablevalue.MaxListAggregateBytes || len(canonicalList) > portablevalue.MaxListCanonicalBytes {
			return
		}
		fields := validHTTPFieldFixtures()
		fields[1]["Strings"] = values
		exact, err := canon.CanonicalizeTyped(map[string]any{
			"Fields": fields, "Kind": "HTTPProjection", "SchemaVersion": "countershape/v1",
		})
		if err != nil {
			t.Fatal(err)
		}
		tuple, err := resolved.Translate(exact)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := tuple.Value(string(counterhttp.HTTPFieldContentType))
		members, listOK := got.OrderedStrings()
		if !ok || !listOK || !equalTestStrings(members, values) {
			t.Fatalf("translated list = (%q,%t,%t), want %q", members, ok, listOK, values)
		}
	})
}

func printableASCII(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

func equalTestBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
