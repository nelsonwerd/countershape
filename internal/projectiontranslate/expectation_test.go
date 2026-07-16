package projectiontranslate

import (
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestPortableExpectationDomainCLICompletionRealizability(t *testing.T) {
	domain := cliExpectationDomain(t,
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal,
	)
	missing := portablevalue.Missing()
	exited := portableString(t, "EXITED")
	signaled := portableString(t, "SIGNALED")
	zero := portableInteger(t, "0")
	max := portableInteger(t, "255")
	signal := portableString(t, "SIGTERM")
	maxSignal := portableString(t, strings.Repeat("s", 1024))

	for name, selected := range map[string][]SelectedValue{
		"exited complete": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: exited},
			{FieldID: string(cli.CLIFieldExitCode), Value: zero},
			{FieldID: string(cli.CLIFieldExitSignal), Value: missing},
		},
		"signaled complete": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: signaled},
			{FieldID: string(cli.CLIFieldExitCode), Value: missing},
			{FieldID: string(cli.CLIFieldExitSignal), Value: signal},
		},
		"maximum exit":   {{FieldID: string(cli.CLIFieldExitCode), Value: max}},
		"missing code":   {{FieldID: string(cli.CLIFieldExitCode), Value: missing}},
		"missing signal": {{FieldID: string(cli.CLIFieldExitSignal), Value: missing}},
		"maximum signal": {{FieldID: string(cli.CLIFieldExitSignal), Value: maxSignal}},
	} {
		if err := domain.ValidateSelected(selected); err != nil {
			t.Fatalf("%s: realizable selected expectation was refused: %v", name, err)
		}
	}

	for name, selected := range map[string][]SelectedValue{
		"unknown completion": {{FieldID: string(cli.CLIFieldCompletionKind), Value: portableString(t, "CRASHED")}},
		"negative exit":      {{FieldID: string(cli.CLIFieldExitCode), Value: portableInteger(t, "-1")}},
		"large exit":         {{FieldID: string(cli.CLIFieldExitCode), Value: portableInteger(t, "256")}},
		"empty signal":       {{FieldID: string(cli.CLIFieldExitSignal), Value: portableString(t, "")}},
		"large signal":       {{FieldID: string(cli.CLIFieldExitSignal), Value: portableString(t, strings.Repeat("s", 1025))}},
		"exited missing code": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: exited},
			{FieldID: string(cli.CLIFieldExitCode), Value: missing},
		},
		"exited with signal": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: exited},
			{FieldID: string(cli.CLIFieldExitSignal), Value: signal},
		},
		"signaled with code": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: signaled},
			{FieldID: string(cli.CLIFieldExitCode), Value: zero},
		},
		"signaled missing signal": {
			{FieldID: string(cli.CLIFieldCompletionKind), Value: signaled},
			{FieldID: string(cli.CLIFieldExitSignal), Value: missing},
		},
		"code and signal present": {
			{FieldID: string(cli.CLIFieldExitCode), Value: zero},
			{FieldID: string(cli.CLIFieldExitSignal), Value: signal},
		},
		"code and signal missing": {
			{FieldID: string(cli.CLIFieldExitCode), Value: missing},
			{FieldID: string(cli.CLIFieldExitSignal), Value: missing},
		},
	} {
		if err := domain.ValidateSelected(selected); !IsCode(err, CodeCustomExpectationNotRealizable) {
			t.Fatalf("%s: unrealizable selected expectation error = %v, want %s", name, err, CodeCustomExpectationNotRealizable)
		}
	}
}

func TestPortableExpectationDomainCLIStdoutCoherence(t *testing.T) {
	bytesOnly := cliExpectationDomain(t, cli.CLIFieldStdoutBytes)
	requireExpectationAccepted(t, bytesOnly, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, []byte{0xff, 0x00})},
	})

	full := cliExpectationDomain(t, cli.CLIFieldStdoutBytes, cli.CLIFieldStdoutJSONMode, cli.CLIFieldStdoutJSONSource)
	requireExpectationAccepted(t, full, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, []byte(`{ "mode" : "safe" }`))},
		{FieldID: string(cli.CLIFieldStdoutJSONMode), Value: portableString(t, "safe")},
		{FieldID: string(cli.CLIFieldStdoutJSONSource), Value: portablevalue.Missing()},
	})
	requireExpectationAccepted(t, full, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutJSONMode), Value: portableString(t, "any")},
	})

	for name, body := range map[string][]byte{
		"non json":              []byte("not-json"),
		"array":                 []byte(`[]`),
		"duplicate":             []byte(`{"mode":"a","mode":"b"}`),
		"trailing":              []byte(`{"mode":"a"}x`),
		"active wrong type":     []byte(`{"mode":3}`),
		"unselected wrong type": []byte(`{"source":3}`),
	} {
		selected := []SelectedValue{{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, body)}}
		if err := full.ValidateSelected(selected); !IsCode(err, CodeCustomExpectationNotRealizable) {
			t.Fatalf("%s: unrealizable stdout expectation error = %v, want %s", name, err, CodeCustomExpectationNotRealizable)
		}
	}
	requireExpectationRefused(t, full, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, []byte(`{"mode":"actual"}`))},
		{FieldID: string(cli.CLIFieldStdoutJSONMode), Value: portableString(t, "other")},
	})
	requireExpectationRefused(t, full, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, []byte(`{"mode":"actual"}`))},
		{FieldID: string(cli.CLIFieldStdoutJSONMode), Value: portablevalue.Missing()},
	})

	modeOnly := cliExpectationDomain(t, cli.CLIFieldStdoutBytes, cli.CLIFieldStdoutJSONMode)
	requireExpectationAccepted(t, modeOnly, []SelectedValue{
		{FieldID: string(cli.CLIFieldStdoutBytes), Value: portableBytes(t, []byte(`{"mode":"safe","source":3}`))},
		{FieldID: string(cli.CLIFieldStdoutJSONMode), Value: portableString(t, "safe")},
	})
}

func TestPortableExpectationDomainHTTPRealizability(t *testing.T) {
	domain := httpExpectationDomain(t)
	for _, status := range []string{"200", "401", "599"} {
		requireExpectationAccepted(t, domain, []SelectedValue{
			{FieldID: string(counterhttp.HTTPFieldStatus), Value: portableInteger(t, status)},
		})
	}
	for _, status := range []string{"199", "600"} {
		requireExpectationRefused(t, domain, []SelectedValue{
			{FieldID: string(counterhttp.HTTPFieldStatus), Value: portableInteger(t, status)},
		})
	}
	requireExpectationAccepted(t, domain, []SelectedValue{
		{FieldID: string(counterhttp.HTTPFieldContentType), Value: portablevalue.Missing()},
	})
	requireExpectationAccepted(t, domain, []SelectedValue{
		{FieldID: string(counterhttp.HTTPFieldContentType), Value: portableList(t, []string{"", "application/json", "application/json"})},
	})
	for _, members := range [][]string{{}, {"ok\x1f"}, {"ok\x7f"}, {"line\nbreak"}, {"café"}} {
		requireExpectationRefused(t, domain, []SelectedValue{
			{FieldID: string(counterhttp.HTTPFieldContentType), Value: portableList(t, members)},
		})
	}
	requireExpectationAccepted(t, domain, []SelectedValue{
		{FieldID: string(counterhttp.HTTPFieldBodyKind), Value: portableString(t, "")},
		{FieldID: string(counterhttp.HTTPFieldBodyMetadata), Value: portableJSON(t, []byte(`{}`))},
	})
	for _, exact := range [][]byte{[]byte(`1`), []byte(`[]`)} {
		requireExpectationRefused(t, domain, []SelectedValue{
			{FieldID: string(counterhttp.HTTPFieldBodyMetadata), Value: portableJSON(t, exact)},
		})
	}
}

func TestPortableExpectationDomainBindsExactProfileAndRuleRoster(t *testing.T) {
	stdout := cliExpectationDomain(t, cli.CLIFieldStdoutBytes)
	stderr := cliExpectationDomain(t, cli.CLIFieldStderrText)
	http := httpExpectationDomain(t)
	if !stdout.Valid() || !stderr.Valid() || !http.Valid() ||
		stdout.Digest() == stderr.Digest() || stdout.Digest() == http.Digest() ||
		stdout.ProfileDigest() == stderr.ProfileDigest() {
		t.Fatal("portable expectation domains lost exact profile, adapter, or rule-roster identity")
	}
	if err := stdout.ValidateSelected([]SelectedValue{
		{FieldID: string(cli.CLIFieldStderrText), Value: portableString(t, "foreign")},
	}); !IsCode(err, CodeProfileMismatch) {
		t.Fatalf("foreign exact profile field error = %v, want %s", err, CodeProfileMismatch)
	}
}

func cliExpectationDomain(t testing.TB, fields ...cli.CLIFieldID) ExpectationDomain {
	t.Helper()
	definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	domain, err := resolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	return domain
}

func httpExpectationDomain(t testing.TB) ExpectationDomain {
	t.Helper()
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	domain, err := resolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	return domain
}

func portableString(t testing.TB, text string) portablevalue.Value {
	t.Helper()
	value, err := portablevalue.String(text)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func portableInteger(t testing.TB, canonical string) portablevalue.Value {
	t.Helper()
	value, err := portablevalue.Integer(canonical)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func portableBytes(t testing.TB, exact []byte) portablevalue.Value {
	t.Helper()
	value, err := portablevalue.Bytes(exact)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func portableList(t testing.TB, members []string) portablevalue.Value {
	t.Helper()
	value, err := portablevalue.OrderedStringList(members)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func portableJSON(t testing.TB, exact []byte) portablevalue.Value {
	t.Helper()
	value, err := portablevalue.CanonicalJSON(exact)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func requireExpectationAccepted(t testing.TB, domain ExpectationDomain, selected []SelectedValue) {
	t.Helper()
	if err := domain.ValidateSelected(selected); err != nil {
		t.Fatalf("realizable selected expectation was refused: %v", err)
	}
}

func requireExpectationRefused(t testing.TB, domain ExpectationDomain, selected []SelectedValue) {
	t.Helper()
	if err := domain.ValidateSelected(selected); !IsCode(err, CodeCustomExpectationNotRealizable) {
		t.Fatalf("unrealizable selected expectation error = %v, want %s", err, CodeCustomExpectationNotRealizable)
	}
}
