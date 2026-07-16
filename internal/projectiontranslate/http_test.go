package projectiontranslate

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestHTTPTranslationPreservesStatusOrderedDuplicatesAndMetadata(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	exact := emittedHTTPProjection(t, 500, []string{"application/json", "text/plain", "application/json"})
	original := append([]byte(nil), exact...)
	tuple, err := resolved.Translate(exact)
	if err != nil {
		t.Fatal(err)
	}
	if !tuple.Valid() || len(tuple.Fields()) != 4 {
		t.Fatal("HTTP translation did not produce one complete four-field tuple")
	}
	status, _ := tuple.Value(string(counterhttp.HTTPFieldStatus))
	if got, ok := status.IntegerText(); !ok || got != "500" {
		t.Fatalf("status = (%q,%t)", got, ok)
	}
	contentType, _ := tuple.Value(string(counterhttp.HTTPFieldContentType))
	gotList, ok := contentType.OrderedStrings()
	wantList := []string{"application/json", "text/plain", "application/json"}
	if !ok || !equalTestStrings(gotList, wantList) {
		t.Fatalf("content type = (%q,%t), want %q", gotList, ok, wantList)
	}
	kind, _ := tuple.Value(string(counterhttp.HTTPFieldBodyKind))
	if got, ok := kind.StringText(); !ok || got != "invoice" {
		t.Fatalf("body kind = (%q,%t)", got, ok)
	}
	metadata, _ := tuple.Value(string(counterhttp.HTTPFieldBodyMetadata))
	if got, ok := metadata.CanonicalJSONBytes(); !ok || string(got) != `{"owner":"tenant"}` {
		t.Fatalf("metadata = (%q,%t)", got, ok)
	}
	exact[0] ^= 1
	gotList[0] = "mutated"
	again, _ := tuple.Value(string(counterhttp.HTTPFieldContentType))
	againList, _ := again.OrderedStrings()
	if !equalTestStrings(againList, wantList) || bytes.Equal(exact, original) {
		t.Fatal("HTTP translation exposed input or ordered-list storage")
	}
}

func TestHTTPTranslationKeepsMissingEmptyListAndPresentEmptyDistinct(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	absent, err := resolved.Translate(emittedHTTPProjection(t, 200, nil))
	if err != nil {
		t.Fatal(err)
	}
	presentEmpty, err := resolved.Translate(emittedHTTPProjection(t, 200, []string{""}))
	if err != nil {
		t.Fatal(err)
	}
	absentValue, _ := absent.Value(string(counterhttp.HTTPFieldContentType))
	presentValue, _ := presentEmpty.Value(string(counterhttp.HTTPFieldContentType))
	emptyList, err := portablevalue.OrderedStringList([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if absentValue.Tag() != portablevalue.TagMissing || presentValue.Tag() != portablevalue.TagOrderedStringList ||
		absentValue.Equal(presentValue) || absentValue.Equal(emptyList) || presentValue.Equal(emptyList) {
		t.Fatal("HTTP missing, present-empty member, and portable empty list collapsed")
	}
	members, _ := presentValue.OrderedStrings()
	if len(members) != 1 || members[0] != "" {
		t.Fatalf("present-empty content type = %q", members)
	}
}

func TestHTTPTranslationFreezesHistoricalCapitalizedWire(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	exact := emittedHTTPProjection(t, 200, nil)
	for _, fragment := range [][]byte{
		[]byte(`{"Fields":[`), []byte(`"Kind":"HTTPProjection"`), []byte(`"SchemaVersion":"countershape/v1"`),
		[]byte(`"CanonicalJSON":""`), []byte(`"FieldID":"http.status"`), []byte(`"Strings":[]`),
	} {
		if !bytes.Contains(exact, fragment) {
			t.Fatalf("historical HTTP projection omitted %s: %s", fragment, exact)
		}
	}
	if _, err := resolved.Translate(exact); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPTranslatorRejectsMalformedHistoricalMatrix(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	valid := emittedHTTPProjection(t, 500, []string{"application/json"})
	statusString := bytes.Replace(valid, []byte(`"String":""`), []byte(`"String":"not-zero"`), 1)
	statusNullStrings := bytes.Replace(valid, []byte(`"Strings":[]`), []byte(`"Strings":null`), 1)
	statusOutOfRange := bytes.Replace(valid, []byte(`"Integer":500`), []byte(`"Integer":199`), 1)
	absent := emittedHTTPProjection(t, 200, nil)
	emptyTaggedList := bytes.Replace(absent, []byte(`"Tag":"MISSING"`), []byte(`"Tag":"STRING_LIST"`), 1)
	fields := validHTTPFieldFixtures()
	fields[3]["CanonicalJSON"] = `{"z":1,"a":2}`
	noncanonicalMetadata := httpProjectionFixture(t, fields, nil)
	reordered := validHTTPFieldFixtures()
	reordered[0], reordered[1] = reordered[1], reordered[0]
	tests := []struct {
		name  string
		exact []byte
		code  Code
	}{
		{name: "leading whitespace", exact: append([]byte(" "), valid...), code: CodeNoncanonical},
		{name: "status unused string", exact: statusString, code: CodeInvalidPayload},
		{name: "status strings null", exact: statusNullStrings, code: CodeInvalidPayload},
		{name: "status outside historical range", exact: statusOutOfRange, code: CodeInvalidPayload},
		{name: "empty string list tag", exact: emptyTaggedList, code: CodeInvalidTag},
		{name: "noncanonical metadata object", exact: noncanonicalMetadata, code: CodeInvalidPayload},
		{name: "reordered fields", exact: httpProjectionFixture(t, reordered, nil), code: CodeRosterMismatch},
		{name: "extra root", exact: httpProjectionFixture(t, validHTTPFieldFixtures(), map[string]any{"Extra": ""}), code: CodeInvalidProjection},
	}
	for _, dirty := range []struct {
		name       string
		fieldIndex int
		slot       string
		value      any
	}{
		{name: "status canonical JSON", fieldIndex: 0, slot: "CanonicalJSON", value: `{}`},
		{name: "status string", fieldIndex: 0, slot: "String", value: "x"},
		{name: "status strings", fieldIndex: 0, slot: "Strings", value: []string{"x"}},
		{name: "content type canonical JSON", fieldIndex: 1, slot: "CanonicalJSON", value: `{}`},
		{name: "content type integer", fieldIndex: 1, slot: "Integer", value: 1},
		{name: "content type string", fieldIndex: 1, slot: "String", value: "x"},
		{name: "body kind canonical JSON", fieldIndex: 2, slot: "CanonicalJSON", value: `{}`},
		{name: "body kind integer", fieldIndex: 2, slot: "Integer", value: 1},
		{name: "body kind strings", fieldIndex: 2, slot: "Strings", value: []string{"x"}},
		{name: "metadata integer", fieldIndex: 3, slot: "Integer", value: 1},
		{name: "metadata string", fieldIndex: 3, slot: "String", value: "x"},
		{name: "metadata strings", fieldIndex: 3, slot: "Strings", value: []string{"x"}},
	} {
		tests = append(tests, struct {
			name  string
			exact []byte
			code  Code
		}{name: "dirty unused " + dirty.name, exact: dirtyHTTPFixture(t, dirty.fieldIndex, dirty.slot, dirty.value), code: CodeInvalidPayload})
	}
	tests = append(tests,
		struct {
			name  string
			exact []byte
			code  Code
		}{name: "extra field member", exact: dirtyHTTPFixture(t, 3, "Extra", ""), code: CodeInvalidProjection},
		struct {
			name  string
			exact []byte
			code  Code
		}{name: "wrong integer slot type", exact: dirtyHTTPFixture(t, 3, "Integer", "0"), code: CodeInvalidPayload},
		struct {
			name  string
			exact []byte
			code  Code
		}{name: "metadata canonical scalar", exact: dirtyHTTPFixture(t, 3, "CanonicalJSON", `1`), code: CodeInvalidPayload},
	)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := resolved.Translate(test.exact); !IsCode(err, test.code) {
				t.Fatalf("Translate() error = %v, want %s; wire=%s", err, test.code, test.exact)
			}
		})
	}
}

func TestHTTPTranslatorRejectsDirtyMetadataUnusedSlot(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	dirty := dirtyHTTPFixture(t, 3, "String", "x")
	if _, err := resolved.Translate(dirty); !IsCode(err, CodeInvalidPayload) {
		t.Fatalf("Translate(dirty metadata unused slot) error = %v, want %s", err, CodeInvalidPayload)
	}
}

func TestHTTPTranslatorEnforcesPortableResourceCeilings(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	tooMany := make([]string, portablevalue.MaxListMembers+1)
	for index := range tooMany {
		tooMany[index] = "x"
	}
	listFields := validHTTPFieldFixtures()
	listFields[1]["Strings"] = tooMany
	memberFields := validHTTPFieldFixtures()
	memberFields[1]["Strings"] = []string{strings.Repeat("x", portablevalue.MaxListMemberBytes+1)}
	aggregateFields := validHTTPFieldFixtures()
	aggregateFields[1]["Strings"] = []string{strings.Repeat("x", 40*1024), strings.Repeat("y", 30*1024)}
	encodedListFields := validHTTPFieldFixtures()
	encodedListFields[1]["Strings"] = []string{strings.Repeat("\\", 40*1024)}
	metadataFields := validHTTPFieldFixtures()
	metadataFields[3]["CanonicalJSON"] = `{"x":"` + strings.Repeat("x", portablevalue.MaxCanonicalJSONBytes) + `"}`
	kindFields := validHTTPFieldFixtures()
	kindFields[2]["String"] = strings.Repeat("x", portablevalue.MaxStringBytes+1)
	tests := []struct {
		name  string
		exact []byte
	}{
		{name: "list count", exact: httpProjectionFixture(t, listFields, nil)},
		{name: "list member bytes", exact: httpProjectionFixture(t, memberFields, nil)},
		{name: "list aggregate bytes", exact: httpProjectionFixture(t, aggregateFields, nil)},
		{name: "list canonical bytes", exact: httpProjectionFixture(t, encodedListFields, nil)},
		{name: "metadata bytes", exact: httpProjectionFixture(t, metadataFields, nil)},
		{name: "body kind bytes", exact: httpProjectionFixture(t, kindFields, nil)},
		{name: "outer bytes", exact: bytes.Repeat([]byte("x"), canon.MaxInputBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := resolved.Translate(test.exact); !IsCode(err, CodeTranslationLimit) {
				t.Fatalf("Translate() error = %v, want %s", err, CodeTranslationLimit)
			}
		})
	}
}

func resolveHTTPFixed(t *testing.T) Resolved {
	t.Helper()
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func emittedHTTPProjection(t *testing.T, status int, contentTypes []string) []byte {
	t.Helper()
	body := []byte(`{"kind":"invoice","metadata":{"owner":"tenant"},"request_id":"volatile","scratch_root":"/tmp/root"}`)
	policy, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	var headers strings.Builder
	for _, value := range contentTypes {
		headers.WriteString("content-type: ")
		headers.WriteString(value)
		headers.WriteString("\r\n")
	}
	headers.WriteString(fmt.Sprintf("content-length: %d\r\n", len(body)))
	raw := []byte(fmt.Sprintf("HTTP/1.1 %d Result\r\n%s\r\n%s", status, headers.String(), body))
	response, err := counterhttp.ParseResponse(raw, policy)
	if err != nil {
		t.Fatalf("ParseResponse(%q): %v", raw, err)
	}
	exact, err := counterhttp.ProjectParsedResponse(response, "/tmp/root")
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func validHTTPFieldFixtures() []map[string]any {
	return []map[string]any{
		{"CanonicalJSON": "", "FieldID": string(counterhttp.HTTPFieldStatus), "Integer": 500, "String": "", "Strings": []string{}, "Tag": "INTEGER"},
		{"CanonicalJSON": "", "FieldID": string(counterhttp.HTTPFieldContentType), "Integer": 0, "String": "", "Strings": []string{"application/json"}, "Tag": "STRING_LIST"},
		{"CanonicalJSON": "", "FieldID": string(counterhttp.HTTPFieldBodyKind), "Integer": 0, "String": "invoice", "Strings": []string{}, "Tag": "STRING"},
		{"CanonicalJSON": `{"owner":"tenant"}`, "FieldID": string(counterhttp.HTTPFieldBodyMetadata), "Integer": 0, "String": "", "Strings": []string{}, "Tag": "CANONICAL_JSON"},
	}
}

func httpProjectionFixture(t *testing.T, fields []map[string]any, extra map[string]any) []byte {
	t.Helper()
	root := map[string]any{"Fields": fields, "Kind": "HTTPProjection", "SchemaVersion": "countershape/v1"}
	for key, value := range extra {
		root[key] = value
	}
	exact, err := canon.CanonicalizeTyped(root)
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func dirtyHTTPFixture(t *testing.T, fieldIndex int, slot string, value any) []byte {
	t.Helper()
	fields := validHTTPFieldFixtures()
	fields[fieldIndex][slot] = value
	return httpProjectionFixture(t, fields, nil)
}

func equalTestStrings(left, right []string) bool {
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
