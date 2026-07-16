package contractsource

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

func TestClosedProjectionModelsMatchAdaptersAndPortableProfilesExhaustively(t *testing.T) {
	registry := []cli.CLIFieldID{
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal, cli.CLIFieldStdoutBytes,
		cli.CLIFieldStderrText, cli.CLIFieldStdoutJSONMode, cli.CLIFieldStdoutJSONSource,
	}
	capture, err := cli.NewCLICapturePolicy(cli.CLICapturePolicyConfig{StdoutBytes: 32 << 10, StderrBytes: 16 << 10})
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := cli.NewCLIFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	for mask := 1; mask < 1<<len(registry); mask++ {
		fields := make([]cli.CLIFieldID, 0, len(registry))
		fieldNames := make([]string, 0, len(registry))
		for index, field := range registry {
			if mask&(1<<index) != 0 {
				fields = append(fields, field)
				fieldNames = append(fieldNames, string(field))
			}
		}
		definition, definitionErr := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
		modelAuthority, modelErr := climodel.NewCLIProjectionAuthority(fieldNames)
		if definitionErr != nil || modelErr != nil || definition.Digest() != modelAuthority.Digest() ||
			!bytes.Equal(definition.CanonicalBytes(), modelAuthority.CanonicalBytes()) ||
			definition.Binding().Digest() != modelAuthority.Binding().Digest() ||
			!bytes.Equal(definition.Binding().CanonicalBytes(), modelAuthority.Binding().CanonicalBytes()) {
			t.Fatalf("CLI closed-schema parity failed for mask %07b: definition=%v model=%v", mask, definitionErr, modelErr)
		}
		resolved, resolveErr := projectiontranslate.Resolve(definition.Binding())
		if resolveErr != nil {
			t.Fatalf("CLI profile resolution failed for mask %07b: %v", mask, resolveErr)
		}
		plan := testPlan(
			t, domain.AdapterCLI, domain.OneCLIInvocation, []string{"node", "fixture/subject.mjs"},
			domain.Readiness{Kind: domain.ReadinessNone}, capture.Digest(), recipe.Digest(), definition.Binding(),
			domain.Budgets{StdoutBytes: capture.StdoutBytes(), StderrBytes: capture.StderrBytes(), HTTPBodyBytes: 64 << 10},
		)
		if err := validateCLIProjection(plan, resolved.Profile(), modelAuthority); err != nil {
			t.Fatalf("CLI source-profile roster parity failed for mask %07b: %v", mask, err)
		}
	}

	httpDefinition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	httpAuthority, err := httpmodel.NewHTTPProjectionAuthority()
	if err != nil || httpDefinition.Digest() != httpAuthority.Digest() ||
		!bytes.Equal(httpDefinition.CanonicalBytes(), httpAuthority.CanonicalBytes()) ||
		httpDefinition.Binding().Digest() != httpAuthority.Binding().Digest() ||
		!bytes.Equal(httpDefinition.Binding().CanonicalBytes(), httpAuthority.Binding().CanonicalBytes()) {
		t.Fatalf("HTTP closed-schema parity failed: definition=%v model=%v", httpDefinition.Valid(), err)
	}
	httpResolved, err := projectiontranslate.Resolve(httpDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	httpInput := testHTTPInput(t, false)
	if err := validateHTTPProjection(httpInput.Plan, httpResolved.Profile(), httpAuthority); err != nil {
		t.Fatalf("HTTP source-profile roster parity failed: %v", err)
	}
}

func TestPortableSourceRejectsProjectionProfileCrossPairsAndDrift(t *testing.T) {
	input := testCLIInput(t, false)
	wrongAuthority, err := climodel.NewCLIProjectionAuthority([]string{"cli.exit.signal"})
	if err != nil {
		t.Fatal(err)
	}
	crossPaired := input
	crossPaired.Projection = wrongAuthority
	if _, err := NewCLISource(crossPaired); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("projection authority cross-pair refusal = %v", err)
	}

	wrongTranslator, err := projectionprofile.NewDerived(projectionprofile.DerivedConfig{
		TranslatorName: "CLI_PROJECTION_TO_PORTABLE_DRIFT", TranslatorVersion: input.Profile.TranslatorVersion(),
		Binding: input.Profile.Binding(), Fields: input.Profile.Fields(),
	})
	if err != nil {
		t.Fatal(err)
	}
	translatorDrift := input
	translatorDrift.Profile = wrongTranslator
	if _, err := NewCLISource(translatorDrift); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("translator drift refusal = %v", err)
	}

	descriptors := input.Profile.Fields()
	descriptors[0].SourcePath = []string{"invented", "path"}
	wrongDescriptor, err := projectionprofile.NewDerived(projectionprofile.DerivedConfig{
		TranslatorName: input.Profile.TranslatorName(), TranslatorVersion: input.Profile.TranslatorVersion(),
		Binding: input.Profile.Binding(), Fields: descriptors,
	})
	if err != nil {
		t.Fatal(err)
	}
	descriptorDrift := input
	descriptorDrift.Profile = wrongDescriptor
	if _, err := NewCLISource(descriptorDrift); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("descriptor drift refusal = %v", err)
	}
}

func TestPortableSourceRejectsCLIProfileOrderAlias(t *testing.T) {
	input := testCLIInput(t, false)
	fields := input.Profile.Fields()
	slices.Reverse(fields)
	alias, err := projectionprofile.NewDerived(projectionprofile.DerivedConfig{
		TranslatorName: input.Profile.TranslatorName(), TranslatorVersion: input.Profile.TranslatorVersion(),
		Binding: input.Profile.Binding(), Fields: fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Profile = alias
	if _, err := NewCLISource(input); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("out-of-registry-order profile alias refusal = %v", err)
	}
}

func TestPortableSourceReconstructsExactCLIAndHTTPAuthorities(t *testing.T) {
	cliSource := testCLISource(t, false)
	assertSourceRoundTrip(t, cliSource, domain.AdapterCLI, "fixture/subject.mjs", CLIStartProfileV1)
	if cliSource.StimulusKind() != "CLIStimulus" || len(cliSource.ExecutionBindingCanonicalBytes()) == 0 {
		t.Fatal("CLI source lost stimulus or execution-binding bytes")
	}

	httpSource := testHTTPSource(t, false)
	assertSourceRoundTrip(t, httpSource, domain.AdapterHTTP, "fixture/server.mjs", counterhttp.HTTPPortableStartAuthorityV1)
	if httpSource.StimulusKind() != "HTTPStimulus" || httpSource.StartProfile() != counterhttp.HTTPPortableStartAuthorityV1 {
		t.Fatal("HTTP source lost its child-bind start authority")
	}

	for _, source := range []PortableSource{cliSource, httpSource} {
		wire := string(source.CanonicalBytes())
		for _, forbidden := range []string{
			"projection_proof", "allowed_tuple", "disallowed_tuple", "candidate_execution_key", "producer_metadata",
		} {
			if strings.Contains(wire, forbidden) {
				t.Fatalf("portable source leaked forbidden %q authority", forbidden)
			}
		}
	}
}

func TestPortableSourceRequiresEveryExactPayloadByte(t *testing.T) {
	cliSource := testCLISource(t, false)
	var cliWire sourceIdentity
	if err := json.Unmarshal(cliSource.CanonicalBytes(), &cliWire); err != nil {
		t.Fatal(err)
	}
	cliWire.CLI.Stdin.BytesBase64 = ""
	assertMutatedSourceRefused(t, cliWire)

	var fixtureWire sourceIdentity
	if err := json.Unmarshal(cliSource.CanonicalBytes(), &fixtureWire); err != nil {
		t.Fatal(err)
	}
	fixtureWire.CLI.Fixtures[0].ContentsBase64 = ""
	assertMutatedSourceRefused(t, fixtureWire)

	httpSource := testHTTPSource(t, false)
	var httpWire sourceIdentity
	if err := json.Unmarshal(httpSource.CanonicalBytes(), &httpWire); err != nil {
		t.Fatal(err)
	}
	httpWire.HTTP.Body.BytesBase64 = ""
	assertMutatedSourceRefused(t, httpWire)

	var seedWire sourceIdentity
	if err := json.Unmarshal(httpSource.CanonicalBytes(), &seedWire); err != nil {
		t.Fatal(err)
	}
	seedWire.HTTP.Seeds[0].ContentsBase64 = ""
	assertMutatedSourceRefused(t, seedWire)
}

func TestPortableSourceRequiresExactAuthorityJoin(t *testing.T) {
	source := testCLISource(t, false)
	for _, key := range []string{
		"world_plan_digest", "projection_binding_digest", "portable_profile_digest", "adapter_projection_definition_digest",
		"stimulus_digest", "execution_binding_digest",
	} {
		var wire sourceIdentity
		if err := json.Unmarshal(source.CanonicalBytes(), &wire); err != nil {
			t.Fatal(err)
		}
		switch key {
		case "world_plan_digest":
			wire.PlanDigest = fixedDigest("f").String()
		case "projection_binding_digest":
			wire.ProjectionBindingDigest = fixedDigest("f").String()
		case "portable_profile_digest":
			wire.PortableProfileDigest = fixedDigest("f").String()
		case "adapter_projection_definition_digest":
			wire.AdapterProjectionDigest = fixedDigest("f").String()
		case "stimulus_digest":
			wire.StimulusDigest = fixedDigest("f").String()
		case "execution_binding_digest":
			wire.ExecutionBindingDigest = fixedDigest("f").String()
		}
		assertMutatedSourceRefused(t, wire)
	}

	unknown := bytes.Replace(
		source.CanonicalBytes(), []byte(`"cwd_policy":"MATERIALIZED_ROOT"`),
		[]byte(`"claimed_digest_only_payload":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","cwd_policy":"MATERIALIZED_ROOT"`), 1,
	)
	if _, err := Parse(unknown); err == nil {
		t.Fatal("unknown digest-shaped payload member minted source authority")
	}

	duplicate := bytes.Replace(
		source.CanonicalBytes(), []byte(`"kind":"PortableSource"`),
		[]byte(`"kind":"PortableSource","kind":"PortableSource"`), 1,
	)
	if _, err := Parse(duplicate); !IsCode(err, CodeSourceWireInvalid) {
		t.Fatalf("duplicate JSON member refusal = %v", err)
	}
}

func TestPortableSourceLaunchRosterIsExact(t *testing.T) {
	portableRunner, err := RequiredRunnerDigest(domain.AdapterHTTP)
	if err != nil {
		t.Fatal(err)
	}
	const portableHTTPRunnerDigest = "sha256:dfaeb9bdc715564ba7fa542b731f18c11857250e3cc0d07b63cbc7c472c46026"
	if portableRunner.String() != portableHTTPRunnerDigest {
		t.Fatalf("portable HTTP runner digest = %s, want %s", portableRunner, portableHTTPRunnerDigest)
	}
	input := testHTTPInput(t, false)
	legacyStart, err := counterhttp.NewHTTPStartSpec("fixture/server.mjs")
	if err != nil {
		t.Fatal(err)
	}
	legacyReady, err := counterhttp.NewHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	input.Start = legacyStart
	input.Readiness = legacyReady
	if _, err := NewHTTPSource(input); !IsCode(err, CodeNonportableStartProfile) {
		t.Fatalf("inherited-listener source refusal = %v", err)
	}

	cliSource := testCLISource(t, false)
	var cliWire sourceIdentity
	if err := json.Unmarshal(cliSource.CanonicalBytes(), &cliWire); err != nil {
		t.Fatal(err)
	}
	cliWire.LaunchProfile = "AMBIENT_PATH_LOOKUP_V1"
	assertMutatedSourceRefused(t, cliWire)
}

func TestPortableSourceRejectsNormalizedEnvelopeAliases(t *testing.T) {
	cliSource := testCLISource(t, false)
	var reordered sourceIdentity
	if err := json.Unmarshal(cliSource.CanonicalBytes(), &reordered); err != nil {
		t.Fatal(err)
	}
	reordered.CLI.Environment[0], reordered.CLI.Environment[1] = reordered.CLI.Environment[1], reordered.CLI.Environment[0]
	assertMutatedSourceCode(t, reordered, CodeSourceAuthorityMismatch)

	httpSource := testHTTPSource(t, false)
	var normalizedHeader sourceIdentity
	if err := json.Unmarshal(httpSource.CanonicalBytes(), &normalizedHeader); err != nil {
		t.Fatal(err)
	}
	normalizedHeader.HTTP.Headers[0].Name = "X-Case"
	assertMutatedSourceCode(t, normalizedHeader, CodeSourceAuthorityMismatch)

	var reorderedHTTP sourceIdentity
	if err := json.Unmarshal(httpSource.CanonicalBytes(), &reorderedHTTP); err != nil {
		t.Fatal(err)
	}
	reorderedHTTP.HTTP.Query[0], reorderedHTTP.HTTP.Query[1] = reorderedHTTP.HTTP.Query[1], reorderedHTTP.HTTP.Query[0]
	assertMutatedSourceCode(t, reorderedHTTP, CodeSourceAuthorityMismatch)
	if err := json.Unmarshal(httpSource.CanonicalBytes(), &reorderedHTTP); err != nil {
		t.Fatal(err)
	}
	reorderedHTTP.HTTP.Headers[0], reorderedHTTP.HTTP.Headers[1] = reorderedHTTP.HTTP.Headers[1], reorderedHTTP.HTTP.Headers[0]
	assertMutatedSourceCode(t, reorderedHTTP, CodeSourceAuthorityMismatch)

	nullAlias := bytes.Replace(cliSource.CanonicalBytes(), []byte(`"plan_secret_slots":null`), []byte(`"plan_secret_slots":[]`), 1)
	if bytes.Equal(nullAlias, cliSource.CanonicalBytes()) {
		t.Fatal("test fixture did not contain the expected null empty-slice spelling")
	}
	if _, err := Parse(nullAlias); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("null/empty source alias refusal = %v", err)
	}
}

func TestPortableSourceRequiresExactSerializedHTTPAuthorities(t *testing.T) {
	source := testHTTPSource(t, false)
	for _, mutate := range []func(*sourceIdentity){
		func(wire *sourceIdentity) { wire.HTTP.StartAuthority = counterhttp.HTTPStartAuthorityV1 },
		func(wire *sourceIdentity) { wire.HTTP.ReadinessSignal = counterhttp.ReadinessSignalNameV1 },
		func(wire *sourceIdentity) { wire.HTTP.ReadinessProtocol = counterhttp.ReadinessProtocolV1 },
	} {
		var wire sourceIdentity
		if err := json.Unmarshal(source.CanonicalBytes(), &wire); err != nil {
			t.Fatal(err)
		}
		mutate(&wire)
		assertMutatedSourceCode(t, wire, CodeNonportableStartProfile)
	}
}

func TestPortableSourceWireMemberRostersAreExact(t *testing.T) {
	topLevel := []string{
		"adapter", "adapter_projection_definition_base64", "adapter_projection_definition_digest",
		"cli", "closed_facts", "entrypoint", "execution_binding_base64", "execution_binding_digest",
		"http", "kind", "launch_profile", "limits", "plan_environment", "plan_secret_slots",
		"portable_profile_base64", "portable_profile_digest", "projection_binding_base64", "projection_binding_digest",
		"schema_version", "source_profile", "start_profile", "stimulus_base64", "stimulus_digest", "stimulus_kind",
		"version", "world_plan_base64", "world_plan_digest",
	}
	cliArm := []string{
		"argv", "base_argv", "cwd_policy", "environment", "executable", "fixtures", "stderr_bytes", "stdin", "stdout_bytes",
	}
	httpArm := []string{
		"body", "body_bytes", "header_bytes", "header_count", "method", "ordered_query_multimap",
		"ordered_request_header_multimap", "path", "readiness_protocol", "readiness_signal", "seeds",
		"start_authority", "status_line_bytes",
	}
	assertJSONKeys(t, testCLISource(t, false).CanonicalBytes(), topLevel, "cli", cliArm)
	httpSource := testHTTPSource(t, false)
	assertJSONKeys(t, httpSource.CanonicalBytes(), topLevel, "http", httpArm)
	executionKeys := []string{
		"authority", "capture_policy_digest", "execution_payload_digest", "fixture_recipe_digest",
		"http_adapter_projection_definition_digest", "http_body_capture_bytes", "kind",
		"projection_definition_digest", "readiness_contract_digest", "schema_version", "start_spec_digest",
		"stimulus_digest", "world_plan_digest",
	}
	var execution map[string]json.RawMessage
	if err := json.Unmarshal(httpSource.ExecutionBindingCanonicalBytes(), &execution); err != nil {
		t.Fatal(err)
	}
	assertMapKeys(t, execution, executionKeys, "portable HTTP execution binding")
	var authority string
	if err := json.Unmarshal(execution["authority"], &authority); err != nil || authority != counterhttp.HTTPPortableExecutionAuthorityV1 {
		t.Fatalf("portable execution binding authority = %q %v", authority, err)
	}
}

func TestPortableSourceRequiresExactRunnerToolSecretsAndEntrypoint(t *testing.T) {
	input := testCLIInput(t, false)
	nearRunner := rebuildPlan(t, input.Plan, func(config *domain.WorldPlanConfig) {
		config.Adapter.RunnerDigest = fixedDigest("f")
	})
	input.Plan = nearRunner
	if _, err := NewCLISource(input); !IsCode(err, CodeNonportableStartProfile) {
		t.Fatalf("near runner lineage refusal = %v", err)
	}

	input = testCLIInput(t, false)
	input.Plan = rebuildPlan(t, input.Plan, func(config *domain.WorldPlanConfig) {
		config.RequiredTools = []domain.RequiredTool{{Name: "node", VersionConstraint: "target-admitted"}}
	})
	if _, err := NewCLISource(input); !IsCode(err, CodeNonportableStartProfile) {
		t.Fatalf("nonphysical tool constraint refusal = %v", err)
	}

	input = testCLIInput(t, false)
	input.Plan = rebuildPlan(t, input.Plan, func(config *domain.WorldPlanConfig) {
		config.SecretSlots = []domain.SecretSlot{{Name: "API_TOKEN", Presence: domain.SecretRequiredNoCapture}}
	})
	if _, err := NewCLISource(input); !IsCode(err, CodeSourceAuthorityMismatch) {
		t.Fatalf("plan secret-slot refusal = %v", err)
	}

	for _, entrypoint := range []string{"fixture/.subject.mjs", "fixture/sübject.mjs"} {
		invalid := testCLIInputWithEntrypoint(t, entrypoint)
		if _, err := NewCLISource(invalid); !IsCode(err, CodeNonportableStartProfile) {
			t.Fatalf("entrypoint %q refusal = %v", entrypoint, err)
		}
	}
	for _, entrypoint := range []string{
		"/fixture/subject.mjs", "../subject.mjs", "fixture/../subject.mjs", `fixture\subject.mjs`,
		"--eval.mjs", "fixture/subject.py",
	} {
		input := testCLIInput(t, false)
		stimulus := input.Stimulus
		candidate, err := cli.NewCLIStimulus(cli.CLIStimulusConfig{
			Executable: stimulus.Executable(), BaseArgv: []string{entrypoint}, Argv: stimulus.Argv(), Stdin: stimulus.Stdin(),
			Environment: stimulus.Environment(), Fixtures: stimulus.Fixtures(), CWDPolicy: stimulus.CWDPolicy(),
		})
		if err == nil {
			config := planConfig(input.Plan)
			config.StartArgv = []string{"node", entrypoint}
			candidatePlan, planErr := domain.NewWorldPlan(config)
			if planErr == nil {
				input.Plan, input.Stimulus = candidatePlan, candidate
				if _, sourceErr := NewCLISource(input); sourceErr == nil {
					t.Fatalf("all source gates admitted invalid entrypoint %q", entrypoint)
				}
			}
		}
	}
}

func TestPortableSourceParseRechecksSelfConsistentNodeLineage(t *testing.T) {
	wrongRunner := testCLIInput(t, false)
	wrongRunner.Plan = rebuildPlan(t, wrongRunner.Plan, func(config *domain.WorldPlanConfig) {
		config.Adapter.RunnerDigest = fixedDigest("f")
	})
	wrongTool := testCLIInput(t, false)
	wrongTool.Plan = rebuildPlan(t, wrongTool.Plan, func(config *domain.WorldPlanConfig) {
		config.RequiredTools = []domain.RequiredTool{{Name: "node", VersionConstraint: "target-admitted"}}
	})
	wrongEntrypoint := testCLIInputWithEntrypoint(t, "fixture/.subject.mjs")
	for _, input := range []CLIInput{wrongRunner, wrongTool, wrongEntrypoint} {
		if _, err := Parse(selfConsistentCLIIdentityBytes(t, input)); !IsCode(err, CodeNonportableStartProfile) {
			t.Fatalf("self-consistent nonportable CLI source refusal = %v", err)
		}
	}

	httpInput := testHTTPInput(t, false)
	httpInput.Plan = rebuildPlan(t, httpInput.Plan, func(config *domain.WorldPlanConfig) {
		config.Adapter.RunnerDigest = fixedDigest("e")
	})
	if _, err := Parse(selfConsistentHTTPIdentityBytes(t, httpInput)); !IsCode(err, CodeNonportableStartProfile) {
		t.Fatalf("self-consistent nonportable HTTP source refusal = %v", err)
	}
}

func TestPortableSourceCeilingIsExactAndDefensive(t *testing.T) {
	const expectedMaxSourceCanonicalBytes = 384 << 10
	if MaxSourceCanonicalBytes != expectedMaxSourceCanonicalBytes {
		t.Fatalf("portable source ceiling = %d, want %d", MaxSourceCanonicalBytes, expectedMaxSourceCanonicalBytes)
	}
	var exactOversize sourceIdentity
	if err := json.Unmarshal(testCLISource(t, false).CanonicalBytes(), &exactOversize); err != nil {
		t.Fatal(err)
	}
	exactOversize.CLI.Fixtures[0].ContentsBase64 = ""
	base, err := canon.CanonicalizeTyped(exactOversize)
	if err != nil {
		t.Fatal(err)
	}
	padding := MaxSourceCanonicalBytes + 1 - len(base)
	if padding <= 0 {
		t.Fatalf("source fixture cannot construct exact oversized boundary from %d bytes", len(base))
	}
	exactOversize.CLI.Fixtures[0].ContentsBase64 = strings.Repeat("A", padding)
	exact, err := canon.CanonicalizeTyped(exactOversize)
	if err != nil || len(exact) != MaxSourceCanonicalBytes+1 {
		t.Fatalf("exact oversized source length = %d, want %d: %v", len(exact), MaxSourceCanonicalBytes+1, err)
	}
	if _, err := buildSource(exactOversize); !IsCode(err, CodeSourceLimitExceeded) {
		t.Fatalf("exact +1 canonical source refusal = %v", err)
	}

	low, high := 0, MaxSourceCanonicalBytes
	var boundary PortableSource
	for low <= high {
		middle := low + (high-low)/2
		source, err := NewCLISource(testCLIInputWithFixtureBytes(t, middle))
		if err == nil {
			boundary = source
			low = middle + 1
			continue
		}
		if !IsCode(err, CodeSourceLimitExceeded) {
			t.Fatalf("source boundary search failed at %d bytes: %v", middle, err)
		}
		high = middle - 1
	}
	if !boundary.Valid() || MaxSourceCanonicalBytes-len(boundary.CanonicalBytes()) > 4 {
		t.Fatalf("accepted boundary was not near the exact ceiling: %d of %d", len(boundary.CanonicalBytes()), MaxSourceCanonicalBytes)
	}
	if _, err := NewCLISource(testCLIInputWithFixtureBytes(t, high+1)); !IsCode(err, CodeSourceLimitExceeded) {
		t.Fatalf("first oversized source refusal = %v", err)
	}
	if _, err := Parse(bytes.Repeat([]byte("x"), MaxSourceCanonicalBytes+1)); !IsCode(err, CodePortableSourceRequired) {
		t.Fatalf("oversized parse refusal = %v", err)
	}

	canonical := boundary.CanonicalBytes()
	canonical[0] ^= 0xff
	stimulus := boundary.StimulusCanonicalBytes()
	stimulus[0] ^= 0xff
	execution := boundary.ExecutionBindingCanonicalBytes()
	execution[0] ^= 0xff
	if !boundary.Valid() {
		t.Fatal("portable source exposed mutable identity bytes")
	}
}

func TestPortableSourcePreservesAbsentVersusPresentEmpty(t *testing.T) {
	absentCLI := testCLISource(t, true)
	presentCLI := testCLISourcePresentEmpty(t)
	if absentCLI.Digest() == presentCLI.Digest() || bytes.Equal(absentCLI.CanonicalBytes(), presentCLI.CanonicalBytes()) {
		t.Fatal("CLI absent stdin collapsed into present-empty")
	}
	absentHTTP := testHTTPSource(t, true)
	presentHTTP := testHTTPSourcePresentEmpty(t)
	if absentHTTP.Digest() == presentHTTP.Digest() || bytes.Equal(absentHTTP.CanonicalBytes(), presentHTTP.CanonicalBytes()) {
		t.Fatal("HTTP absent body collapsed into present-empty")
	}
}

func TestPortableSourceViewsRecoverExactRawPayloadsWithoutPrivateJSONParsing(t *testing.T) {
	cliSource := testCLISource(t, false)
	cliView, ok := cliSource.CLIView()
	if !ok || !cliView.Valid() || cliView.SourceDigest() != cliSource.Digest() {
		t.Fatal("CLI source did not issue its sealed typed view")
	}
	cliStimulus := cliView.Stimulus()
	if !bytes.Equal(cliStimulus.Stdin().Bytes(), []byte("contract-input")) ||
		len(cliStimulus.Fixtures()) != 1 ||
		!bytes.Equal(cliStimulus.Fixtures()[0].Contents(), []byte(`{"mode":"fixture"}`)) ||
		cliView.Capture().StdoutBytes() != 32<<10 || !cliView.Projection().Valid() ||
		cliView.Projection().Binding().Digest() != cliSource.ProjectionBinding().Digest() {
		t.Fatal("CLI typed view lost byte-complete source payloads")
	}
	fixtureCopy := cliStimulus.Fixtures()[0].Contents()
	fixtureCopy[0] ^= 0xff
	if !cliView.Valid() || !cliSource.Valid() {
		t.Fatal("CLI source view exposed mutable fixture bytes")
	}
	if _, present := cliSource.HTTPView(); present {
		t.Fatal("CLI source issued an HTTP view")
	}

	httpSource := testHTTPSource(t, false)
	httpView, ok := httpSource.HTTPView()
	if !ok || !httpView.Valid() || httpView.SourceDigest() != httpSource.Digest() ||
		httpView.Start().Authority() != counterhttp.HTTPPortableStartAuthorityV1 ||
		httpView.Readiness().Protocol() != counterhttp.PortableReadinessProtocolV1 || !httpView.Projection().Valid() ||
		httpView.Projection().Binding().Digest() != httpSource.ProjectionBinding().Digest() {
		t.Fatal("HTTP source did not issue its sealed typed view")
	}
	httpStimulus := httpView.Stimulus()
	if !bytes.Equal(httpStimulus.Body().Bytes(), []byte("request-body")) || len(httpStimulus.Seeds()) != 1 ||
		!bytes.Equal(httpStimulus.Seeds()[0].Contents(), []byte(`{"seed":true}`)) ||
		len(httpStimulus.Query()) != 2 || httpStimulus.Query()[0].Name() != "mode" || httpStimulus.Query()[1].Name() != "mode" ||
		len(httpStimulus.Headers()) != 2 || httpStimulus.Headers()[0].Name() != "x-case" ||
		httpStimulus.Headers()[0].Value() != "first" || httpStimulus.Headers()[1].Value() != "second" {
		t.Fatal("HTTP typed view lost byte-complete ordered source payloads")
	}
	bodyCopy := httpStimulus.Body().Bytes()
	bodyCopy[0] ^= 0xff
	seedCopy := httpStimulus.Seeds()[0].Contents()
	seedCopy[0] ^= 0xff
	if !httpView.Valid() || !httpSource.Valid() {
		t.Fatal("HTTP source view exposed mutable body or seed bytes")
	}
	if _, present := httpSource.CLIView(); present {
		t.Fatal("HTTP source issued a CLI view")
	}
}

func FuzzPortableSourceParseNeverMintsUnreconstructedAuthority(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"schema_version":"countershape/v1","kind":"PortableSource"}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		source, err := Parse(input)
		if err == nil && !source.Valid() {
			t.Fatal("parser returned an invalid source capability")
		}
	})
}

func assertSourceRoundTrip(t *testing.T, source PortableSource, adapter domain.AdapterDomain, entrypoint, start string) {
	t.Helper()
	if !source.Valid() || source.Adapter() != adapter || source.Entrypoint() != entrypoint || source.StartProfile() != start {
		t.Fatal("portable source capability facts disagree")
	}
	reopened, err := Parse(source.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Valid() || reopened.Digest() != source.Digest() ||
		!bytes.Equal(reopened.CanonicalBytes(), source.CanonicalBytes()) ||
		!bytes.Equal(reopened.StimulusCanonicalBytes(), source.StimulusCanonicalBytes()) ||
		reopened.ExecutionBindingDigest() != source.ExecutionBindingDigest() ||
		reopened.Plan().Digest() != source.Plan().Digest() ||
		reopened.ProjectionBinding().Digest() != source.ProjectionBinding().Digest() ||
		reopened.Profile().Digest() != source.Profile().Digest() {
		t.Fatal("portable source did not reopen byte-exactly")
	}
}

func assertMutatedSourceRefused(t *testing.T, wire any) {
	t.Helper()
	exact, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(exact); err == nil {
		t.Fatal("mutated source minted authority")
	}
}

func assertMutatedSourceCode(t *testing.T, wire any, code Code) {
	t.Helper()
	exact, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(exact); !IsCode(err, code) {
		t.Fatalf("mutated source refusal = %v, want %s", err, code)
	}
}

func assertJSONKeys(t *testing.T, exact []byte, topLevel []string, arm string, armKeys []string) {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(exact, &object); err != nil {
		t.Fatal(err)
	}
	assertMapKeys(t, object, topLevel, "top-level source")
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(object[arm], &nested); err != nil {
		t.Fatal(err)
	}
	assertMapKeys(t, nested, armKeys, arm+" source")
}

func assertMapKeys(t *testing.T, object map[string]json.RawMessage, expected []string, label string) {
	t.Helper()
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	want := slices.Clone(expected)
	slices.Sort(want)
	if !slices.Equal(keys, want) {
		t.Fatalf("%s roster = %q, want %q", label, keys, want)
	}
}

func selfConsistentCLIIdentityBytes(t *testing.T, input CLIInput) []byte {
	t.Helper()
	execution, err := climodel.BindExecution(input.Plan, input.Stimulus, input.Capture, input.Projection)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.CanonicalizeTyped(cliSourceIdentity(
		input.Plan, input.Profile, input.Projection, input.Stimulus, input.Capture, execution,
	))
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func selfConsistentHTTPIdentityBytes(t *testing.T, input HTTPInput) []byte {
	t.Helper()
	execution, err := httpmodel.BindExecution(
		input.Plan, input.Stimulus, input.Start, input.Capture, input.Readiness, input.Projection,
	)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.CanonicalizeTyped(httpSourceIdentity(
		input.Plan, input.Profile, input.Projection, input.Stimulus, input.Start, input.Capture, input.Readiness, execution,
	))
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func testCLISource(t *testing.T, absentStdin bool) PortableSource {
	t.Helper()
	source, err := NewCLISource(testCLIInput(t, absentStdin))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func testCLIInput(t *testing.T, absentStdin bool) CLIInput {
	t.Helper()
	projection, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldStdoutBytes,
	}})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := cli.NewCLICapturePolicy(cli.CLICapturePolicyConfig{StdoutBytes: 32 << 10, StderrBytes: 16 << 10})
	if err != nil {
		t.Fatal(err)
	}
	fixtureRecipe, err := cli.NewCLIFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	mode, err := cli.PresentEnvironment("APP_MODE", "public-test-input")
	if err != nil {
		t.Fatal(err)
	}
	absent, err := cli.AbsentEnvironment("OPTIONAL_FLAG")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := cli.NewFixtureFile("fixture.json", []byte(`{"mode":"fixture"}`), cli.FixtureMode0644)
	if err != nil {
		t.Fatal(err)
	}
	stdin := cli.AbsentStdin()
	if !absentStdin {
		stdin, err = cli.PresentStdin([]byte("contract-input"))
		if err != nil {
			t.Fatal(err)
		}
	}
	stimulus, err := cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture/subject.mjs"}, Argv: []string{"--mode", "argv"},
		Stdin: stdin, Environment: []cli.CLIEnvironmentBinding{mode, absent},
		Fixtures: []cli.CLIFixtureFile{fixture}, CWDPolicy: cli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := testPlan(t, domain.AdapterCLI, domain.OneCLIInvocation, []string{"node", "fixture/subject.mjs"},
		domain.Readiness{Kind: domain.ReadinessNone}, capture.Digest(), fixtureRecipe.Digest(), projection.Binding(),
		domain.Budgets{StdoutBytes: capture.StdoutBytes(), StderrBytes: capture.StderrBytes(), HTTPBodyBytes: 64 << 10})
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		t.Fatal(err)
	}
	authority, err := climodel.ResolveCLIProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return CLIInput{
		Plan: plan, Stimulus: stimulus, Capture: capture, Profile: resolved.Profile(), Projection: authority,
	}
}

func testCLIInputWithEntrypoint(t *testing.T, entrypoint string) CLIInput {
	t.Helper()
	input := testCLIInput(t, false)
	stimulus := input.Stimulus
	var err error
	input.Stimulus, err = cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: stimulus.Executable(), BaseArgv: []string{entrypoint}, Argv: stimulus.Argv(), Stdin: stimulus.Stdin(),
		Environment: stimulus.Environment(), Fixtures: stimulus.Fixtures(), CWDPolicy: stimulus.CWDPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Plan = rebuildPlan(t, input.Plan, func(config *domain.WorldPlanConfig) {
		config.StartArgv = []string{"node", entrypoint}
	})
	return input
}

func testCLIInputWithFixtureBytes(t *testing.T, count int) CLIInput {
	t.Helper()
	input := testCLIInput(t, false)
	fixture, err := cli.NewFixtureFile("fixture.json", bytes.Repeat([]byte{'x'}, count), cli.FixtureMode0644)
	if err != nil {
		t.Fatal(err)
	}
	stimulus := input.Stimulus
	input.Stimulus, err = cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: stimulus.Executable(), BaseArgv: stimulus.BaseArgv(), Argv: stimulus.Argv(), Stdin: stimulus.Stdin(),
		Environment: stimulus.Environment(), Fixtures: []cli.CLIFixtureFile{fixture}, CWDPolicy: stimulus.CWDPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func rebuildPlan(t *testing.T, plan domain.WorldPlan, mutate func(*domain.WorldPlanConfig)) domain.WorldPlan {
	t.Helper()
	config := planConfig(plan)
	mutate(&config)
	rebuilt, err := domain.NewWorldPlan(config)
	if err != nil {
		t.Fatal(err)
	}
	return rebuilt
}

func planConfig(plan domain.WorldPlan) domain.WorldPlanConfig {
	return domain.WorldPlanConfig{
		CandidateSetDigest: plan.CandidateSetDigest(), MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
		ComparisonEnvelopeDigest: plan.ComparisonEnvelopeDigest(), Adapter: plan.Adapter(),
		ExecutionShape: plan.ExecutionShape(), StartArgv: plan.StartArgv(), SetupArgv: plan.SetupArgv(),
		Environment: plan.Environment(), SecretSlots: plan.SecretSlots(), FixtureRecipeDigest: plan.FixtureRecipeDigest(),
		Readiness: plan.Readiness(), CapturePolicyDigest: plan.CapturePolicyDigest(),
		ProjectionDefinition: plan.ProjectionDefinitionBinding(), RepeatSchedule: plan.RepeatSchedule(),
		RequiredTools: plan.RequiredTools(), Budgets: plan.Budgets(),
	}
}

func testCLISourcePresentEmpty(t *testing.T) PortableSource {
	t.Helper()
	input := testCLIInput(t, true)
	empty, err := cli.PresentStdin([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	stimulus := input.Stimulus
	input.Stimulus, err = cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: stimulus.Executable(), BaseArgv: stimulus.BaseArgv(), Argv: stimulus.Argv(), Stdin: empty,
		Environment: stimulus.Environment(), Fixtures: stimulus.Fixtures(), CWDPolicy: stimulus.CWDPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewCLISource(input)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func testHTTPSource(t *testing.T, absentBody bool) PortableSource {
	t.Helper()
	source, err := NewHTTPSource(testHTTPInput(t, absentBody))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func testHTTPInput(t *testing.T, absentBody bool) HTTPInput {
	t.Helper()
	projection, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	capture, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 16 << 10, HeaderCount: 32, BodyBytes: 64 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	start, err := counterhttp.NewPortableHTTPStartSpec("fixture/server.mjs")
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := counterhttp.NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	fixtureRecipe, err := counterhttp.NewHTTPFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	flag, err := counterhttp.QueryFlag("mode")
	if err != nil {
		t.Fatal(err)
	}
	value, err := counterhttp.QueryValue("mode", "exact value")
	if err != nil {
		t.Fatal(err)
	}
	header, err := counterhttp.NewRequestHeader("x-case", "first")
	if err != nil {
		t.Fatal(err)
	}
	headerAgain, err := counterhttp.NewRequestHeader("X-Case", "second")
	if err != nil {
		t.Fatal(err)
	}
	seed, err := counterhttp.NewSeedFile("seed.json", []byte(`{"seed":true}`), counterhttp.SeedMode0644)
	if err != nil {
		t.Fatal(err)
	}
	body := counterhttp.AbsentBody()
	if !absentBody {
		body, err = counterhttp.PresentBody([]byte("request-body"))
		if err != nil {
			t.Fatal(err)
		}
	}
	stimulus, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: counterhttp.MethodPOST, Path: "/contract", Query: []counterhttp.HTTPQueryEntry{flag, value},
		Headers: []counterhttp.HTTPRequestHeader{header, headerAgain}, Body: body, Seeds: []counterhttp.HTTPSeedFile{seed},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := testPlan(t, domain.AdapterHTTP, domain.OneLoopbackHTTPRequest, []string{"node", "fixture/server.mjs"},
		domain.Readiness{Kind: domain.FixtureOwnedReadiness, SignalName: counterhttp.PortableReadinessSignalNameV1},
		capture.Digest(), fixtureRecipe.Digest(), projection.Binding(),
		domain.Budgets{StdoutBytes: 32 << 10, StderrBytes: 16 << 10, HTTPBodyBytes: capture.BodyBytes(), ReadinessMS: 1500})
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		t.Fatal(err)
	}
	authority, err := httpmodel.ResolveHTTPProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return HTTPInput{
		Plan: plan, Stimulus: stimulus, Start: start, Capture: capture, Readiness: readiness,
		Profile: resolved.Profile(), Projection: authority,
	}
}

func testHTTPSourcePresentEmpty(t *testing.T) PortableSource {
	t.Helper()
	input := testHTTPInput(t, true)
	empty, err := counterhttp.PresentBody([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	stimulus := input.Stimulus
	input.Stimulus, err = counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: stimulus.Method(), Path: stimulus.Path(), Query: stimulus.Query(), Headers: stimulus.Headers(),
		Body: empty, Seeds: stimulus.Seeds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewHTTPSource(input)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func testPlan(
	t *testing.T,
	adapter domain.AdapterDomain,
	shape domain.ExecutionShape,
	start []string,
	readiness domain.Readiness,
	capture, fixture domain.Digest,
	projection domain.ProjectionDefinitionBinding,
	overrides domain.Budgets,
) domain.WorldPlan {
	t.Helper()
	budgets := domain.Budgets{
		CandidateCount: 2, MaterializedEntryCount: 32, MaterializedBytesPerWorld: 2 << 20,
		SingleBlobBytes: 1 << 20, ReadinessMS: overrides.ReadinessMS, ProbeMS: 1000, TeardownMS: 800,
		StdoutBytes: overrides.StdoutBytes, StderrBytes: overrides.StderrBytes, HTTPBodyBytes: overrides.HTTPBodyBytes,
		ProposedShrinkStimuli: 4, TotalCandidateTrials: 32, ShrinkWallMS: 30_000,
	}
	runnerDigest, err := RequiredRunnerDigest(adapter)
	if err != nil {
		t.Fatal(err)
	}
	adapterVersion := CLIAdapterVersionV1
	if adapter == domain.AdapterHTTP {
		adapterVersion = HTTPAdapterVersionV1
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: fixedDigest("1"), MaterializationPolicyDigest: fixedDigest("2"),
		ComparisonEnvelopeDigest: fixedDigest("3"),
		Adapter:                  domain.Adapter{Domain: adapter, AdapterVersion: adapterVersion, RunnerDigest: runnerDigest},
		ExecutionShape:           shape, StartArgv: start, SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}, {Name: "NO_COLOR", Value: "1"}, {Name: "TZ", Value: "UTC"}},
		SecretSlots: []domain.SecretSlot{}, FixtureRecipeDigest: fixture, Readiness: readiness,
		CapturePolicyDigest: capture, ProjectionDefinition: projection,
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 2, ConfirmationRepeats: 2, Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools:  []domain.RequiredTool{{Name: "node", VersionConstraint: NodeToolConstraintV1}}, Budgets: budgets,
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func fixedDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}
