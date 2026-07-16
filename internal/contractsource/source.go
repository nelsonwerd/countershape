package contractsource

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"slices"

	cli "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
)

const (
	SourceVersionV1             = "portable-source/v1"
	SourceProfileV1             = "countershape-node-core-exact/v1"
	LaunchProfileV1             = "NODE_REPO_SCRIPT_V1"
	CLIStartProfileV1           = "DIRECT_CHILD_V1"
	NodeToolConstraintV1        = runnerprofile.NodeToolConstraintV1
	CLIAdapterVersionV1         = runnerprofile.CLIAdapterVersionV1
	HTTPAdapterVersionV1        = runnerprofile.HTTPAdapterVersionV1
	CLIRunnerLineageV1          = runnerprofile.CLIRunnerLineageV1
	HTTPPortableRunnerLineageV1 = runnerprofile.HTTPPortableRunnerLineageV1
	PrivateRootPolicyV1         = "NEW_PRIVATE_ROOT_COPY_VERIFIED_SOURCE_V1"
	EnvironmentProfileV1        = "EXPLICIT_PLAN_PLUS_STIMULUS_NO_AMBIENT_INHERITANCE_V1"
	MaxSourceCanonicalBytes     = 384 << 10
	maxSourcePayloadBytes       = 16 << 20
)

type CLIInput struct {
	Plan       domain.WorldPlan
	Stimulus   cli.CLIStimulus
	Capture    cli.CLICapturePolicy
	Profile    projectionprofile.Profile
	Projection cli.CLIProjectionAuthority
}

type HTTPInput struct {
	Plan       domain.WorldPlan
	Stimulus   counterhttp.HTTPStimulus
	Start      counterhttp.HTTPStartSpec
	Capture    counterhttp.HTTPCapturePolicy
	Readiness  counterhttp.HTTPReadinessContract
	Profile    projectionprofile.Profile
	Projection counterhttp.HTTPProjectionAuthority
}

type PortableSource struct {
	digest           domain.Digest
	canonical        []byte
	plan             domain.WorldPlan
	binding          domain.ProjectionDefinitionBinding
	profile          projectionprofile.Profile
	stimulusKind     string
	stimulusBytes    []byte
	stimulusDigest   domain.Digest
	executionBinding domain.Digest
	executionBytes   []byte
	adapter          domain.AdapterDomain
	entrypoint       string
	startProfile     string
	authorities      reconstructedAuthorities
	seal             *sourceSeal
}

type reconstructedAuthorities struct {
	cliStimulus    cli.CLIStimulus
	cliCapture     cli.CLICapturePolicy
	cliProjection  cli.CLIProjectionAuthority
	httpStimulus   counterhttp.HTTPStimulus
	httpStart      counterhttp.HTTPStartSpec
	httpCapture    counterhttp.HTTPCapturePolicy
	httpReadiness  counterhttp.HTTPReadinessContract
	httpProjection counterhttp.HTTPProjectionAuthority
}

type CLISourceView struct {
	sourceDigest domain.Digest
	stimulus     cli.CLIStimulus
	capture      cli.CLICapturePolicy
	projection   cli.CLIProjectionAuthority
	seal         *sourceSeal
}

type HTTPSourceView struct {
	sourceDigest domain.Digest
	stimulus     counterhttp.HTTPStimulus
	start        counterhttp.HTTPStartSpec
	capture      counterhttp.HTTPCapturePolicy
	readiness    counterhttp.HTTPReadinessContract
	projection   counterhttp.HTTPProjectionAuthority
	seal         *sourceSeal
}

type sourceSeal struct{}

var portableSourceAuthority = &sourceSeal{}

type stdinIdentity struct {
	Presence    string `json:"presence"`
	BytesBase64 string `json:"bytes_base64"`
}

type environmentBindingIdentity struct {
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Value    string `json:"value"`
}

type fileIdentity struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentsBase64 string `json:"contents_base64"`
}

type cliIdentity struct {
	Executable  string                       `json:"executable"`
	BaseArgv    []string                     `json:"base_argv"`
	Argv        []string                     `json:"argv"`
	Stdin       stdinIdentity                `json:"stdin"`
	Environment []environmentBindingIdentity `json:"environment"`
	Fixtures    []fileIdentity               `json:"fixtures"`
	CWDPolicy   string                       `json:"cwd_policy"`
	StdoutBytes int64                        `json:"stdout_bytes"`
	StderrBytes int64                        `json:"stderr_bytes"`
}

type presenceValueIdentity struct {
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Value    string `json:"value"`
}

type headerIdentity struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bodyIdentity struct {
	Presence    string `json:"presence"`
	BytesBase64 string `json:"bytes_base64"`
}

type httpIdentity struct {
	Method            string                  `json:"method"`
	Path              string                  `json:"path"`
	Query             []presenceValueIdentity `json:"ordered_query_multimap"`
	Headers           []headerIdentity        `json:"ordered_request_header_multimap"`
	Body              bodyIdentity            `json:"body"`
	Seeds             []fileIdentity          `json:"seeds"`
	StartAuthority    string                  `json:"start_authority"`
	ReadinessSignal   string                  `json:"readiness_signal"`
	ReadinessProtocol string                  `json:"readiness_protocol"`
	StatusLineBytes   int64                   `json:"status_line_bytes"`
	HeaderBytes       int64                   `json:"header_bytes"`
	HeaderCount       int                     `json:"header_count"`
	BodyBytes         int64                   `json:"body_bytes"`
}

type limitsIdentity struct {
	MaterializedEntryCount    int   `json:"materialized_entry_count"`
	MaterializedBytesPerWorld int64 `json:"materialized_bytes_per_world"`
	SingleBlobBytes           int64 `json:"single_blob_bytes"`
	ReadinessMS               int64 `json:"readiness_ms"`
	ProbeMS                   int64 `json:"probe_ms"`
	TeardownMS                int64 `json:"teardown_ms"`
	StdoutBytes               int64 `json:"stdout_bytes"`
	StderrBytes               int64 `json:"stderr_bytes"`
	HTTPBodyBytes             int64 `json:"http_body_bytes"`
}

type closedFactsIdentity struct {
	PrivateRootPolicy          string `json:"private_root_policy"`
	EnvironmentProfile         string `json:"environment_profile"`
	AmbientExecutableLookup    bool   `json:"ambient_executable_lookup"`
	ShellExecution             bool   `json:"shell_execution"`
	SetupCommand               bool   `json:"setup_command"`
	ExternalHost               bool   `json:"external_host"`
	ConfidentialityEstablished bool   `json:"confidentiality_established"`
}

type sourceIdentity struct {
	SchemaVersion           string                    `json:"schema_version"`
	Kind                    string                    `json:"kind"`
	Version                 string                    `json:"version"`
	SourceProfile           string                    `json:"source_profile"`
	Adapter                 string                    `json:"adapter"`
	LaunchProfile           string                    `json:"launch_profile"`
	StartProfile            string                    `json:"start_profile"`
	Entrypoint              string                    `json:"entrypoint"`
	PlanBase64              string                    `json:"world_plan_base64"`
	PlanDigest              string                    `json:"world_plan_digest"`
	ProjectionBindingBase64 string                    `json:"projection_binding_base64"`
	ProjectionBindingDigest string                    `json:"projection_binding_digest"`
	PortableProfileBase64   string                    `json:"portable_profile_base64"`
	PortableProfileDigest   string                    `json:"portable_profile_digest"`
	AdapterProjectionBase64 string                    `json:"adapter_projection_definition_base64"`
	AdapterProjectionDigest string                    `json:"adapter_projection_definition_digest"`
	StimulusKind            string                    `json:"stimulus_kind"`
	StimulusBase64          string                    `json:"stimulus_base64"`
	StimulusDigest          string                    `json:"stimulus_digest"`
	ExecutionBindingBase64  string                    `json:"execution_binding_base64"`
	ExecutionBindingDigest  string                    `json:"execution_binding_digest"`
	PlanEnvironment         []domain.EnvironmentEntry `json:"plan_environment"`
	PlanSecretSlots         []domain.SecretSlot       `json:"plan_secret_slots"`
	Limits                  limitsIdentity            `json:"limits"`
	ClosedFacts             closedFactsIdentity       `json:"closed_facts"`
	CLI                     *cliIdentity              `json:"cli"`
	HTTP                    *httpIdentity             `json:"http"`
}

func NewCLISource(input CLIInput) (PortableSource, error) {
	if !input.Plan.Digest().Valid() || !input.Stimulus.Valid() || !input.Capture.Valid() ||
		!input.Profile.Valid() || !input.Projection.Valid() {
		return PortableSource{}, refuse(CodePortableSourceRequired, "exact CLI plan, stimulus, capture, profile, and projection authorities are required", nil)
	}
	if err := validateCLIProjection(input.Plan, input.Profile, input.Projection); err != nil {
		return PortableSource{}, err
	}
	if err := validateNodePlan(input.Plan, input.Stimulus.Executable(), input.Stimulus.BaseArgv(), domain.AdapterCLI); err != nil {
		return PortableSource{}, err
	}
	execution, err := cli.BindExecution(input.Plan, input.Stimulus, input.Capture, input.Projection)
	if err != nil || execution.ProjectionDefinitionBinding().Digest() != input.Projection.Binding().Digest() {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "CLI execution binding did not reconstruct from the exact plan", err)
	}
	return buildSource(cliSourceIdentity(
		input.Plan, input.Profile, input.Projection, input.Stimulus, input.Capture, execution,
	))
}

func NewHTTPSource(input HTTPInput) (PortableSource, error) {
	if !input.Plan.Digest().Valid() || !input.Stimulus.Valid() || !input.Start.Valid() ||
		!input.Capture.Valid() || !input.Readiness.Valid() || !input.Profile.Valid() || !input.Projection.Valid() {
		return PortableSource{}, refuse(CodePortableSourceRequired, "exact HTTP plan, stimulus, start, capture, readiness, profile, and projection authorities are required", nil)
	}
	if input.Start.Authority() != counterhttp.HTTPPortableStartAuthorityV1 ||
		input.Readiness.Protocol() != counterhttp.PortableReadinessProtocolV1 {
		return PortableSource{}, refuse(CodeNonportableStartProfile, "HTTP source requires the child-bind pipe-ready profile", nil)
	}
	if err := validateHTTPProjection(input.Plan, input.Profile, input.Projection); err != nil {
		return PortableSource{}, err
	}
	if err := validateNodePlan(input.Plan, input.Start.Executable(), []string{input.Start.Entrypoint()}, domain.AdapterHTTP); err != nil {
		return PortableSource{}, err
	}
	execution, err := counterhttp.BindExecution(
		input.Plan, input.Stimulus, input.Start, input.Capture, input.Readiness, input.Projection,
	)
	if err != nil || execution.ProjectionDefinitionBinding().Digest() != input.Projection.Binding().Digest() ||
		execution.Authority() != counterhttp.HTTPPortableExecutionAuthorityV1 {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "HTTP execution binding did not reconstruct from the exact plan", err)
	}
	return buildSource(httpSourceIdentity(
		input.Plan, input.Profile, input.Projection, input.Stimulus, input.Start, input.Capture, input.Readiness, execution,
	))
}

func cliSourceIdentity(
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection cli.CLIProjectionAuthority,
	stimulus cli.CLIStimulus,
	capture cli.CLICapturePolicy,
	execution cli.CLIExecutionBinding,
) sourceIdentity {
	stdin := stimulus.Stdin()
	wire := &cliIdentity{
		Executable: stimulus.Executable(), BaseArgv: stimulus.BaseArgv(), Argv: stimulus.Argv(),
		Stdin:       stdinIdentity{Presence: string(stdin.Presence()), BytesBase64: base64.StdEncoding.EncodeToString(stdin.Bytes())},
		Environment: make([]environmentBindingIdentity, len(stimulus.Environment())),
		Fixtures:    make([]fileIdentity, len(stimulus.Fixtures())), CWDPolicy: string(stimulus.CWDPolicy()),
		StdoutBytes: capture.StdoutBytes(), StderrBytes: capture.StderrBytes(),
	}
	for index, entry := range stimulus.Environment() {
		wire.Environment[index] = environmentBindingIdentity{
			Name: entry.Name(), Presence: string(entry.Presence()), Value: entry.Value(),
		}
	}
	for index, file := range stimulus.Fixtures() {
		wire.Fixtures[index] = fileIdentity{
			Path: file.Path(), Mode: string(file.Mode()), ContentsBase64: base64.StdEncoding.EncodeToString(file.Contents()),
		}
	}
	identity := commonIdentity(
		plan, profile, projection.CanonicalBytes(), projection.Digest(),
		stimulus.CanonicalBytes(), stimulus.Digest(), execution.CanonicalBytes(), execution.Digest(),
	)
	identity.StartProfile = CLIStartProfileV1
	identity.Entrypoint = stimulus.BaseArgv()[0]
	identity.CLI = wire
	return identity
}

func httpSourceIdentity(
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection counterhttp.HTTPProjectionAuthority,
	stimulus counterhttp.HTTPStimulus,
	start counterhttp.HTTPStartSpec,
	capture counterhttp.HTTPCapturePolicy,
	readiness counterhttp.HTTPReadinessContract,
	execution counterhttp.HTTPExecutionBinding,
) sourceIdentity {
	body := stimulus.Body()
	wire := &httpIdentity{
		Method: string(stimulus.Method()), Path: stimulus.Path(),
		Query:   make([]presenceValueIdentity, len(stimulus.Query())),
		Headers: make([]headerIdentity, len(stimulus.Headers())),
		Body:    bodyIdentity{Presence: string(body.Presence()), BytesBase64: base64.StdEncoding.EncodeToString(body.Bytes())},
		Seeds:   make([]fileIdentity, len(stimulus.Seeds())), StartAuthority: start.Authority(),
		ReadinessSignal: readiness.SignalName(), ReadinessProtocol: readiness.Protocol(),
		StatusLineBytes: capture.StatusLineBytes(), HeaderBytes: capture.HeaderBytes(),
		HeaderCount: capture.HeaderCount(), BodyBytes: capture.BodyBytes(),
	}
	for index, entry := range stimulus.Query() {
		value, _ := entry.Value()
		wire.Query[index] = presenceValueIdentity{Name: entry.Name(), Presence: string(entry.Presence()), Value: value}
	}
	for index, header := range stimulus.Headers() {
		wire.Headers[index] = headerIdentity{Name: header.Name(), Value: header.Value()}
	}
	for index, file := range stimulus.Seeds() {
		wire.Seeds[index] = fileIdentity{
			Path: file.Path(), Mode: string(file.Mode()), ContentsBase64: base64.StdEncoding.EncodeToString(file.Contents()),
		}
	}
	identity := commonIdentity(
		plan, profile, projection.CanonicalBytes(), projection.Digest(),
		stimulus.CanonicalBytes(), stimulus.Digest(), execution.CanonicalBytes(), execution.Digest(),
	)
	identity.StartProfile = start.Authority()
	identity.Entrypoint = start.Entrypoint()
	identity.HTTP = wire
	return identity
}

func commonIdentity(
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	adapterProjectionBytes []byte,
	adapterProjectionDigest domain.Digest,
	stimulusBytes []byte,
	stimulusDigest domain.Digest,
	executionBytes []byte,
	executionDigest domain.Digest,
) sourceIdentity {
	budgets := plan.Budgets()
	return sourceIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "PortableSource", Version: SourceVersionV1,
		SourceProfile: SourceProfileV1, Adapter: string(plan.Adapter().Domain), LaunchProfile: LaunchProfileV1,
		PlanBase64: base64.StdEncoding.EncodeToString(plan.CanonicalBytes()), PlanDigest: plan.Digest().String(),
		ProjectionBindingBase64: base64.StdEncoding.EncodeToString(plan.ProjectionDefinitionBinding().CanonicalBytes()),
		ProjectionBindingDigest: plan.ProjectionDefinitionBinding().Digest().String(),
		PortableProfileBase64:   base64.StdEncoding.EncodeToString(profile.CanonicalBytes()),
		PortableProfileDigest:   profile.Digest().String(), StimulusKind: plan.Adapter().Domain.CanonicalStimulusKind(),
		AdapterProjectionBase64: base64.StdEncoding.EncodeToString(adapterProjectionBytes),
		AdapterProjectionDigest: adapterProjectionDigest.String(),
		StimulusBase64:          base64.StdEncoding.EncodeToString(stimulusBytes), StimulusDigest: stimulusDigest.String(),
		ExecutionBindingBase64: base64.StdEncoding.EncodeToString(executionBytes), ExecutionBindingDigest: executionDigest.String(),
		PlanEnvironment: slices.Clone(plan.Environment()), PlanSecretSlots: slices.Clone(plan.SecretSlots()),
		Limits: limitsIdentity{
			MaterializedEntryCount: budgets.MaterializedEntryCount, MaterializedBytesPerWorld: budgets.MaterializedBytesPerWorld,
			SingleBlobBytes: budgets.SingleBlobBytes, ReadinessMS: budgets.ReadinessMS, ProbeMS: budgets.ProbeMS,
			TeardownMS: budgets.TeardownMS, StdoutBytes: budgets.StdoutBytes, StderrBytes: budgets.StderrBytes,
			HTTPBodyBytes: budgets.HTTPBodyBytes,
		},
		ClosedFacts: closedFactsIdentity{
			PrivateRootPolicy: PrivateRootPolicyV1, EnvironmentProfile: EnvironmentProfileV1,
			AmbientExecutableLookup: false, ShellExecution: false, SetupCommand: false,
			ExternalHost: false, ConfidentialityEstablished: false,
		},
	}
}

func buildSource(identity sourceIdentity) (PortableSource, error) {
	if identity.CLI == nil == (identity.HTTP == nil) {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "source must contain exactly one adapter arm", nil)
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "PortableSource" ||
		identity.Version != SourceVersionV1 || identity.SourceProfile != SourceProfileV1 ||
		identity.LaunchProfile != LaunchProfileV1 || identity.ClosedFacts != (closedFactsIdentity{
		PrivateRootPolicy: PrivateRootPolicyV1, EnvironmentProfile: EnvironmentProfileV1,
	}) {
		// Boolean closed facts are all required false; equality to this literal
		// rejects any attempted overclaim or ambient execution authority.
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source closed facts disagree", nil)
	}
	if len(identity.PlanSecretSlots) != 0 {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "portable source cannot retain a plan requiring secret slots", nil)
	}
	canonicalBytes, err := canon.CanonicalizeTyped(identity)
	if err != nil || len(canonicalBytes) == 0 || len(canonicalBytes) > MaxSourceCanonicalBytes {
		return PortableSource{}, refuse(CodeSourceLimitExceeded, "portable source exceeds its canonical ceiling", err)
	}
	digestRaw, err := canon.DigestBytes("PortableSource", canonicalBytes)
	if err != nil {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source digest failed", err)
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source digest is invalid", err)
	}
	return reconstructSource(identity, canonicalBytes, digest)
}

func reconstructSource(identity sourceIdentity, canonicalBytes []byte, digest domain.Digest) (PortableSource, error) {
	bindingBytes, err := strictBase64(identity.ProjectionBindingBase64, MaxSourceCanonicalBytes)
	if err != nil {
		return PortableSource{}, err
	}
	binding, err := domain.ParseProjectionDefinitionBinding(bindingBytes)
	if err != nil || binding.Digest().String() != identity.ProjectionBindingDigest {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "projection binding bytes and identity disagree", err)
	}
	planBytes, err := strictBase64(identity.PlanBase64, MaxSourceCanonicalBytes)
	if err != nil {
		return PortableSource{}, err
	}
	plan, err := domain.ParseWorldPlan(planBytes, binding)
	if err != nil || plan.Digest().String() != identity.PlanDigest || string(plan.Adapter().Domain) != identity.Adapter ||
		!slices.Equal(plan.Environment(), identity.PlanEnvironment) || !slices.Equal(plan.SecretSlots(), identity.PlanSecretSlots) ||
		limitsFrom(plan.Budgets()) != identity.Limits {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "world plan bytes and source facts disagree", err)
	}
	profileBytes, err := strictBase64(identity.PortableProfileBase64, MaxSourceCanonicalBytes)
	if err != nil {
		return PortableSource{}, err
	}
	profile, err := projectionprofile.Parse(profileBytes, binding)
	if err != nil || profile.Digest().String() != identity.PortableProfileDigest {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "portable profile did not reconstruct exactly", err)
	}
	adapterProjectionBytes, err := strictBase64(identity.AdapterProjectionBase64, MaxSourceCanonicalBytes)
	if err != nil {
		return PortableSource{}, err
	}
	adapterProjectionDigest, err := domain.ParseDigest(identity.AdapterProjectionDigest)
	if err != nil {
		return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "adapter projection digest is invalid", err)
	}
	stimulusBytes, err := strictBase64(identity.StimulusBase64, maxSourcePayloadBytes)
	if err != nil {
		return PortableSource{}, err
	}
	executionBytes, err := strictBase64(identity.ExecutionBindingBase64, MaxSourceCanonicalBytes)
	if err != nil {
		return PortableSource{}, err
	}
	var stimulusDigest, executionDigest domain.Digest
	var entrypoint, startProfile string
	var reconstructedIdentity sourceIdentity
	var authorities reconstructedAuthorities
	if identity.CLI != nil {
		projection, projectionErr := cli.ResolveCLIProjectionAuthority(
			adapterProjectionDigest, adapterProjectionBytes, binding,
		)
		if projectionErr != nil {
			return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "CLI projection authority did not reconstruct", projectionErr)
		}
		if err := validateCLIProjection(plan, profile, projection); err != nil {
			return PortableSource{}, err
		}
		stimulusDigest, executionDigest, entrypoint, startProfile, err = reconstructCLI(
			identity, plan, profile, projection, stimulusBytes, executionBytes, &reconstructedIdentity, &authorities,
		)
	} else {
		projection, projectionErr := counterhttp.ResolveHTTPProjectionAuthority(
			adapterProjectionDigest, adapterProjectionBytes, binding,
		)
		if projectionErr != nil {
			return PortableSource{}, refuse(CodeSourceAuthorityMismatch, "HTTP projection authority did not reconstruct", projectionErr)
		}
		if err := validateHTTPProjection(plan, profile, projection); err != nil {
			return PortableSource{}, err
		}
		stimulusDigest, executionDigest, entrypoint, startProfile, err = reconstructHTTP(
			identity, plan, profile, projection, stimulusBytes, executionBytes, &reconstructedIdentity, &authorities,
		)
	}
	if err != nil {
		return PortableSource{}, err
	}
	reconstructedBytes, err := canon.CanonicalizeTyped(reconstructedIdentity)
	if err != nil || !bytes.Equal(reconstructedBytes, canonicalBytes) {
		return PortableSource{}, refuse(
			CodeSourceAuthorityMismatch, "source envelope did not regenerate byte-exactly from reconstructed authorities", err,
		)
	}
	return PortableSource{
		digest: digest, canonical: append([]byte(nil), canonicalBytes...), plan: plan, binding: binding, profile: profile,
		stimulusKind: identity.StimulusKind, stimulusBytes: append([]byte(nil), stimulusBytes...),
		stimulusDigest: stimulusDigest, executionBinding: executionDigest, executionBytes: append([]byte(nil), executionBytes...),
		adapter: plan.Adapter().Domain, entrypoint: entrypoint, startProfile: startProfile,
		authorities: authorities, seal: portableSourceAuthority,
	}, nil
}

func reconstructCLI(
	identity sourceIdentity,
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection cli.CLIProjectionAuthority,
	stimulusBytes, executionBytes []byte,
	reconstructedIdentity *sourceIdentity,
	authorities *reconstructedAuthorities,
) (domain.Digest, domain.Digest, string, string, error) {
	wire := identity.CLI
	if identity.HTTP != nil || identity.Adapter != string(domain.AdapterCLI) || identity.StimulusKind != "CLIStimulus" ||
		identity.StartProfile != CLIStartProfileV1 || len(wire.BaseArgv) != 1 || wire.BaseArgv[0] != identity.Entrypoint {
		return "", "", "", "", refuse(CodeSourceWireInvalid, "CLI source arm or launch profile is invalid", nil)
	}
	stdinBytes, err := strictBase64(wire.Stdin.BytesBase64, maxSourcePayloadBytes)
	if err != nil {
		return "", "", "", "", err
	}
	var stdin cli.CLIStdin
	switch cli.Presence(wire.Stdin.Presence) {
	case cli.PresenceAbsent:
		if len(stdinBytes) != 0 {
			return "", "", "", "", refuse(CodeSourceWireInvalid, "absent CLI stdin carries bytes", nil)
		}
		stdin = cli.AbsentStdin()
	case cli.PresencePresent:
		stdin, err = cli.PresentStdin(stdinBytes)
	default:
		err = refuse(CodeSourceWireInvalid, "CLI stdin presence is invalid", nil)
	}
	if err != nil {
		return "", "", "", "", err
	}
	environment := make([]cli.CLIEnvironmentBinding, len(wire.Environment))
	for index, entry := range wire.Environment {
		switch cli.Presence(entry.Presence) {
		case cli.PresenceAbsent:
			if entry.Value != "" {
				return "", "", "", "", refuse(CodeSourceWireInvalid, "absent CLI environment entry carries a value", nil)
			}
			environment[index], err = cli.AbsentEnvironment(entry.Name)
		case cli.PresencePresent:
			environment[index], err = cli.PresentEnvironment(entry.Name, entry.Value)
		default:
			err = refuse(CodeSourceWireInvalid, "CLI environment presence is invalid", nil)
		}
		if err != nil {
			return "", "", "", "", err
		}
	}
	fixtures := make([]cli.CLIFixtureFile, len(wire.Fixtures))
	for index, file := range wire.Fixtures {
		contents, decodeErr := strictBase64(file.ContentsBase64, maxSourcePayloadBytes)
		if decodeErr != nil {
			return "", "", "", "", decodeErr
		}
		fixtures[index], err = cli.NewFixtureFile(file.Path, contents, cli.FixtureMode(file.Mode))
		if err != nil {
			return "", "", "", "", err
		}
	}
	stimulus, err := cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable: wire.Executable, BaseArgv: wire.BaseArgv, Argv: wire.Argv, Stdin: stdin,
		Environment: environment, Fixtures: fixtures, CWDPolicy: cli.CWDPolicy(wire.CWDPolicy),
	})
	if err != nil || !bytes.Equal(stimulus.CanonicalBytes(), stimulusBytes) || stimulus.Digest().String() != identity.StimulusDigest {
		return "", "", "", "", refuse(CodeSourceAuthorityMismatch, "CLI stimulus did not reconstruct byte-exactly", err)
	}
	if err := validateNodePlan(plan, stimulus.Executable(), stimulus.BaseArgv(), domain.AdapterCLI); err != nil {
		return "", "", "", "", err
	}
	capture, err := cli.NewCLICapturePolicy(cli.CLICapturePolicyConfig{StdoutBytes: wire.StdoutBytes, StderrBytes: wire.StderrBytes})
	if err != nil {
		return "", "", "", "", err
	}
	execution, err := cli.BindExecution(plan, stimulus, capture, projection)
	if err != nil || !bytes.Equal(execution.CanonicalBytes(), executionBytes) || execution.Digest().String() != identity.ExecutionBindingDigest {
		return "", "", "", "", refuse(CodeSourceAuthorityMismatch, "CLI execution binding did not reconstruct byte-exactly", err)
	}
	*reconstructedIdentity = cliSourceIdentity(plan, profile, projection, stimulus, capture, execution)
	authorities.cliStimulus, authorities.cliCapture, authorities.cliProjection = stimulus, capture, projection
	return stimulus.Digest(), execution.Digest(), identity.Entrypoint, identity.StartProfile, nil
}

func reconstructHTTP(
	identity sourceIdentity,
	plan domain.WorldPlan,
	profile projectionprofile.Profile,
	projection counterhttp.HTTPProjectionAuthority,
	stimulusBytes, executionBytes []byte,
	reconstructedIdentity *sourceIdentity,
	authorities *reconstructedAuthorities,
) (domain.Digest, domain.Digest, string, string, error) {
	wire := identity.HTTP
	if identity.CLI != nil || identity.Adapter != string(domain.AdapterHTTP) || identity.StimulusKind != "HTTPStimulus" ||
		identity.StartProfile != counterhttp.HTTPPortableStartAuthorityV1 ||
		wire.StartAuthority != counterhttp.HTTPPortableStartAuthorityV1 ||
		wire.ReadinessSignal != counterhttp.PortableReadinessSignalNameV1 ||
		wire.ReadinessProtocol != counterhttp.PortableReadinessProtocolV1 {
		return "", "", "", "", refuse(CodeNonportableStartProfile, "HTTP source arm is not child-bind pipe-ready", nil)
	}
	query := make([]counterhttp.HTTPQueryEntry, len(wire.Query))
	var err error
	for index, entry := range wire.Query {
		switch counterhttp.Presence(entry.Presence) {
		case counterhttp.PresenceAbsent:
			if entry.Value != "" {
				return "", "", "", "", refuse(CodeSourceWireInvalid, "absent HTTP query member carries a value", nil)
			}
			query[index], err = counterhttp.QueryFlag(entry.Name)
		case counterhttp.PresencePresent:
			query[index], err = counterhttp.QueryValue(entry.Name, entry.Value)
		default:
			err = refuse(CodeSourceWireInvalid, "HTTP query presence is invalid", nil)
		}
		if err != nil {
			return "", "", "", "", err
		}
	}
	headers := make([]counterhttp.HTTPRequestHeader, len(wire.Headers))
	for index, header := range wire.Headers {
		headers[index], err = counterhttp.NewRequestHeader(header.Name, header.Value)
		if err != nil {
			return "", "", "", "", err
		}
	}
	bodyBytes, err := strictBase64(wire.Body.BytesBase64, maxSourcePayloadBytes)
	if err != nil {
		return "", "", "", "", err
	}
	var body counterhttp.HTTPBody
	switch counterhttp.Presence(wire.Body.Presence) {
	case counterhttp.PresenceAbsent:
		if len(bodyBytes) != 0 {
			return "", "", "", "", refuse(CodeSourceWireInvalid, "absent HTTP body carries bytes", nil)
		}
		body = counterhttp.AbsentBody()
	case counterhttp.PresencePresent:
		body, err = counterhttp.PresentBody(bodyBytes)
	default:
		err = refuse(CodeSourceWireInvalid, "HTTP body presence is invalid", nil)
	}
	if err != nil {
		return "", "", "", "", err
	}
	seeds := make([]counterhttp.HTTPSeedFile, len(wire.Seeds))
	for index, file := range wire.Seeds {
		contents, decodeErr := strictBase64(file.ContentsBase64, maxSourcePayloadBytes)
		if decodeErr != nil {
			return "", "", "", "", decodeErr
		}
		seeds[index], err = counterhttp.NewSeedFile(file.Path, contents, counterhttp.SeedMode(file.Mode))
		if err != nil {
			return "", "", "", "", err
		}
	}
	stimulus, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: counterhttp.HTTPMethod(wire.Method), Path: wire.Path, Query: query, Headers: headers, Body: body, Seeds: seeds,
	})
	if err != nil || !bytes.Equal(stimulus.CanonicalBytes(), stimulusBytes) || stimulus.Digest().String() != identity.StimulusDigest {
		return "", "", "", "", refuse(CodeSourceAuthorityMismatch, "HTTP stimulus did not reconstruct byte-exactly", err)
	}
	start, err := counterhttp.NewPortableHTTPStartSpec(identity.Entrypoint)
	if err != nil || start.Authority() != wire.StartAuthority {
		return "", "", "", "", refuse(CodeNonportableStartProfile, "HTTP start authority did not reconstruct", err)
	}
	readiness, err := counterhttp.NewPortableHTTPReadinessContract()
	if err != nil || readiness.SignalName() != wire.ReadinessSignal || readiness.Protocol() != wire.ReadinessProtocol {
		return "", "", "", "", refuse(CodeNonportableStartProfile, "HTTP readiness authority did not reconstruct", err)
	}
	if err := validateNodePlan(plan, start.Executable(), []string{start.Entrypoint()}, domain.AdapterHTTP); err != nil {
		return "", "", "", "", err
	}
	capture, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: wire.StatusLineBytes, HeaderBytes: wire.HeaderBytes,
		HeaderCount: wire.HeaderCount, BodyBytes: wire.BodyBytes,
	})
	if err != nil {
		return "", "", "", "", err
	}
	execution, err := counterhttp.BindExecution(plan, stimulus, start, capture, readiness, projection)
	if err != nil || execution.Authority() != counterhttp.HTTPPortableExecutionAuthorityV1 ||
		!bytes.Equal(execution.CanonicalBytes(), executionBytes) || execution.Digest().String() != identity.ExecutionBindingDigest {
		return "", "", "", "", refuse(CodeSourceAuthorityMismatch, "HTTP execution binding did not reconstruct byte-exactly", err)
	}
	*reconstructedIdentity = httpSourceIdentity(plan, profile, projection, stimulus, start, capture, readiness, execution)
	authorities.httpStimulus, authorities.httpStart = stimulus, start
	authorities.httpCapture, authorities.httpReadiness, authorities.httpProjection = capture, readiness, projection
	return stimulus.Digest(), execution.Digest(), identity.Entrypoint, identity.StartProfile, nil
}

func validateNodePlan(plan domain.WorldPlan, executable string, baseArgv []string, adapter domain.AdapterDomain) error {
	if executable != "node" || len(baseArgv) != 1 || len(plan.SetupArgv()) != 0 ||
		!runnerprofile.ValidRepositoryNodeEntrypoint(baseArgv[0]) ||
		!slices.Equal(plan.StartArgv(), []string{"node", baseArgv[0]}) ||
		plan.Adapter().Domain != adapter {
		return refuse(CodeNonportableStartProfile, "plan is not one exact repository-relative Node script launch", nil)
	}
	tools := plan.RequiredTools()
	if len(tools) != 1 || tools[0].Name != "node" || tools[0].VersionConstraint != NodeToolConstraintV1 {
		return refuse(CodeNonportableStartProfile, "plan must declare only logical node as its required tool", nil)
	}
	expectedRunner, err := RequiredRunnerDigest(adapter)
	if err != nil || plan.Adapter().RunnerDigest != expectedRunner {
		return refuse(CodeNonportableStartProfile, "plan runner lineage is outside the exact portable roster", err)
	}
	if adapter == domain.AdapterCLI && (plan.Adapter().AdapterVersion != CLIAdapterVersionV1 ||
		plan.ExecutionShape() != domain.OneCLIInvocation || plan.Readiness().Kind != domain.ReadinessNone) {
		return refuse(CodeNonportableStartProfile, "CLI plan execution shape is not portable", nil)
	}
	if adapter == domain.AdapterHTTP && (plan.Adapter().AdapterVersion != HTTPAdapterVersionV1 ||
		plan.ExecutionShape() != domain.OneLoopbackHTTPRequest ||
		plan.Readiness().Kind != domain.FixtureOwnedReadiness ||
		plan.Readiness().SignalName != counterhttp.PortableReadinessSignalNameV1) {
		return refuse(CodeNonportableStartProfile, "HTTP plan execution shape is not child-bind pipe-ready", nil)
	}
	return nil
}

// RequiredRunnerDigest derives the one exact historical/portable runner
// lineage admitted by the source constructor. It is structural identity, not
// evidence that a runner was physically invoked.
func RequiredRunnerDigest(adapter domain.AdapterDomain) (domain.Digest, error) {
	switch adapter {
	case domain.AdapterCLI:
		return runnerprofile.CLIDigest()
	case domain.AdapterHTTP:
		return runnerprofile.HTTPPortableDigest()
	default:
		return "", refuse(CodeNonportableStartProfile, "adapter has no portable runner lineage", nil)
	}
}

func limitsFrom(budgets domain.Budgets) limitsIdentity {
	return limitsIdentity{
		MaterializedEntryCount: budgets.MaterializedEntryCount, MaterializedBytesPerWorld: budgets.MaterializedBytesPerWorld,
		SingleBlobBytes: budgets.SingleBlobBytes, ReadinessMS: budgets.ReadinessMS, ProbeMS: budgets.ProbeMS,
		TeardownMS: budgets.TeardownMS, StdoutBytes: budgets.StdoutBytes, StderrBytes: budgets.StderrBytes,
		HTTPBodyBytes: budgets.HTTPBodyBytes,
	}
}

func strictBase64(raw string, limit int) ([]byte, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || len(decoded) > limit || base64.StdEncoding.EncodeToString(decoded) != raw {
		return nil, refuse(CodeSourceWireInvalid, "portable source contains invalid or oversized base64", err)
	}
	return decoded, nil
}

func Parse(exact []byte) (PortableSource, error) {
	if len(exact) == 0 || len(exact) > MaxSourceCanonicalBytes {
		return PortableSource{}, refuse(CodePortableSourceRequired, "portable source bytes are empty or oversized", nil)
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source is not strict canonical JSON", err)
	}
	canonicalBytes, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonicalBytes, exact) {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source bytes are noncanonical", err)
	}
	var identity sourceIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source typed decode failed", err)
	}
	rebuilt, err := buildSource(identity)
	if err != nil {
		if _, semantic := CodeOf(err); semantic {
			return PortableSource{}, err
		}
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source reconstruction failed", err)
	}
	if !bytes.Equal(rebuilt.canonical, exact) {
		return PortableSource{}, refuse(CodeSourceWireInvalid, "portable source did not reconstruct exactly", err)
	}
	return rebuilt, nil
}

func (s PortableSource) Valid() bool {
	if s.seal != portableSourceAuthority || !s.digest.Valid() || len(s.canonical) == 0 {
		return false
	}
	rebuilt, err := Parse(s.canonical)
	return err == nil && rebuilt.digest == s.digest && bytes.Equal(rebuilt.canonical, s.canonical)
}

func (s PortableSource) Digest() domain.Digest                                 { return s.digest }
func (s PortableSource) CanonicalBytes() []byte                                { return append([]byte(nil), s.canonical...) }
func (s PortableSource) Adapter() domain.AdapterDomain                         { return s.adapter }
func (s PortableSource) Plan() domain.WorldPlan                                { return s.plan }
func (s PortableSource) ProjectionBinding() domain.ProjectionDefinitionBinding { return s.binding }
func (s PortableSource) Profile() projectionprofile.Profile                    { return s.profile }
func (s PortableSource) StimulusKind() string                                  { return s.stimulusKind }
func (s PortableSource) StimulusCanonicalBytes() []byte {
	return append([]byte(nil), s.stimulusBytes...)
}
func (s PortableSource) StimulusDigest() domain.Digest         { return s.stimulusDigest }
func (s PortableSource) ExecutionBindingDigest() domain.Digest { return s.executionBinding }
func (s PortableSource) ExecutionBindingCanonicalBytes() []byte {
	return append([]byte(nil), s.executionBytes...)
}
func (s PortableSource) Entrypoint() string   { return s.entrypoint }
func (s PortableSource) StartProfile() string { return s.startProfile }

func (s PortableSource) CLIView() (CLISourceView, bool) {
	if !s.Valid() || s.adapter != domain.AdapterCLI || !s.authorities.cliStimulus.Valid() ||
		!s.authorities.cliCapture.Valid() || !s.authorities.cliProjection.Valid() {
		return CLISourceView{}, false
	}
	return CLISourceView{
		sourceDigest: s.digest, stimulus: s.authorities.cliStimulus,
		capture: s.authorities.cliCapture, projection: s.authorities.cliProjection, seal: portableSourceAuthority,
	}, true
}

func (s PortableSource) HTTPView() (HTTPSourceView, bool) {
	if !s.Valid() || s.adapter != domain.AdapterHTTP || !s.authorities.httpStimulus.Valid() ||
		!s.authorities.httpStart.Valid() || !s.authorities.httpCapture.Valid() || !s.authorities.httpReadiness.Valid() ||
		!s.authorities.httpProjection.Valid() {
		return HTTPSourceView{}, false
	}
	return HTTPSourceView{
		sourceDigest: s.digest, stimulus: s.authorities.httpStimulus, start: s.authorities.httpStart,
		capture: s.authorities.httpCapture, readiness: s.authorities.httpReadiness,
		projection: s.authorities.httpProjection, seal: portableSourceAuthority,
	}, true
}

func (v CLISourceView) Valid() bool {
	return v.seal == portableSourceAuthority && v.sourceDigest.Valid() && v.stimulus.Valid() &&
		v.capture.Valid() && v.projection.Valid()
}
func (v CLISourceView) SourceDigest() domain.Digest            { return v.sourceDigest }
func (v CLISourceView) Stimulus() cli.CLIStimulus              { return v.stimulus }
func (v CLISourceView) Capture() cli.CLICapturePolicy          { return v.capture }
func (v CLISourceView) Projection() cli.CLIProjectionAuthority { return v.projection }

func (v HTTPSourceView) Valid() bool {
	return v.seal == portableSourceAuthority && v.sourceDigest.Valid() && v.stimulus.Valid() && v.start.Valid() &&
		v.start.Authority() == counterhttp.HTTPPortableStartAuthorityV1 && v.capture.Valid() && v.readiness.Valid() &&
		v.readiness.Protocol() == counterhttp.PortableReadinessProtocolV1 && v.projection.Valid()
}
func (v HTTPSourceView) SourceDigest() domain.Digest                     { return v.sourceDigest }
func (v HTTPSourceView) Stimulus() counterhttp.HTTPStimulus              { return v.stimulus }
func (v HTTPSourceView) Start() counterhttp.HTTPStartSpec                { return v.start }
func (v HTTPSourceView) Capture() counterhttp.HTTPCapturePolicy          { return v.capture }
func (v HTTPSourceView) Readiness() counterhttp.HTTPReadinessContract    { return v.readiness }
func (v HTTPSourceView) Projection() counterhttp.HTTPProjectionAuthority { return v.projection }
