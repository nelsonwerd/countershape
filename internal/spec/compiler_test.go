package spec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const validSourceTemplate = `{
  "schema_version":"countershape-source/v1",
  "kind":"SourceSpec",
  "candidate_set_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111",
  "materialization_policy_digest":"sha256:2222222222222222222222222222222222222222222222222222222222222222",
  "comparison_envelope_digest":"sha256:3333333333333333333333333333333333333333333333333333333333333333",
  "adapter":{"domain":"CLI","adapter_version":"cli/v1","runner_digest":"sha256:4444444444444444444444444444444444444444444444444444444444444444"},
  "execution_shape":"ONE_CLI_INVOCATION",
  "start_argv":["node","fixture/cli.mjs","--mode","test"],
  "environment":[{"name":"LANG","value":"C"}],
  "fixture_recipe_digest":"sha256:5555555555555555555555555555555555555555555555555555555555555555",
  "capture_policy_digest":"sha256:6666666666666666666666666666666666666666666666666666666666666666",
  "projection_definition_digest":"sha256:7777777777777777777777777777777777777777777777777777777777777777",
  "required_tools":[{"name":"node","version_constraint":"executed-major-only"}]
}`

var (
	testProjection = mustProjectionBindingForTest()
	validSource    = strings.Replace(validSourceTemplate,
		"sha256:7777777777777777777777777777777777777777777777777777777777777777",
		testProjection.Digest().String(), 1)
)

func mustProjectionBindingForTest() domain.ProjectionDefinitionBinding {
	digest := func(hex byte) domain.Digest {
		return domain.MustDigest("sha256:" + strings.Repeat(string(hex), 64))
	}
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: digest('8'), ConfigurationDigest: digest('9'),
		AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations:       []domain.ProjectionOperationBinding{{Name: "capture-cli", RuleDigest: digest('a')}},
		Comparator:       domain.ProjectionComparatorExact, FieldRegistryDigest: digest('b'),
	})
	if err != nil {
		panic(err)
	}
	return binding
}

func mustSource(t *testing.T, exact string) ParsedSource {
	t.Helper()
	source, err := ParseSource([]byte(exact))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func TestParseSourceAndCompileMaterializeEveryDefault(t *testing.T) {
	source := mustSource(t, validSource)
	plan, err := Compile(source, testProjection)
	if err != nil {
		t.Fatal(err)
	}
	wantBudgets := DefaultBudgets()
	wantBudgets.ReadinessMS = 0
	if plan.Budgets() != wantBudgets {
		t.Fatalf("budgets = %#v, want %#v", plan.Budgets(), wantBudgets)
	}
	wantRepeats := plan.RepeatSchedule()
	if wantRepeats.DiscoveryRepeats != 3 || wantRepeats.ConfirmationRepeats != 3 {
		t.Fatalf("repeat defaults were not materialized: %#v", wantRepeats)
	}
	second, err := Compile(mustSource(t, validSource), testProjection)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest() != second.Digest() || !bytes.Equal(plan.CanonicalBytes(), second.CanonicalBytes()) {
		t.Fatal("same inert source did not produce exact deterministic plan bytes")
	}
}

func TestOwnedSourceExampleParsesAndCompiles(t *testing.T) {
	exact, err := os.ReadFile("../../spec/examples/v1/source-spec.valid.json")
	if err != nil {
		t.Fatal(err)
	}
	example := strings.Replace(string(exact), "sha256:7777777777777777777777777777777777777777777777777777777777777777", testProjection.Digest().String(), 1)
	if _, err := Compile(mustSource(t, example), testProjection); err != nil {
		t.Fatal(err)
	}
}

func TestCompileRefusesUnparsedZeroValue(t *testing.T) {
	if _, err := Compile(ParsedSource{}, testProjection); codeOf(err) != CodeUnparsedSource {
		t.Fatalf("code = %q, want %q (err=%v)", codeOf(err), CodeUnparsedSource, err)
	}
}

func TestCompileRequiresExactResolvedProjectionDefinition(t *testing.T) {
	source := mustSource(t, validSource)
	if _, err := Compile(source, domain.ProjectionDefinitionBinding{}); codeOf(err) != CodeUnresolvedProjectionDefinition {
		t.Fatalf("zero binding code = %q, want %q (err=%v)", codeOf(err), CodeUnresolvedProjectionDefinition, err)
	}
	other, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain:        domain.AdapterCLI,
		ImplementationDigest: domain.MustDigest("sha256:" + strings.Repeat("c", 64)),
		ConfigurationDigest:  domain.MustDigest("sha256:" + strings.Repeat("d", 64)),
		AcceptedChannels:     []string{"exit"},
		Operations:           []domain.ProjectionOperationBinding{{Name: "other", RuleDigest: domain.MustDigest("sha256:" + strings.Repeat("e", 64))}},
		Comparator:           domain.ProjectionComparatorExact,
		FieldRegistryDigest:  domain.MustDigest("sha256:" + strings.Repeat("f", 64)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(source, other); codeOf(err) != CodeUnresolvedProjectionDefinition {
		t.Fatalf("mismatched binding code = %q, want %q (err=%v)", codeOf(err), CodeUnresolvedProjectionDefinition, err)
	}
}

func TestParseSourceRefusesUnknownKeysAtEveryLevel(t *testing.T) {
	cases := []string{
		strings.Replace(validSource, `"kind":"SourceSpec",`, `"kind":"SourceSpec","surprise":true,`, 1),
		strings.Replace(validSource, `"adapter_version":"cli/v1",`, `"adapter_version":"cli/v1","surprise":true,`, 1),
		strings.Replace(validSource, `{"name":"LANG","value":"C"}`, `{"name":"LANG","value":"C","surprise":true}`, 1),
		strings.Replace(validSource, `"required_tools":[`, `"secret_slots":[{"name":"TOKEN","presence":"ABSENT","surprise":true}],"required_tools":[`, 1),
		strings.Replace(validSource, `"required_tools":[`, `"readiness":{"kind":"NONE","surprise":true},"required_tools":[`, 1),
		strings.Replace(validSource, `"required_tools":[`, `"repeat_schedule":{"discovery_repeats":3,"surprise":true},"required_tools":[`, 1),
		strings.Replace(validSource, `"version_constraint":"executed-major-only"`, `"version_constraint":"executed-major-only","surprise":true`, 1),
		strings.Replace(validSource, `"required_tools":[`, `"budgets":{"probe_ms":3,"surprise":true},"required_tools":[`, 1),
	}
	for _, exact := range cases {
		_, err := ParseSource([]byte(exact))
		if codeOf(err) != CodeUnknownSourceKey {
			t.Fatalf("code = %q, want %q (err=%v)", codeOf(err), CodeUnknownSourceKey, err)
		}
	}
}

func TestParseSourceRefusalSelectionIsDeterministic(t *testing.T) {
	missingTwo := strings.Replace(validSource, `  "schema_version":"countershape-source/v1",
  "kind":"SourceSpec",
`, "", 1)
	var first string
	for index := 0; index < 50; index++ {
		_, err := ParseSource([]byte(missingTwo))
		if codeOf(err) != CodeMissingSourceField {
			t.Fatalf("missing-field code = %q (err=%v)", codeOf(err), err)
		}
		if index == 0 {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("refusal changed between runs: %q then %q", first, err)
		}
	}

	badBudgets := strings.Replace(validSource, `"required_tools":[`, `"budgets":{"probe_ms":"bad","candidate_count":"bad"},"required_tools":[`, 1)
	_, err := ParseSource([]byte(badBudgets))
	if err == nil || !strings.Contains(err.Error(), "candidate_count") {
		t.Fatalf("budget refusal did not follow the fixed field order: %v", err)
	}
}

func TestParseSourceRefusesMissingRequiredAndWrongType(t *testing.T) {
	missing := strings.Replace(validSource, `  "start_argv":["node","fixture/cli.mjs","--mode","test"],
`, "", 1)
	if _, err := ParseSource([]byte(missing)); codeOf(err) != CodeMissingSourceField {
		t.Fatalf("missing code = %q (err=%v)", codeOf(err), err)
	}
	wrong := strings.Replace(validSource, `"execution_shape":"ONE_CLI_INVOCATION"`, `"execution_shape":7`, 1)
	if _, err := ParseSource([]byte(wrong)); codeOf(err) != CodeInvalidSourceType {
		t.Fatalf("wrong-type code = %q (err=%v)", codeOf(err), err)
	}
}

func TestParseSourceRefusesDuplicateBeforeTypedConstruction(t *testing.T) {
	duplicate := strings.Replace(validSource, `"kind":"SourceSpec",`, `"kind":"SourceSpec","kind":"SourceSpec",`, 1)
	if _, err := ParseSource([]byte(duplicate)); err == nil || !strings.Contains(err.Error(), "DUPLICATE_KEY") {
		t.Fatalf("duplicate name was not refused by strict parser: %v", err)
	}
}

func TestParseSourceRefusesOversizedAndSemanticallyInvalidSources(t *testing.T) {
	if _, err := ParseSource(make([]byte, maxSourceBytes+1)); codeOf(err) != CodeSourceTooLarge {
		t.Fatalf("oversized code = %q (err=%v)", codeOf(err), err)
	}
	badDomain := strings.Replace(validSource, `"domain":"CLI"`, `"domain":"UNKNOWN"`, 1)
	if _, err := ParseSource([]byte(badDomain)); err == nil {
		t.Fatal("unsupported adapter domain received a ParsedSource capability")
	}
	badBudget := strings.Replace(validSource, `"required_tools":[`, `"budgets":{"candidate_count":1},"required_tools":[`, 1)
	if _, err := ParseSource([]byte(badBudget)); err == nil {
		t.Fatal("out-of-range budget received a ParsedSource capability")
	}
}

func TestParseSourceRefusesExecutionGrammarBypasses(t *testing.T) {
	cases := map[string]string{
		"env assignments":       strings.Replace(validSource, `["node","fixture/cli.mjs","--mode","test"]`, `["env","API_TOKEN=secret","node","fixture/cli.mjs"]`, 1),
		"env split shell":       strings.Replace(validSource, `["node","fixture/cli.mjs","--mode","test"]`, `["env","-S","sh -c 'echo hi'"]`, 1),
		"absolute executable":   strings.Replace(validSource, `["node","fixture/cli.mjs","--mode","test"]`, `["/usr/bin/node","fixture/cli.mjs"]`, 1),
		"absolute option":       strings.Replace(validSource, `"--mode","test"`, `"--dest=/tmp/out","test"`, 1),
		"ambient interpolation": strings.Replace(validSource, `"--mode","test"`, `"--mode","$HOME"`, 1),
		"secret in public env":  strings.Replace(validSource, `{"name":"LANG","value":"C"}`, `{"name":"LANG","value":"credential-bytes"}`, 1),
		"environment NUL":       strings.Replace(validSource, `{"name":"LANG","value":"C"}`, `{"name":"LANG","value":"C\u0000"}`, 1),
		"wrong adapter version": strings.Replace(validSource, `"adapter_version":"cli/v1"`, `"adapter_version":"http/v1"`, 1),
		"undeclared executable": strings.Replace(validSource, `"required_tools":[{"name":"node","version_constraint":"executed-major-only"}]`, `"required_tools":[]`, 1),
	}
	for name, exact := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseSource([]byte(exact)); err == nil {
				t.Fatal("execution-grammar bypass received a ParsedSource capability")
			}
		})
	}
}

func TestParseSourceRefusesAdapterReadinessCrossProduct(t *testing.T) {
	httpWithoutReadiness := strings.NewReplacer(
		`"domain":"CLI"`, `"domain":"HTTP"`,
		`"adapter_version":"cli/v1"`, `"adapter_version":"http/v1"`,
		`"execution_shape":"ONE_CLI_INVOCATION"`, `"execution_shape":"ONE_LOOPBACK_HTTP_REQUEST"`,
	).Replace(validSource)
	if _, err := ParseSource([]byte(httpWithoutReadiness)); err == nil {
		t.Fatal("HTTP without fixture-owned readiness received authority")
	}

	cliWithReadiness := strings.Replace(validSource, `"required_tools":[`, `"readiness":{"kind":"FIXTURE_OWNED_SIGNAL","signal_name":"ready"},"required_tools":[`, 1)
	if _, err := ParseSource([]byte(cliWithReadiness)); err == nil {
		t.Fatal("CLI with service readiness received authority")
	}

	httpReady := strings.Replace(httpWithoutReadiness, `"required_tools":[`, `"readiness":{"kind":"FIXTURE_OWNED_SIGNAL","signal_name":"ready"},"required_tools":[`, 1)
	httpReady = strings.Replace(httpReady, `"required_tools":[`, `"budgets":{"readiness_ms":0},"required_tools":[`, 1)
	if _, err := ParseSource([]byte(httpReady)); err == nil {
		t.Fatal("HTTP with a zero readiness budget received authority")
	}
}

func TestParseSourceEnforcesPreExecutionCollectionAndStringBounds(t *testing.T) {
	long := strings.Repeat("x", maxSourceStringBytes+1)
	oversizedString := strings.Replace(validSource, `"test"]`, `"`+long+`"]`, 1)
	if _, err := ParseSource([]byte(oversizedString)); codeOf(err) != CodeInvalidSourceValue {
		t.Fatalf("long string code = %q (err=%v)", codeOf(err), err)
	}

	entries := `{"name":"LANG","value":"C"},{"name":"LC_ALL","value":"C"},{"name":"TZ","value":"UTC"},{"name":"NO_COLOR","value":"1"},{"name":"NODE_NO_WARNINGS","value":"1"},{"name":"LANG","value":"C"}`
	tooManyEnvironment := strings.Replace(validSource, `{"name":"LANG","value":"C"}`, entries, 1)
	if _, err := ParseSource([]byte(tooManyEnvironment)); codeOf(err) != CodeInvalidSourceValue {
		t.Fatalf("environment bound code = %q (err=%v)", codeOf(err), err)
	}

	tools := make([]string, maxRequiredToolItems+1)
	for index := range tools {
		tools[index] = fmt.Sprintf(`{"name":"tool%d","version_constraint":"v1"}`, index)
	}
	tooManyTools := strings.Replace(validSource, `{"name":"node","version_constraint":"executed-major-only"}`, strings.Join(tools, ","), 1)
	if _, err := ParseSource([]byte(tooManyTools)); codeOf(err) != CodeInvalidSourceValue {
		t.Fatalf("tool bound code = %q (err=%v)", codeOf(err), err)
	}
}

func TestCompileDistinguishesMissingOverrideFromExplicitZero(t *testing.T) {
	exact := strings.Replace(validSource, `"required_tools":[`, `"budgets":{"probe_ms":0},"required_tools":[`, 1)
	if _, err := ParseSource([]byte(exact)); err == nil {
		t.Fatal("explicit invalid zero was silently defaulted")
	}
}

func TestCompilePreservesOrderedArgvButCanonicalizesObjectOrder(t *testing.T) {
	left := mustSource(t, validSource)
	rightText := strings.Replace(validSource, `["node","fixture/cli.mjs","--mode","test"]`, `["node","fixture/cli.mjs","test","--mode"]`, 1)
	right := mustSource(t, rightText)
	leftPlan, err := Compile(left, testProjection)
	if err != nil {
		t.Fatal(err)
	}
	rightPlan, err := Compile(right, testProjection)
	if err != nil {
		t.Fatal(err)
	}
	if leftPlan.Digest() == rightPlan.Digest() {
		t.Fatal("ordered argv collapsed into unordered identity")
	}

	reordered := strings.Replace(validSource, `"schema_version":"countershape-source/v1",
  "kind":"SourceSpec",`, `"kind":"SourceSpec",
  "schema_version":"countershape-source/v1",`, 1)
	reorderedSource := mustSource(t, reordered)
	if !bytes.Equal(left.CanonicalBytes(), reorderedSource.CanonicalBytes()) || left.Digest() != reorderedSource.Digest() {
		t.Fatal("object member order changed canonical source bytes")
	}
}

func TestParseSourceRejectsWorldPlanWireEnvelopeAsSource(t *testing.T) {
	withArtifact := strings.Replace(validSource, `"kind":"SourceSpec",`, `"kind":"SourceSpec","artifact_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",`, 1)
	if _, err := ParseSource([]byte(withArtifact)); codeOf(err) != CodeUnknownSourceKey {
		t.Fatalf("wire artifact field bypassed source grammar: %v", err)
	}
}

func FuzzCompileMalformedValuesNeverPanic(f *testing.F) {
	f.Add([]byte(validSource))
	f.Add([]byte(`{"schema_version":"countershape-source/v1","kind":"SourceSpec"}`))
	f.Add([]byte{0xff, '{', '}'})
	f.Fuzz(func(t *testing.T, exact []byte) {
		source, err := ParseSource(exact)
		if err == nil {
			_, _ = Compile(source, testProjection)
		}
	})
}

func codeOf(err error) string {
	if err == nil {
		return ""
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return ""
}
