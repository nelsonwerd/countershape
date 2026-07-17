// Package contracts provides deterministic public-constructor fixtures for
// contract compiler and runtime tests. It is test support, never an alternate
// authority constructor.
package contracts

import (
	"strings"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

// CLISource returns one exact dependency-free logical-Node CLI source through
// the same public constructors used by production authority.
func CLISource() (contractsource.PortableSource, error) {
	stdin, err := cli.PresentStdin([]byte("contract-input"))
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	return CLISourceWithStdin(stdin)
}

// CLISourceWithStdin preserves the exact public fixture while allowing tests
// to exercise ABSENT versus PRESENT-empty stdin without alternate authority.
func CLISourceWithStdin(stdin cli.CLIStdin) (contractsource.PortableSource, error) {
	return cliSourceWithStdinAndEnvironment(stdin, "public-contract-test")
}

// CLISourceWithEnvironmentValue preserves the public constructor path while
// letting recovery tests prove that authorized opaque source text is data, not
// a policy-scanning surface.
func CLISourceWithEnvironmentValue(value string) (contractsource.PortableSource, error) {
	stdin, err := cli.PresentStdin([]byte("contract-input"))
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	return cliSourceWithStdinAndEnvironment(stdin, value)
}

func cliSourceWithStdinAndEnvironment(stdin cli.CLIStdin, environmentValue string) (contractsource.PortableSource, error) {
	projection, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldStdoutBytes,
	}})
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	capture, err := cli.NewCLICapturePolicy(cli.CLICapturePolicyConfig{
		StdoutBytes: 32 << 10,
		StderrBytes: 16 << 10,
	})
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	fixtureRecipe, err := cli.NewCLIFixtureRecipe()
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	mode, err := cli.PresentEnvironment("APP_MODE", environmentValue)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	absent, err := cli.AbsentEnvironment("OPTIONAL_FLAG")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	fixture, err := cli.NewFixtureFile("fixture.json", []byte(`{"mode":"fixture"}`), cli.FixtureMode0644)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	stimulus, err := cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: "node",
		BaseArgv:   []string{"fixture/subject.mjs"},
		Argv:       []string{"--mode", "argv"},
		Stdin:      stdin,
		Environment: []cli.CLIEnvironmentBinding{
			mode,
			absent,
		},
		Fixtures:  []cli.CLIFixtureFile{fixture},
		CWDPolicy: cli.CWDMaterializedRoot,
	})
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	plan, err := cliPlan(capture, fixtureRecipe, projection.Binding())
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	authority, err := climodel.ResolveCLIProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	return contractsource.NewCLISource(contractsource.CLIInput{
		Plan: plan, Stimulus: stimulus, Capture: capture,
		Profile: resolved.Profile(), Projection: authority,
	})
}

// HTTPSource returns one exact dependency-free portable-start HTTP source
// through the same public constructors used by production authority.
func HTTPSource() (contractsource.PortableSource, error) {
	return HTTPSourceWithBody(counterhttp.AbsentBody())
}

// HTTPSourceWithBody preserves the exact public fixture while allowing tests
// to distinguish ABSENT from PRESENT-empty request bodies.
func HTTPSourceWithBody(body counterhttp.HTTPBody) (contractsource.PortableSource, error) {
	projection, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	capture, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 16 << 10, HeaderCount: 32, BodyBytes: 64 << 10,
	})
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	start, err := counterhttp.NewPortableHTTPStartSpec("fixture/server.mjs")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	readiness, err := counterhttp.NewPortableHTTPReadinessContract()
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	fixtureRecipe, err := counterhttp.NewHTTPFixtureRecipe()
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	flag, err := counterhttp.QueryFlag("contract")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	value, err := counterhttp.QueryValue("mode", "exact")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	header, err := counterhttp.NewRequestHeader("x-countershape-contract", "v1")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	stimulus, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: counterhttp.MethodGET, Path: "/contract",
		Query:   []counterhttp.HTTPQueryEntry{flag, value},
		Headers: []counterhttp.HTTPRequestHeader{header}, Body: body,
	})
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	plan, err := httpPlan(capture, fixtureRecipe, projection.Binding(), readiness)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	authority, err := httpmodel.ResolveHTTPProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	return contractsource.NewHTTPSource(contractsource.HTTPInput{
		Plan: plan, Stimulus: stimulus, Start: start, Capture: capture, Readiness: readiness,
		Profile: resolved.Profile(), Projection: authority,
	})
}

func cliPlan(
	capture cli.CLICapturePolicy,
	fixture cli.CLIFixtureRecipe,
	projection domain.ProjectionDefinitionBinding,
) (domain.WorldPlan, error) {
	runner, err := contractsource.RequiredRunnerDigest(domain.AdapterCLI)
	if err != nil {
		return domain.WorldPlan{}, err
	}
	return domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          fixedDigest("1"),
		MaterializationPolicyDigest: fixedDigest("2"),
		ComparisonEnvelopeDigest:    fixedDigest("3"),
		Adapter: domain.Adapter{
			Domain: domain.AdapterCLI, AdapterVersion: contractsource.CLIAdapterVersionV1, RunnerDigest: runner,
		},
		ExecutionShape: domain.OneCLIInvocation,
		StartArgv:      []string{"node", "fixture/subject.mjs"},
		SetupArgv:      []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"},
			{Name: "NO_COLOR", Value: "1"},
			{Name: "TZ", Value: "UTC"},
		},
		SecretSlots:          []domain.SecretSlot{},
		FixtureRecipeDigest:  fixture.Digest(),
		Readiness:            domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest:  capture.Digest(),
		ProjectionDefinition: projection,
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: contractsource.NodeToolConstraintV1}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 32, MaterializedBytesPerWorld: 2 << 20,
			SingleBlobBytes: 1 << 20, ProbeMS: 1000, TeardownMS: 800,
			StdoutBytes: capture.StdoutBytes(), StderrBytes: capture.StderrBytes(), HTTPBodyBytes: 64 << 10,
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 32, ShrinkWallMS: 30_000,
		},
	})
}

func httpPlan(
	capture counterhttp.HTTPCapturePolicy,
	fixture counterhttp.HTTPFixtureRecipe,
	projection domain.ProjectionDefinitionBinding,
	readiness counterhttp.HTTPReadinessContract,
) (domain.WorldPlan, error) {
	runner, err := contractsource.RequiredRunnerDigest(domain.AdapterHTTP)
	if err != nil {
		return domain.WorldPlan{}, err
	}
	return domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          fixedDigest("4"),
		MaterializationPolicyDigest: fixedDigest("5"),
		ComparisonEnvelopeDigest:    fixedDigest("6"),
		Adapter: domain.Adapter{
			Domain: domain.AdapterHTTP, AdapterVersion: contractsource.HTTPAdapterVersionV1, RunnerDigest: runner,
		},
		ExecutionShape: domain.OneLoopbackHTTPRequest,
		StartArgv:      []string{"node", "fixture/server.mjs"},
		SetupArgv:      []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"},
			{Name: "NO_COLOR", Value: "1"},
			{Name: "TZ", Value: "UTC"},
		},
		SecretSlots:         []domain.SecretSlot{},
		FixtureRecipeDigest: fixture.Digest(),
		Readiness: domain.Readiness{
			Kind: domain.FixtureOwnedReadiness, SignalName: readiness.SignalName(),
		},
		CapturePolicyDigest:  capture.Digest(),
		ProjectionDefinition: projection,
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: contractsource.NodeToolConstraintV1}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 32, MaterializedBytesPerWorld: 2 << 20,
			SingleBlobBytes: 1 << 20, ReadinessMS: 1500, ProbeMS: 1000, TeardownMS: 800,
			StdoutBytes: 32 << 10, StderrBytes: 16 << 10, HTTPBodyBytes: capture.BodyBytes(),
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 32, ShrinkWallMS: 30_000,
		},
	})
}

func fixedDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}
