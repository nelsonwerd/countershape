package choice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
)

func TestU6SchemasAndExamplesMatchStrictRuntimeRecords(t *testing.T) {
	choicepointBytes := strictExampleBytes(t, "choicepoint.valid.json")
	decisionBytes := strictExampleBytes(t, "decision-record.valid.json")

	choicepoint, err := ParseChoicepointRecord(choicepointBytes)
	if err != nil || !choicepoint.Valid() {
		t.Fatalf("checked-in Choicepoint example is not a strict runtime record: %v", err)
	}
	decision, err := ParseDecisionRecord(decisionBytes, choicepoint)
	if err != nil || !decision.Valid() || decision.ChoicepointDigest() != choicepoint.Digest() {
		t.Fatalf("checked-in DecisionRecord example is not bound to the strict Choicepoint: %v", err)
	}

	assertSchemaMatchesExactExample(t, "choicepoint.schema.json", choicepointBytes, choicepointMemberCount)
	assertSchemaMatchesExactExample(t, "decision-record.schema.json", decisionBytes, decisionMemberCount)
	assertNoDecodedHostOrCredentialMarkers(t, "choicepoint.valid.json", choicepointBytes)
	assertNoDecodedHostOrCredentialMarkers(t, "decision-record.valid.json", decisionBytes)
}

func strictExampleBytes(t testing.TB, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(repositoryPath(t, "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || raw[len(raw)-1] != '\n' || bytes.Contains(raw[:len(raw)-1], []byte{'\n'}) {
		t.Fatalf("%s must contain one canonical JSON line plus one trailing newline", name)
	}
	exact := append([]byte(nil), raw[:len(raw)-1]...)
	value, err := canon.Parse(exact)
	if err != nil {
		t.Fatalf("%s is outside the canonical profile: %v", name, err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		t.Fatalf("%s is not exact canonical JSON: %v", name, err)
	}
	return exact
}

func assertSchemaMatchesExactExample(t testing.TB, schemaName string, example []byte, wantMembers int) {
	t.Helper()
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	raw, err := os.ReadFile(repositoryPath(t, "spec", "schema", "v1", schemaName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode %s: %v", schemaName, err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(example, &object); err != nil {
		t.Fatal(err)
	}
	if len(schema.Required) != wantMembers || len(schema.Properties) != wantMembers || len(object) != wantMembers {
		t.Fatalf("%s/example members = required:%d properties:%d example:%d, want %d",
			schemaName, len(schema.Required), len(schema.Properties), len(object), wantMembers)
	}
	required := append([]string(nil), schema.Required...)
	properties := make([]string, 0, len(schema.Properties))
	exampleKeys := make([]string, 0, len(object))
	for key := range schema.Properties {
		properties = append(properties, key)
	}
	for key := range object {
		exampleKeys = append(exampleKeys, key)
	}
	sort.Strings(required)
	sort.Strings(properties)
	sort.Strings(exampleKeys)
	if !equalStrings(required, properties) || !equalStrings(required, exampleKeys) {
		t.Fatalf("%s required/properties/example key sets disagree", schemaName)
	}
	if _, selfDigest := object["artifact_digest"]; selfDigest {
		t.Fatalf("%s example contains a self-referential artifact_digest", schemaName)
	}
}

func equalStrings(left, right []string) bool {
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

func assertNoDecodedHostOrCredentialMarkers(t testing.TB, name string, exact []byte) {
	t.Helper()
	var root any
	if err := json.Unmarshal(exact, &root); err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"/Users/", "/private/", "/tmp/", "TOKEN", "PASSWORD", "SECRET", "API_KEY"}
	var visit func(any, string)
	visit = func(value any, path string) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				next := path + "/" + key
				if text, ok := child.(string); ok && strings.HasSuffix(key, "_base64") && text != "" {
					decoded, err := base64.StdEncoding.Strict().DecodeString(text)
					if err != nil {
						t.Fatalf("%s %s is not strict base64: %v", name, next, err)
					}
					for _, marker := range forbidden {
						if bytes.Contains(decoded, []byte(marker)) {
							t.Fatalf("%s %s decoded bytes contain forbidden marker %q", name, next, marker)
						}
					}
					var nested any
					if json.Unmarshal(decoded, &nested) == nil {
						visit(nested, next+"#decoded")
					}
				}
				visit(child, next)
			}
		case []any:
			for index, child := range typed {
				visit(child, path+"/"+strconv.Itoa(index))
			}
		}
	}
	visit(root, "")
}

func repositoryPath(t testing.TB, parts ...string) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve repository path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	return filepath.Join(append([]string{root}, parts...)...)
}
