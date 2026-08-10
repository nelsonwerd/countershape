package httpstudy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

const (
	discoveryRepetitions = 3
	discoveryTrialBudget = 12
	discoveryWallBudget  = 2 * time.Minute
)

// Config binds one HTTP reference run to the harness-owned repository and
// private process roots. The repository is the candidate authority observed by
// the outer U7 harness; this package never creates a nested candidate repo.
type Config struct {
	RepositoryRoot string
	ScratchRoot    string
	EvidenceRoot   string
	GitExecutable  string
	NodeExecutable string
	Ordinal        int
	runStandalone  standaloneRunner
}

// standaloneRunInput is the narrow physical seam from HTTP workflow
// authority to the app-owned Node process boundary. It deliberately carries
// no semantic outcome supplied by the caller.
type standaloneRunInput struct {
	workingDirectory   string
	testFile           string
	homeDirectory      string
	temporaryDirectory string
	timeout            time.Duration
}

type standaloneRunObservation struct {
	stdout           []byte
	stderr           []byte
	invocationSHA256 string
	testFileSHA256   string
	exitCode         int
}

type standaloneRunner func(context.Context, standaloneRunInput) (standaloneRunObservation, error)

type phaseRunSpec struct {
	label                   string
	purpose                 domain.AttemptPurpose
	repetitions             int
	planDiscoveryRepeats    int
	planConfirmationRepeats int
	stimulus                *counterhttp.HTTPStimulus
	confirmation            *confirmationRunInput
}

type confirmationRunInput struct {
	reducedBaseline compare.DivergentBaseline
	reductionRun    reduce.ReductionRun
	reductionResult reduction.Result
}

// Trial is the typed adapter bridge retained for product tests and U7D replay.
// Truth remains in the sealed world/observe/compare values, not this wrapper.
type Trial struct {
	Role                reference.CandidateRole
	Slot                observe.ScheduledTrial
	Admitted            bool
	Result              world.Result
	Measurements        domain.InstanceMeasurements
	Observation         counterhttp.HTTPCapturedObservation
	Projection          counterhttp.HTTPProjectionResult
	Projected           bool
	ProjectionRejection *counterhttp.ProjectionRejection
}

// Result exposes the exact deterministic and fresh core authorities needed by
// evidence publication and later U7D reproduction.
type Result struct {
	Root                 string
	SourceSpecDigest     domain.Digest
	SourceSpecBytes      []byte
	Plan                 domain.WorldPlan
	Envelope             domain.ComparisonEnvelope
	Stimulus             counterhttp.HTTPStimulus
	Binding              counterhttp.HTTPExecutionBinding
	StartSpec            counterhttp.HTTPStartSpec
	CapturePolicy        counterhttp.HTTPCapturePolicy
	Readiness            counterhttp.HTTPReadinessContract
	ProjectionDefinition counterhttp.HTTPProjectionDefinition
	Observation          observe.ObservationRun
	OutcomeMap           compare.CandidateOutcomeMap
	HasOutcomeMap        bool
	CandidateBindings    []domain.CandidateExecutionBinding
	CandidateRoles       map[domain.CandidateExecutionKey]reference.CandidateRole
	Trials               []Trial
	Confirmation         confirmation.Completed
	HasConfirmation      bool
}

// Handler is the domain-neutral app seam. It is intentionally a thin wrapper:
// app owns command framing, while this package owns the typed HTTP workflow.
func Handler() app.StudyHandler {
	return func(request app.StudyRequest) error {
		completed, err := CompleteHTTPStudy(request.Context, Config{
			RepositoryRoot: request.WorkingDirectory,
			ScratchRoot:    request.ScratchRoot,
			EvidenceRoot:   request.EvidenceRoot,
			GitExecutable:  request.GitExecutable,
			NodeExecutable: request.NodeExecutable,
			Ordinal:        request.Ordinal,
			runStandalone: func(ctx context.Context, input standaloneRunInput) (standaloneRunObservation, error) {
				observed, runErr := request.RunNodeTest(ctx, app.StudyNodeTestInput{
					WorkingDirectory:   input.workingDirectory,
					TestFile:           input.testFile,
					HomeDirectory:      input.homeDirectory,
					TemporaryDirectory: input.temporaryDirectory,
					Timeout:            input.timeout,
				})
				if runErr != nil {
					return standaloneRunObservation{}, runErr
				}
				return standaloneRunObservation{
					stdout:           append([]byte(nil), observed.Stdout...),
					stderr:           append([]byte(nil), observed.Stderr...),
					invocationSHA256: observed.InvocationSHA256,
					testFileSHA256:   observed.TestFileSHA256,
					exitCode:         observed.ExitCode,
				}, nil
			},
		})
		if err != nil {
			return err
		}
		if err := request.Context.Err(); err != nil {
			return fmt.Errorf("HTTP_STUDY_CONTEXT_ENDED_BEFORE_PUBLICATION: %w", err)
		}
		_, err = publishCompletedEvidence(completed, request.EvidenceRoot)
		return err
	}
}

// Run executes one complete fresh discovery batch directly against the exact
// driver-prepared fixture repository.
func Run(ctx context.Context, config Config) (result Result, returnErr error) {
	return runPhase(ctx, config, phaseRunSpec{
		purpose: domain.AttemptDiscovery, repetitions: discoveryRepetitions,
	})
}

func runPhase(ctx context.Context, config Config, phase phaseRunSpec) (result Result, returnErr error) {
	if phase.planDiscoveryRepeats == 0 {
		phase.planDiscoveryRepeats = phase.repetitions
	}
	if phase.planConfirmationRepeats == 0 {
		phase.planConfirmationRepeats = phase.repetitions
	}
	expectedRepeats := phase.planDiscoveryRepeats
	if phase.purpose == domain.AttemptConfirmation {
		expectedRepeats = phase.planConfirmationRepeats
	}
	if ctx == nil || config.Ordinal < 1 || config.Ordinal > 3 ||
		!cleanAbsolute(config.RepositoryRoot) || !cleanAbsolute(config.ScratchRoot) ||
		!cleanAbsolute(config.EvidenceRoot) || !cleanAbsolute(config.GitExecutable) ||
		!cleanAbsolute(config.NodeExecutable) || !phase.purpose.Valid() ||
		phase.repetitions < 1 || phase.repetitions > 5 || phase.repetitions != expectedRepeats ||
		phase.planDiscoveryRepeats < 1 || phase.planDiscoveryRepeats > 5 ||
		phase.planConfirmationRepeats < 1 || phase.planConfirmationRepeats > 5 || !validRunLabel(phase.label) {
		return Result{}, fmt.Errorf("HTTP_STUDY_CONFIG_REFUSED")
	}
	if phase.confirmation != nil && (phase.purpose != domain.AttemptConfirmation ||
		!phase.confirmation.reducedBaseline.Valid() || !phase.confirmation.reductionRun.Valid() ||
		!phase.confirmation.reductionResult.Valid()) {
		return Result{}, fmt.Errorf("HTTP_STUDY_CONFIRMATION_CONFIG_REFUSED")
	}
	runRoot, err := privateRunRoot(config.ScratchRoot, config.Ordinal, phase.label)
	if err != nil {
		return Result{}, err
	}
	gitScratch, err := privateDirectory(runRoot, "git-scratch")
	if err != nil {
		return Result{}, err
	}
	fixture, err := reference.OpenHTTPFixture(ctx, reference.HTTPFixtureConfig{
		Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
	})
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if closeErr := fixture.Close(); returnErr == nil && closeErr != nil {
			returnErr = closeErr
		}
	}()
	inspectionRoot, err := privateDirectory(runRoot, "inspection")
	if err != nil {
		return Result{}, err
	}
	allocationRoot, err := privateDirectory(runRoot, "attempts")
	if err != nil {
		return Result{}, err
	}
	toolScratch, err := privateDirectory(runRoot, "tools")
	if err != nil {
		return Result{}, err
	}

	repository := fixture.Repository()
	if !repository.Valid() {
		return Result{}, fmt.Errorf("HTTP_STUDY_REPOSITORY_AUTHORITY_REFUSED")
	}

	roles := fixture.Roles()
	pins := make([]gitobj.PinnedTree, 0, len(roles))
	roleByTreeIdentity := make(map[domain.Digest]reference.CandidateRole, len(roles))
	for _, role := range roles {
		ref, refErr := fixture.Ref(role)
		if refErr != nil {
			return Result{}, refErr
		}
		pinned, pinErr := repository.Pin(ctx, ref)
		if pinErr != nil {
			return Result{}, pinErr
		}
		pins = append(pins, pinned)
		roleByTreeIdentity[pinned.IdentityDigest()] = role
	}
	selected, err := gitobj.SelectTrees(pins...)
	if err != nil {
		return Result{}, err
	}
	materializationPolicy, err := gitobj.NewPolicy(16, 1<<20, 1<<19)
	if err != nil {
		return Result{}, err
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, materializationPolicy, inspectionRoot)
	if err != nil {
		return Result{}, err
	}

	var stimulus counterhttp.HTTPStimulus
	if phase.stimulus == nil {
		stimulus, err = invoiceStimulus(reference.HTTPSeedJSON())
	} else {
		stimulus = *phase.stimulus
	}
	if err != nil || !stimulus.Valid() {
		return Result{}, errors.Join(err, fmt.Errorf("HTTP_STUDY_STIMULUS_REFUSED"))
	}
	startSpec, err := counterhttp.NewPortableHTTPStartSpec(fixture.Entrypoint())
	if err != nil {
		return Result{}, err
	}
	fixtureRecipe, err := counterhttp.NewHTTPFixtureRecipe()
	if err != nil {
		return Result{}, err
	}
	readiness, err := counterhttp.NewPortableHTTPReadinessContract()
	if err != nil {
		return Result{}, err
	}
	capturePolicy, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024,
		HeaderBytes:     32 << 10,
		HeaderCount:     64,
		BodyBytes:       64 << 10,
	})
	if err != nil {
		return Result{}, err
	}
	projection, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return Result{}, err
	}
	envelope, err := studyEnvelope()
	if err != nil {
		return Result{}, err
	}
	runnerDigest, err := runnerprofile.HTTPPortableDigest()
	if err != nil {
		return Result{}, err
	}
	plan, sourceDigest, sourceBytes, err := compilePlan(planInput{
		CandidateSetDigest:          declaration.Digest(),
		MaterializationPolicyDigest: materializationPolicy.Digest(),
		ComparisonEnvelopeDigest:    envelope.Digest(),
		RunnerDigest:                runnerDigest,
		StartArgv:                   startSpec.LogicalArgv(),
		ReadinessSignal:             readiness.SignalName(),
		FixtureRecipeDigest:         fixtureRecipe.Digest(),
		CapturePolicy:               capturePolicy,
		ProjectionDefinition:        projection.Binding(),
		DiscoveryRepetitions:        phase.planDiscoveryRepeats,
		ConfirmationRepetitions:     phase.planConfirmationRepeats,
		CandidateCount:              len(roles),
	})
	if err != nil {
		return Result{}, err
	}
	binding, err := counterhttp.BindExecution(plan, stimulus, startSpec, capturePolicy, readiness, projection)
	if err != nil {
		return Result{}, err
	}
	boundCandidates, err := declaration.Bind(plan)
	if err != nil {
		return Result{}, err
	}
	candidateByKey := make(map[domain.CandidateExecutionKey]gitobj.BoundCandidate, len(boundCandidates))
	roleByKey := make(map[domain.CandidateExecutionKey]reference.CandidateRole, len(boundCandidates))
	for _, candidate := range boundCandidates {
		key := candidate.Binding().Key()
		role, present := roleByTreeIdentity[candidate.TreeIdentityDigest()]
		if !present {
			return Result{}, fmt.Errorf("HTTP_STUDY_CANDIDATE_AUTHORITY_REFUSED")
		}
		candidateByKey[key] = candidate
		roleByKey[key] = role
	}
	tools, err := world.NewToolRegistry(ctx, toolScratch, world.ToolSpec{
		Name:              "node",
		AbsolutePath:      config.NodeExecutable,
		VersionConstraint: "executed-major-only",
		VersionArgs:       []string{"--version"},
	})
	if err != nil {
		return Result{}, err
	}
	roster := make([]domain.CandidateExecutionKey, 0, len(boundCandidates))
	bindings := make([]domain.CandidateExecutionBinding, 0, len(boundCandidates))
	for _, candidate := range boundCandidates {
		roster = append(roster, candidate.Binding().Key())
		bindings = append(bindings, candidate.Binding())
	}
	trialBudget := len(roster) * phase.repetitions
	trials := make([]Trial, 0, trialBudget)
	executeTrial := func(trialContext context.Context, slot observe.ScheduledTrial, nonce string) (world.Result, observe.PreparedTrial, error) {
		candidate, present := candidateByKey[slot.CandidateKey()]
		if !present {
			return world.Result{}, observe.PreparedTrial{}, fmt.Errorf("HTTP_STUDY_SCHEDULE_REFUSED")
		}
		worldResult, executeErr := world.ExecuteHTTP(trialContext, world.HTTPRequest{
			Binding:         binding,
			Candidate:       candidate,
			Tools:           tools,
			AllocationRoot:  allocationRoot,
			Purpose:         phase.purpose,
			InstanceNonce:   nonce,
			ScheduleOrdinal: slot.Ordinal(),
		})
		if executeErr != nil {
			return world.Result{}, observe.PreparedTrial{}, executeErr
		}
		measurements, measurementErr := measurementsFor(envelope, binding, worldResult)
		if measurementErr != nil {
			return world.Result{}, observe.PreparedTrial{}, measurementErr
		}
		observation, captureErr := counterhttp.AdaptWorldResult(worldResult, binding, slot.Ordinal())
		if captureErr != nil {
			return world.Result{}, observe.PreparedTrial{}, captureErr
		}
		evidence := Trial{
			Role:         roleByKey[slot.CandidateKey()],
			Slot:         slot,
			Result:       worldResult,
			Measurements: measurements,
			Observation:  observation,
		}
		if worldResult.FinalizedAttempt().HasControls() {
			prepared, prepareErr := observe.NewPreparedControlledTrial(slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements)
			if prepareErr == nil {
				trials = append(trials, evidence)
			}
			return worldResult, prepared, prepareErr
		}
		projected, projectErr := projection.Project(observation)
		if projectErr != nil {
			var rejection *counterhttp.ProjectionRejection
			if !errors.As(projectErr, &rejection) {
				return world.Result{}, observe.PreparedTrial{}, projectErr
			}
			rejectionEvidence, bridgeErr := counterhttp.PrepareProjectionRejectionEvidence(observation, rejection)
			if bridgeErr != nil {
				return world.Result{}, observe.PreparedTrial{}, bridgeErr
			}
			prepared, prepareErr := observe.NewPreparedProjectionRejectedTrial(slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, rejectionEvidence)
			if prepareErr == nil {
				evidence.ProjectionRejection = rejection
				trials = append(trials, evidence)
			}
			return worldResult, prepared, prepareErr
		}
		structural, bridgeErr := counterhttp.PrepareStructuralCapture(worldResult.World(), worldResult.FinalizedAttempt(), observation, projected)
		if bridgeErr != nil {
			return world.Result{}, observe.PreparedTrial{}, bridgeErr
		}
		prepared, prepareErr := observe.NewPreparedCapturedTrial(slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, structural)
		if prepareErr == nil {
			evidence.Projected = true
			evidence.Projection = projected
			trials = append(trials, evidence)
		}
		return worldResult, prepared, prepareErr
	}
	var observation observe.ObservationRun
	var completed confirmation.Completed
	if phase.confirmation == nil {
		observation, err = observe.RunObservation(ctx, observe.ObservationConfig{
			Plan: plan, Purpose: phase.purpose, Envelope: envelope,
			CandidateRoster: roster, Repetitions: phase.repetitions,
			Budget: observe.TrialBudget{MaxTotalTrials: trialBudget, WallBudget: discoveryWallBudget},
		}, func(trialContext context.Context, slot observe.ScheduledTrial) (observe.PreparedTrial, error) {
			nonce := fmt.Sprintf("u7-http-%d-%s-%d-%d", config.Ordinal, runLabelForNonce(phase.label), slot.Repetition(), slot.Ordinal())
			_, prepared, trialErr := executeTrial(trialContext, slot, nonce)
			return prepared, trialErr
		})
		if err != nil {
			return Result{}, err
		}
	} else {
		completed, err = confirmation.Run(ctx, confirmation.Request{
			Plan: plan, Envelope: envelope,
			ReducedBaseline: phase.confirmation.reducedBaseline,
			ReductionRun:    phase.confirmation.reductionRun, ReductionResult: phase.confirmation.reductionResult,
			WallBudget: discoveryWallBudget,
			Execute: func(trialContext context.Context, request confirmation.TrialRequest) (world.Result, observe.PreparedTrial, error) {
				return executeTrial(trialContext, request.Slot(), request.InstanceNonce())
			},
		})
		if err != nil {
			return Result{}, err
		}
	}
	admitted := make(map[domain.Digest]struct{})
	if phase.confirmation == nil {
		for _, batch := range observation.Batches() {
			for _, digest := range batch.AttemptDigests() {
				admitted[digest] = struct{}{}
			}
		}
	} else {
		for _, digest := range completed.ConfirmedOutcomeMap().OutcomeMap().EvidenceAttemptDigests() {
			admitted[digest] = struct{}{}
		}
	}
	for index := range trials {
		_, trials[index].Admitted = admitted[trials[index].Result.FinalizedAttempt().ArtifactDigest()]
	}
	result = Result{
		Root:                 runRoot,
		SourceSpecDigest:     sourceDigest,
		SourceSpecBytes:      append([]byte(nil), sourceBytes...),
		Plan:                 plan,
		Envelope:             envelope,
		Stimulus:             stimulus,
		Binding:              binding,
		StartSpec:            startSpec,
		CapturePolicy:        capturePolicy,
		Readiness:            readiness,
		ProjectionDefinition: projection,
		Observation:          observation,
		CandidateBindings:    append([]domain.CandidateExecutionBinding(nil), bindings...),
		CandidateRoles:       cloneRoles(roleByKey),
		Trials:               append([]Trial(nil), trials...),
		Confirmation:         completed,
		HasConfirmation:      phase.confirmation != nil && completed.Valid(),
	}
	if phase.confirmation != nil {
		result.OutcomeMap = completed.ConfirmedOutcomeMap().OutcomeMap()
		result.HasOutcomeMap = completed.Valid()
	} else if input, present := observation.OutcomeMapInput(); present {
		outcome, outcomeErr := compare.NewCandidateOutcomeMap(input.StimulusDigest(), input.EnvelopeDigest(), input.Roster(), input.Batches())
		if outcomeErr != nil {
			return Result{}, outcomeErr
		}
		result.OutcomeMap = outcome
		result.HasOutcomeMap = true
	}
	return result, nil
}

func cleanAbsolute(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path
}

func privateRunRoot(parent string, ordinal int, label string) (string, error) {
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("HTTP_STUDY_SCRATCH_REFUSED")
	}
	name := fmt.Sprintf("http-study-%d", ordinal)
	if label != "" {
		name += "-" + label
	}
	return privateDirectory(parent, name)
}

func validRunLabel(label string) bool {
	if len(label) > 48 {
		return false
	}
	for index, character := range label {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') &&
			(character != '-' || index == 0 || index == len(label)-1) {
			return false
		}
	}
	return true
}

func runLabelForNonce(label string) string {
	if label == "" {
		return "discovery"
	}
	return label
}

func privateDirectory(parent, name string) (string, error) {
	path := filepath.Join(parent, name)
	if filepath.Dir(path) != parent {
		return "", fmt.Errorf("HTTP_STUDY_PATH_REFUSED")
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("HTTP_STUDY_PRIVATE_DIRECTORY_REFUSED")
	}
	return path, nil
}

func invoiceStimulus(seedBytes []byte) (counterhttp.HTTPStimulus, error) {
	query := make([]counterhttp.HTTPQueryEntry, 0, 5)
	for _, input := range [][2]string{{"actor_tenant", "tenant-a"}, {"role", "support"}, {"tag", "first"}, {"tag", "second"}} {
		entry, err := counterhttp.QueryValue(input[0], input[1])
		if err != nil {
			return counterhttp.HTTPStimulus{}, err
		}
		query = append(query, entry)
	}
	flag, err := counterhttp.QueryFlag("audit")
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	query = append(query, flag)
	headers := make([]counterhttp.HTTPRequestHeader, 0, 9)
	for _, input := range [][2]string{
		{"accept", "application/json"}, {"x-countershape-tenant", "tenant-a"}, {"x-countershape-role", "support"},
		{"x-countershape-audit", "required"}, {"x-countershape-tag", "first"}, {"x-countershape-tag", "second"},
		{"x-countershape-trace", "first"}, {"x-countershape-trace", "second"}, {"x-countershape-present-empty", ""},
	} {
		header, headerErr := counterhttp.NewRequestHeader(input[0], input[1])
		if headerErr != nil {
			return counterhttp.HTTPStimulus{}, headerErr
		}
		headers = append(headers, header)
	}
	seed, err := counterhttp.NewSeedFile(reference.HTTPSeedFilename, seedBytes, counterhttp.SeedMode0644)
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	return counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method:  counterhttp.MethodGET,
		Path:    "/v1/invoices/inv-204",
		Query:   query,
		Headers: headers,
		Body:    counterhttp.AbsentBody(),
		Seeds:   []counterhttp.HTTPSeedFile{seed},
	})
}

type planInput struct {
	CandidateSetDigest          domain.Digest
	MaterializationPolicyDigest domain.Digest
	ComparisonEnvelopeDigest    domain.Digest
	RunnerDigest                domain.Digest
	StartArgv                   []string
	ReadinessSignal             string
	FixtureRecipeDigest         domain.Digest
	CapturePolicy               counterhttp.HTTPCapturePolicy
	ProjectionDefinition        domain.ProjectionDefinitionBinding
	DiscoveryRepetitions        int
	ConfirmationRepetitions     int
	CandidateCount              int
}

type sourceAdapter struct {
	Domain         string `json:"domain"`
	AdapterVersion string `json:"adapter_version"`
	RunnerDigest   string `json:"runner_digest"`
}

type sourceReadiness struct {
	Kind       string `json:"kind"`
	SignalName string `json:"signal_name"`
}

type sourceSpec struct {
	SchemaVersion               string                    `json:"schema_version"`
	Kind                        string                    `json:"kind"`
	CandidateSetDigest          string                    `json:"candidate_set_digest"`
	MaterializationPolicyDigest string                    `json:"materialization_policy_digest"`
	ComparisonEnvelopeDigest    string                    `json:"comparison_envelope_digest"`
	Adapter                     sourceAdapter             `json:"adapter"`
	ExecutionShape              string                    `json:"execution_shape"`
	StartArgv                   []string                  `json:"start_argv"`
	SetupArgv                   []string                  `json:"setup_argv"`
	Environment                 []domain.EnvironmentEntry `json:"environment"`
	SecretSlots                 []domain.SecretSlot       `json:"secret_slots"`
	FixtureRecipeDigest         string                    `json:"fixture_recipe_digest"`
	Readiness                   sourceReadiness           `json:"readiness"`
	CapturePolicyDigest         string                    `json:"capture_policy_digest"`
	ProjectionDefinitionDigest  string                    `json:"projection_definition_digest"`
	RepeatSchedule              domain.RepeatSchedule     `json:"repeat_schedule"`
	RequiredTools               []domain.RequiredTool     `json:"required_tools"`
	Budgets                     domain.Budgets            `json:"budgets"`
}

func compilePlan(input planInput) (domain.WorldPlan, domain.Digest, []byte, error) {
	budgets := domain.Budgets{
		CandidateCount:            input.CandidateCount,
		MaterializedEntryCount:    16,
		MaterializedBytesPerWorld: 1 << 20,
		SingleBlobBytes:           1 << 19,
		ReadinessMS:               1500,
		ProbeMS:                   2000,
		TeardownMS:                1000,
		StdoutBytes:               64 << 10,
		StderrBytes:               64 << 10,
		HTTPBodyBytes:             input.CapturePolicy.BodyBytes(),
		ProposedShrinkStimuli:     2,
		TotalCandidateTrials:      36,
		ShrinkWallMS:              (3 * time.Minute).Milliseconds(),
	}
	identity := sourceSpec{
		SchemaVersion:               "countershape-source/v1",
		Kind:                        "SourceSpec",
		CandidateSetDigest:          input.CandidateSetDigest.String(),
		MaterializationPolicyDigest: input.MaterializationPolicyDigest.String(),
		ComparisonEnvelopeDigest:    input.ComparisonEnvelopeDigest.String(),
		Adapter: sourceAdapter{
			Domain: string(domain.AdapterHTTP), AdapterVersion: "http/v1", RunnerDigest: input.RunnerDigest.String(),
		},
		ExecutionShape: string(domain.OneLoopbackHTTPRequest),
		StartArgv:      append([]string(nil), input.StartArgv...),
		SetupArgv:      []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"}, {Name: "LC_ALL", Value: "C"}, {Name: "NODE_NO_WARNINGS", Value: "1"},
			{Name: "NO_COLOR", Value: "1"}, {Name: "TZ", Value: "UTC"},
		},
		SecretSlots:                []domain.SecretSlot{},
		FixtureRecipeDigest:        input.FixtureRecipeDigest.String(),
		Readiness:                  sourceReadiness{Kind: string(domain.FixtureOwnedReadiness), SignalName: input.ReadinessSignal},
		CapturePolicyDigest:        input.CapturePolicy.Digest().String(),
		ProjectionDefinitionDigest: input.ProjectionDefinition.Digest().String(),
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: input.DiscoveryRepetitions, ConfirmationRepeats: input.ConfirmationRepetitions,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets:       budgets,
	}
	sourceBytes, err := canon.CanonicalizeTyped(identity)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	parsed, err := corespec.ParseSource(sourceBytes)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	plan, err := corespec.Compile(parsed, input.ProjectionDefinition)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	return plan, parsed.Digest(), parsed.CanonicalBytes(), nil
}

func cloneRoles(input map[domain.CandidateExecutionKey]reference.CandidateRole) map[domain.CandidateExecutionKey]reference.CandidateRole {
	result := make(map[domain.CandidateExecutionKey]reference.CandidateRole, len(input))
	for key, role := range input {
		result[key] = role
	}
	return result
}

// CanonicalCandidateRoster returns a defensive, byte-sorted opaque roster.
func (result Result) CanonicalCandidateRoster() []domain.CandidateExecutionKey {
	roster := make([]domain.CandidateExecutionKey, 0, len(result.CandidateRoles))
	for key := range result.CandidateRoles {
		roster = append(roster, key)
	}
	sort.Slice(roster, func(left, right int) bool { return roster[left].String() < roster[right].String() })
	return roster
}

const (
	httpCWDPolicy               = "MATERIALIZED_ROOT"
	dimensionToolDigest         = "tool executable digest"
	dimensionToolMajor          = "tool major version"
	dimensionExecutionAuthority = "HTTP execution authority"
	dimensionLogicalArgv        = "logical argv"
	dimensionCWDPolicy          = "working-directory policy"
	dimensionReadinessProtocol  = "readiness protocol"
	dimensionWorldDigest        = "world instance digest"
	dimensionAttemptDigest      = "attempt artifact digest"
	dimensionScheduleOrdinal    = "schedule ordinal"
	dimensionPID                = "process id"
	dimensionPhysicalExecution  = "physical execution entered"
	dimensionWorkingDirectory   = "working directory"
	dimensionFixtureRoot        = "fixture root"
	dimensionStateRoot          = "scratch state root"
	dimensionSeedOverlay        = "HTTP seed overlay receipt"
	dimensionEndpoint           = "allocated loopback endpoint"
	dimensionPort               = "allocated loopback port"
	dimensionReadinessReceipt   = "readiness receipt"
	dimensionExchangeReceipt    = "HTTP exchange receipt"
	dimensionInvocationEvidence = "HTTP invocation evidence receipt"
)

func studyEnvelope() (domain.ComparisonEnvelope, error) {
	required := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt}, {dimensionToolMajor, domain.MeasuredToolReceipt},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt}, {dimensionLogicalArgv, domain.MeasuredProcessReceipt},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt}, {dimensionReadinessProtocol, domain.MeasuredProcessReceipt},
	}
	recorded := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionWorldDigest, domain.MeasuredWorldInstance}, {dimensionAttemptDigest, domain.MeasuredWorldInstance},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance}, {dimensionPID, domain.MeasuredProcessReceipt},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt}, {dimensionWorkingDirectory, domain.MeasuredProcessReceipt},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe}, {dimensionStateRoot, domain.MeasuredFilesystemProbe},
		{dimensionSeedOverlay, domain.MeasuredFilesystemProbe}, {dimensionEndpoint, domain.MeasuredProcessReceipt},
		{dimensionPort, domain.MeasuredProcessReceipt}, {dimensionReadinessReceipt, domain.MeasuredProcessReceipt},
		{dimensionExchangeReceipt, domain.MeasuredProcessReceipt}, {dimensionInvocationEvidence, domain.MeasuredProcessReceipt},
	}
	measured := make([]domain.MeasuredDimension, 0, len(required)+len(recorded))
	requiredEqual := make([]domain.RequiredEqualDimension, 0, len(required))
	tolerated := make([]domain.ToleratedDimension, 0, len(recorded))
	for _, input := range required {
		measured = append(measured, domain.MeasuredDimension{Name: input.name, Source: input.source, Comparison: domain.CompareExact})
		requiredEqual = append(requiredEqual, domain.RequiredEqualDimension{Name: input.name})
	}
	for _, input := range recorded {
		measured = append(measured, domain.MeasuredDimension{Name: input.name, Source: input.source, Comparison: domain.CompareRecordedOnly})
		tolerance := domain.MayDifferRecorded
		if input.name == dimensionStateRoot {
			tolerance = domain.ProjectedCapturePlaceholder
		}
		tolerated = append(tolerated, domain.ToleratedDimension{Name: input.name, Tolerance: tolerance})
	}
	return domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version:       "u7-http-study/darwin-v1",
		Measured:      measured,
		RequiredEqual: requiredEqual,
		Tolerated:     tolerated,
		Rejected:      []domain.RejectedDimension{},
		Uncontrolled:  []string{"host network availability", "kernel scheduling and wall-clock timing", "trusted candidate host file reads"},
	})
}

func measurementsFor(envelope domain.ComparisonEnvelope, binding counterhttp.HTTPExecutionBinding, result world.Result) (domain.InstanceMeasurements, error) {
	if !binding.Valid() || binding.PlanDigest() != result.World().PlanDigest() {
		return domain.InstanceMeasurements{}, fmt.Errorf("HTTP_STUDY_MEASUREMENT_AUTHORITY_REFUSED")
	}
	process := result.Process()
	materialization := result.Materialization()
	if !materialization.Valid() || materialization.PublishedRoot == "" {
		return domain.InstanceMeasurements{}, fmt.Errorf("HTTP_STUDY_MATERIALIZATION_REFUSED")
	}
	seed, hasSeed := result.HTTPSeedOverlay()
	readiness, hasReadiness := result.HTTPReadiness()
	exchange, hasExchange := result.HTTPExchange()
	invocation, hasInvocation := result.HTTPInvocationEvidence()
	readinessProtocol, endpoint, readinessStatus := "", "", "NOT_APPLIED"
	port := int64(0)
	if hasReadiness {
		readinessProtocol, endpoint, port = readiness.Protocol(), readiness.Endpoint(), int64(readiness.Port())
		readinessStatus = fmt.Sprintf("accepted=%t;diagnostic=%s", readiness.Accepted(), readiness.DiagnosticCode())
	}
	exchangeStatus := "NOT_APPLIED"
	if hasExchange {
		exchangeStatus = fmt.Sprintf("parsed=%t;diagnostic=%s", exchange.ResponseParsed(), exchange.DiagnosticCode())
	}
	invocationStatus := "NOT_INSPECTED"
	if hasInvocation {
		invocationStatus = string(invocation.Status())
	}
	seedValue, err := optionalReceipt(hasSeed, seed.Digest(), "PRIVATE_SEED_OVERLAY")
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	readinessValue, err := optionalReceipt(hasReadiness, readiness.Digest(), readinessStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	exchangeValue, err := optionalReceipt(hasExchange, exchange.Digest(), exchangeStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	invocationValue, err := optionalReceipt(hasInvocation, invocation.Digest(), invocationStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	logicalArgv, err := canonicalStringList(process.LogicalArgv())
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	physicalExecution, err := canon.Object(
		canon.Member{Name: "spawn_attempted", Value: canon.Bool(process.SpawnAttempted())},
		canon.Member{Name: "started", Value: canon.Bool(process.Started())},
	)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	values := []struct {
		name   string
		source domain.MeasuredSource
		value  any
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt, process.ToolExecutableDigest().String()},
		{dimensionToolMajor, domain.MeasuredToolReceipt, int64(process.ToolMajor())},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, binding.Authority()},
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt, logicalArgv},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt, httpCWDPolicy},
		{dimensionReadinessProtocol, domain.MeasuredProcessReceipt, readinessProtocol},
		{dimensionWorldDigest, domain.MeasuredWorldInstance, result.World().Digest().String()},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance, result.FinalizedAttempt().ArtifactDigest().String()},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance, int64(result.World().ScheduleOrdinal())},
		{dimensionPID, domain.MeasuredProcessReceipt, int64(process.PID())},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt, physicalExecution},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt, materialization.PublishedRoot},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe, result.Roots().Fixture()},
		{dimensionStateRoot, domain.MeasuredFilesystemProbe, result.Roots().State()},
		{dimensionSeedOverlay, domain.MeasuredFilesystemProbe, seedValue},
		{dimensionEndpoint, domain.MeasuredProcessReceipt, endpoint}, {dimensionPort, domain.MeasuredProcessReceipt, port},
		{dimensionReadinessReceipt, domain.MeasuredProcessReceipt, readinessValue},
		{dimensionExchangeReceipt, domain.MeasuredProcessReceipt, exchangeValue},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt, invocationValue},
	}
	measurements := make([]domain.MeasurementValue, 0, len(values))
	for _, input := range values {
		var value canon.Value
		switch typed := input.value.(type) {
		case string:
			value, err = canon.String(typed)
		case int64:
			value, err = canon.Integer(typed)
		case canon.Value:
			value = typed
		default:
			err = fmt.Errorf("HTTP_STUDY_MEASUREMENT_TYPE_REFUSED")
		}
		if err != nil {
			return domain.InstanceMeasurements{}, err
		}
		measurements = append(measurements, domain.MeasurementValue{Name: input.name, Source: input.source, Value: value})
	}
	return domain.NewInstanceMeasurements(envelope, result.World(), measurements)
}

func optionalReceipt(present bool, digest domain.Digest, status string) (canon.Value, error) {
	presence, digestText := "ABSENT", ""
	if present {
		if !digest.Valid() {
			return canon.Value{}, fmt.Errorf("HTTP_STUDY_RECEIPT_REFUSED")
		}
		presence, digestText = "PRESENT", digest.String()
	} else if digest.Valid() {
		return canon.Value{}, fmt.Errorf("HTTP_STUDY_RECEIPT_REFUSED")
	}
	presenceValue, err := canon.String(presence)
	if err != nil {
		return canon.Value{}, err
	}
	digestValue, err := canon.String(digestText)
	if err != nil {
		return canon.Value{}, err
	}
	statusValue, err := canon.String(status)
	if err != nil {
		return canon.Value{}, err
	}
	return canon.Object(
		canon.Member{Name: "digest", Value: digestValue},
		canon.Member{Name: "presence", Value: presenceValue},
		canon.Member{Name: "status", Value: statusValue},
	)
}

func canonicalStringList(input []string) (canon.Value, error) {
	values := make([]canon.Value, len(input))
	for index, item := range input {
		value, err := canon.String(item)
		if err != nil {
			return canon.Value{}, err
		}
		values[index] = value
	}
	return canon.Array(values...)
}
