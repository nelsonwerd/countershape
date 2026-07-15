// Package cli_precedence is the physical U3 reference study. It assembles
// deterministic Git candidates, executes every scheduled trial in a fresh U2
// world, and crosses into generic comparison only through typed U3 evidence.
package cli_precedence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

type Behavior string

const (
	BehaviorPrecedence          Behavior = "precedence"
	BehaviorAlternating         Behavior = "alternating"
	BehaviorTimeout             Behavior = "timeout"
	BehaviorOutputLimit         Behavior = "output-limit"
	BehaviorSignal              Behavior = "signal"
	BehaviorMalformedProjection Behavior = "malformed-projection"
	BehaviorMissingInvocation   Behavior = "missing-invocation"
	BehaviorMalformedInvocation Behavior = "malformed-invocation"
	BehaviorEmptyOutput         Behavior = "empty-output"
	BehaviorNonzeroExit         Behavior = "nonzero-exit"
	BehaviorMaterializeFailure  Behavior = "materialization-failure"
)

func (b Behavior) valid() bool {
	switch b {
	case BehaviorPrecedence, BehaviorAlternating, BehaviorTimeout,
		BehaviorOutputLimit, BehaviorSignal, BehaviorMalformedProjection,
		BehaviorMissingInvocation, BehaviorMalformedInvocation,
		BehaviorEmptyOutput, BehaviorNonzeroExit, BehaviorMaterializeFailure:
		return true
	default:
		return false
	}
}

// Config carries only test-harness capabilities and display metadata. Labels
// and producer metadata are deliberately retained in StudyResult but never
// passed into plan, schedule, projection, classification, or map identity.
type Config struct {
	Root             string
	GitExecutable    string
	NodeExecutable   string
	Repetitions      int
	MaxTotalTrials   int
	WallBudget       time.Duration
	Behavior         Behavior
	ProjectionFields []cli.CLIFieldID
	CandidateOrder   []clifixture.CandidateRole
	DisplayLabels    map[clifixture.CandidateRole]string
	ProducerMetadata string
	StdinPresent     bool
	StdinBytes       []byte
}

type TrialEvidence struct {
	Role                clifixture.CandidateRole
	Slot                observe.ScheduledTrial
	Admitted            bool
	Result              world.Result
	Measurements        domain.InstanceMeasurements
	Observation         cli.CLICapturedObservation
	Projection          cli.CLIProjectionResult
	Projected           bool
	ProjectionRejection *cli.ProjectionRejection
}

type StudyResult struct {
	Root                 string
	SourceSpecDigest     domain.Digest
	SourceSpecBytes      []byte
	Plan                 domain.WorldPlan
	Envelope             domain.ComparisonEnvelope
	Stimulus             cli.CLIStimulus
	Binding              cli.CLIExecutionBinding
	CapturePolicy        cli.CLICapturePolicy
	ProjectionDefinition cli.CLIProjectionDefinition
	Observation          observe.ObservationRun
	OutcomeMap           compare.CandidateOutcomeMap
	HasOutcomeMap        bool
	CandidateRoles       map[domain.CandidateExecutionKey]clifixture.CandidateRole
	Trials               []TrialEvidence
	DisplayLabels        map[clifixture.CandidateRole]string
	ProducerMetadata     string
}

func DefaultConfig(root, gitExecutable, nodeExecutable string) Config {
	return Config{
		Root:           root,
		GitExecutable:  gitExecutable,
		NodeExecutable: nodeExecutable,
		Repetitions:    3,
		MaxTotalTrials: 9,
		WallBudget:     30 * time.Second,
		Behavior:       BehaviorPrecedence,
		ProjectionFields: []cli.CLIFieldID{
			cli.CLIFieldStdoutJSONMode,
			cli.CLIFieldStdoutJSONSource,
		},
		CandidateOrder: clifixture.Roles(),
	}
}

func Run(ctx context.Context, config Config) (study StudyResult, returnErr error) {
	if ctx == nil || !config.Behavior.valid() || (!config.StdinPresent && len(config.StdinBytes) != 0) ||
		config.Repetitions < 1 || config.Repetitions > 5 ||
		config.MaxTotalTrials < 1 || config.WallBudget <= 0 {
		return StudyResult{}, fmt.Errorf("invalid CLI precedence study configuration")
	}
	roles, err := normalizeRoles(config.CandidateOrder)
	if err != nil {
		return StudyResult{}, err
	}
	runRoot, err := newPrivateDirectory(config.Root, "cli-precedence-run-")
	if err != nil {
		return StudyResult{}, err
	}
	gitScratch, err := newPrivateDirectory(runRoot, "git-scratch-")
	if err != nil {
		return StudyResult{}, err
	}
	inspectionRoot, err := newPrivateDirectory(runRoot, "inspection-target-")
	if err != nil {
		return StudyResult{}, err
	}
	allocationRoot, err := newPrivateDirectory(runRoot, "attempts-")
	if err != nil {
		return StudyResult{}, err
	}
	toolScratch, err := newPrivateDirectory(runRoot, "tool-scratch-")
	if err != nil {
		return StudyResult{}, err
	}

	fixtureRepository, err := gitrepo.Init(ctx, config.GitExecutable, runRoot, gitrepo.SHA1)
	if err != nil {
		return StudyResult{}, err
	}
	for _, role := range roles {
		files, fileErr := clifixture.CandidateFiles(role)
		if fileErr != nil {
			return StudyResult{}, fileErr
		}
		commit, commitErr := fixtureRepository.CommitFiles(ctx, files, "", "CLI precedence candidate "+string(role))
		if commitErr != nil {
			return StudyResult{}, commitErr
		}
		ref := "refs/heads/" + string(role)
		if updateErr := fixtureRepository.UpdateRef(ctx, ref, commit); updateErr != nil {
			return StudyResult{}, updateErr
		}
	}

	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: config.GitExecutable,
		Repository:    fixtureRepository.Root,
		ScratchRoot:   gitScratch,
	})
	if err != nil {
		return StudyResult{}, err
	}
	defer func() {
		if closeErr := repository.Close(); returnErr == nil && closeErr != nil {
			returnErr = closeErr
		}
	}()
	pins := make([]gitobj.PinnedTree, 0, len(roles))
	roleByTreeIdentity := make(map[domain.Digest]clifixture.CandidateRole, len(roles))
	for _, role := range roles {
		pinned, pinErr := repository.Pin(ctx, "refs/heads/"+string(role))
		if pinErr != nil {
			return StudyResult{}, pinErr
		}
		pins = append(pins, pinned)
		roleByTreeIdentity[pinned.IdentityDigest()] = role
	}
	selected, err := gitobj.SelectTrees(pins...)
	if err != nil {
		return StudyResult{}, err
	}
	materializationPolicy, err := gitobj.NewPolicy(16, 1<<20, 1<<19)
	if err != nil {
		return StudyResult{}, err
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, materializationPolicy, inspectionRoot)
	if err != nil {
		return StudyResult{}, err
	}

	stdin := cli.AbsentStdin()
	if config.StdinPresent {
		stdin, err = cli.PresentStdin(config.StdinBytes)
		if err != nil {
			return StudyResult{}, err
		}
	}
	environment, err := cli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		return StudyResult{}, err
	}
	configFixture, err := cli.NewFixtureFile("config.json", clifixture.ConfigJSON("config"), cli.FixtureMode0644)
	if err != nil {
		return StudyResult{}, err
	}
	stimulus, err := cli.NewCLIStimulus(cli.CLIStimulusConfig{
		Executable:  "node",
		BaseArgv:    []string{clifixture.Entrypoint},
		Argv:        behaviorArgv(config.Behavior),
		Stdin:       stdin,
		Environment: []cli.CLIEnvironmentBinding{environment},
		Fixtures:    []cli.CLIFixtureFile{configFixture},
		CWDPolicy:   cli.CWDMaterializedRoot,
	})
	if err != nil {
		return StudyResult{}, err
	}
	capturePolicy, err := cli.NewCLICapturePolicy(cli.CLICapturePolicyConfig{
		StdoutBytes: 64 << 10,
		StderrBytes: 64 << 10,
	})
	if err != nil {
		return StudyResult{}, err
	}
	fixtureRecipe, err := cli.NewCLIFixtureRecipe()
	if err != nil {
		return StudyResult{}, err
	}
	projection, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{
		Fields: append([]cli.CLIFieldID(nil), config.ProjectionFields...),
	})
	if err != nil {
		return StudyResult{}, err
	}
	envelope, err := newStudyEnvelope()
	if err != nil {
		return StudyResult{}, err
	}
	runnerDigest, err := digestBytes("CLIStudyRunner", []byte("world.ExecuteCLI/U3/opaque-binding/v1"))
	if err != nil {
		return StudyResult{}, err
	}
	probeMS := int64(1500)
	if config.Behavior == BehaviorTimeout {
		probeMS = 150
	}
	plan, sourceDigest, sourceBytes, err := compileStudyPlan(studyPlanInput{
		CandidateSetDigest: declaration.Digest(), MaterializationPolicyDigest: materializationPolicy.Digest(),
		ComparisonEnvelopeDigest: envelope.Digest(), RunnerDigest: runnerDigest,
		StartArgv: stimulus.BaseLogicalArgv(), FixtureRecipeDigest: fixtureRecipe.Digest(),
		CapturePolicy: capturePolicy, ProjectionDefinition: projection.Binding(),
		Repetitions: config.Repetitions, CandidateCount: len(roles), ProbeMS: probeMS,
	})
	if err != nil {
		return StudyResult{}, err
	}
	executionBinding, err := cli.BindExecution(plan, stimulus, capturePolicy, projection)
	if err != nil {
		return StudyResult{}, err
	}
	boundCandidates, err := declaration.Bind(plan)
	if err != nil {
		return StudyResult{}, err
	}
	candidateByKey := make(map[domain.CandidateExecutionKey]gitobj.BoundCandidate, len(boundCandidates))
	roleByKey := make(map[domain.CandidateExecutionKey]clifixture.CandidateRole, len(boundCandidates))
	for _, candidate := range boundCandidates {
		key := candidate.Binding().Key()
		role, present := roleByTreeIdentity[candidate.TreeIdentityDigest()]
		if !present {
			return StudyResult{}, fmt.Errorf("candidate has unknown immutable tree identity %s", candidate.TreeIdentityDigest())
		}
		candidateByKey[key] = candidate
		roleByKey[key] = role
	}
	toolRegistry, err := world.NewToolRegistry(ctx, toolScratch, world.ToolSpec{
		Name:              "node",
		AbsolutePath:      config.NodeExecutable,
		VersionConstraint: "executed-major-only",
		VersionArgs:       []string{"--version"},
	})
	if err != nil {
		return StudyResult{}, err
	}
	if config.Behavior == BehaviorMaterializeFailure {
		if len(boundCandidates) == 0 {
			return StudyResult{}, fmt.Errorf("materialization-failure study has no bound candidate")
		}
		if err := fixtureRepository.RemoveLooseObject(boundCandidates[0].Provenance().TreeOID); err != nil {
			return StudyResult{}, fmt.Errorf("remove admitted loose tree object: %w", err)
		}
	}
	roster := make([]domain.CandidateExecutionKey, 0, len(boundCandidates))
	for _, candidate := range boundCandidates {
		roster = append(roster, candidate.Binding().Key())
	}
	trialEvidence := make([]TrialEvidence, 0, config.Repetitions*len(roster))
	observationRun, err := observe.RunObservation(ctx, observe.ObservationConfig{
		Plan:            plan,
		Purpose:         domain.AttemptDiscovery,
		Envelope:        envelope,
		CandidateRoster: roster,
		Repetitions:     config.Repetitions,
		Budget:          observe.TrialBudget{MaxTotalTrials: config.MaxTotalTrials, WallBudget: config.WallBudget},
	}, func(trialContext context.Context, slot observe.ScheduledTrial) (observe.PreparedTrial, error) {
		candidate, present := candidateByKey[slot.CandidateKey()]
		if !present {
			return observe.PreparedTrial{}, fmt.Errorf("scheduled candidate is outside the opaque roster")
		}
		worldResult, executeErr := world.ExecuteCLI(trialContext, world.CLIRequest{
			Binding:         executionBinding,
			Candidate:       candidate,
			Tools:           toolRegistry,
			AllocationRoot:  allocationRoot,
			Purpose:         domain.AttemptDiscovery,
			InstanceNonce:   fmt.Sprintf("u3-discovery-%d-%d", slot.Repetition(), slot.Ordinal()),
			ScheduleOrdinal: slot.Ordinal(),
		})
		if executeErr != nil {
			return observe.PreparedTrial{}, executeErr
		}
		measurements, measurementErr := studyMeasurements(envelope, worldResult)
		if measurementErr != nil {
			return observe.PreparedTrial{}, measurementErr
		}
		observation, captureErr := cli.AdaptWorldResult(worldResult, executionBinding, slot.Ordinal())
		if captureErr != nil {
			return observe.PreparedTrial{}, captureErr
		}
		evidence := TrialEvidence{
			Role:         roleByKey[slot.CandidateKey()],
			Slot:         slot,
			Result:       worldResult,
			Measurements: measurements,
			Observation:  observation,
		}
		if worldResult.FinalizedAttempt().HasControls() {
			prepared, prepareErr := observe.NewPreparedControlledTrial(
				slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements,
			)
			if prepareErr == nil {
				trialEvidence = append(trialEvidence, evidence)
			}
			return prepared, prepareErr
		}
		projected, projectErr := projection.Project(observation)
		if projectErr != nil {
			var rejection *cli.ProjectionRejection
			if !errors.As(projectErr, &rejection) {
				return observe.PreparedTrial{}, projectErr
			}
			rejectionEvidence, bridgeErr := cli.PrepareProjectionRejectionEvidence(observation, rejection)
			if bridgeErr != nil {
				return observe.PreparedTrial{}, bridgeErr
			}
			prepared, prepareErr := observe.NewPreparedProjectionRejectedTrial(
				slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, rejectionEvidence,
			)
			if prepareErr == nil {
				evidence.ProjectionRejection = rejection
				trialEvidence = append(trialEvidence, evidence)
			}
			return prepared, prepareErr
		}
		structural, bridgeErr := cli.PrepareStructuralCapture(
			worldResult.World(), worldResult.FinalizedAttempt(), observation, projected,
		)
		if bridgeErr != nil {
			return observe.PreparedTrial{}, bridgeErr
		}
		prepared, prepareErr := observe.NewPreparedCapturedTrial(
			slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, structural,
		)
		if prepareErr == nil {
			evidence.Projection = projected
			evidence.Projected = true
			trialEvidence = append(trialEvidence, evidence)
		}
		return prepared, prepareErr
	})
	if err != nil {
		return StudyResult{}, err
	}
	admittedAttempts := make(map[domain.Digest]struct{})
	for _, batch := range observationRun.Batches() {
		for _, digest := range batch.AttemptDigests() {
			admittedAttempts[digest] = struct{}{}
		}
	}
	for index := range trialEvidence {
		_, trialEvidence[index].Admitted = admittedAttempts[trialEvidence[index].Result.FinalizedAttempt().ArtifactDigest()]
	}

	study = StudyResult{
		Root:                 runRoot,
		SourceSpecDigest:     sourceDigest,
		SourceSpecBytes:      append([]byte(nil), sourceBytes...),
		Plan:                 plan,
		Envelope:             envelope,
		Stimulus:             stimulus,
		Binding:              executionBinding,
		CapturePolicy:        capturePolicy,
		ProjectionDefinition: projection,
		Observation:          observationRun,
		CandidateRoles:       cloneRoleMap(roleByKey),
		Trials:               append([]TrialEvidence(nil), trialEvidence...),
		DisplayLabels:        cloneLabels(config.DisplayLabels),
		ProducerMetadata:     config.ProducerMetadata,
	}
	if mapInput, present := observationRun.OutcomeMapInput(); present {
		outcome, outcomeErr := compare.NewCandidateOutcomeMap(
			mapInput.StimulusDigest(), mapInput.EnvelopeDigest(), mapInput.Roster(), mapInput.Batches(),
		)
		if outcomeErr != nil {
			return StudyResult{}, outcomeErr
		}
		study.OutcomeMap = outcome
		study.HasOutcomeMap = true
	}
	return study, nil
}

type studyPlanInput struct {
	CandidateSetDigest          domain.Digest
	MaterializationPolicyDigest domain.Digest
	ComparisonEnvelopeDigest    domain.Digest
	RunnerDigest                domain.Digest
	StartArgv                   []string
	FixtureRecipeDigest         domain.Digest
	CapturePolicy               cli.CLICapturePolicy
	ProjectionDefinition        domain.ProjectionDefinitionBinding
	Repetitions                 int
	CandidateCount              int
	ProbeMS                     int64
}

type studySourceAdapter struct {
	Domain         string `json:"domain"`
	AdapterVersion string `json:"adapter_version"`
	RunnerDigest   string `json:"runner_digest"`
}

type studySourceReadiness struct {
	Kind string `json:"kind"`
}

type studySourceSpec struct {
	SchemaVersion               string                    `json:"schema_version"`
	Kind                        string                    `json:"kind"`
	CandidateSetDigest          string                    `json:"candidate_set_digest"`
	MaterializationPolicyDigest string                    `json:"materialization_policy_digest"`
	ComparisonEnvelopeDigest    string                    `json:"comparison_envelope_digest"`
	Adapter                     studySourceAdapter        `json:"adapter"`
	ExecutionShape              string                    `json:"execution_shape"`
	StartArgv                   []string                  `json:"start_argv"`
	SetupArgv                   []string                  `json:"setup_argv"`
	Environment                 []domain.EnvironmentEntry `json:"environment"`
	SecretSlots                 []domain.SecretSlot       `json:"secret_slots"`
	FixtureRecipeDigest         string                    `json:"fixture_recipe_digest"`
	Readiness                   studySourceReadiness      `json:"readiness"`
	CapturePolicyDigest         string                    `json:"capture_policy_digest"`
	ProjectionDefinitionDigest  string                    `json:"projection_definition_digest"`
	RepeatSchedule              domain.RepeatSchedule     `json:"repeat_schedule"`
	RequiredTools               []domain.RequiredTool     `json:"required_tools"`
	Budgets                     domain.Budgets            `json:"budgets"`
}

func compileStudyPlan(input studyPlanInput) (domain.WorldPlan, domain.Digest, []byte, error) {
	budgets := domain.Budgets{
		CandidateCount:            input.CandidateCount,
		MaterializedEntryCount:    16,
		MaterializedBytesPerWorld: 1 << 20,
		SingleBlobBytes:           1 << 19,
		ReadinessMS:               0,
		ProbeMS:                   input.ProbeMS,
		TeardownMS:                800,
		StdoutBytes:               input.CapturePolicy.StdoutBytes(),
		StderrBytes:               input.CapturePolicy.StderrBytes(),
		HTTPBodyBytes:             64 << 10,
		ProposedShrinkStimuli:     0,
		TotalCandidateTrials:      input.CandidateCount * input.Repetitions * 2,
		ShrinkWallMS:              1000,
	}
	sourceIdentity := studySourceSpec{
		SchemaVersion:               "countershape-source/v1",
		Kind:                        "SourceSpec",
		CandidateSetDigest:          input.CandidateSetDigest.String(),
		MaterializationPolicyDigest: input.MaterializationPolicyDigest.String(),
		ComparisonEnvelopeDigest:    input.ComparisonEnvelopeDigest.String(),
		Adapter: studySourceAdapter{
			Domain: string(domain.AdapterCLI), AdapterVersion: "cli/v1", RunnerDigest: input.RunnerDigest.String(),
		},
		ExecutionShape: string(domain.OneCLIInvocation),
		StartArgv:      append([]string(nil), input.StartArgv...),
		SetupArgv:      []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"},
			{Name: "LC_ALL", Value: "C"},
			{Name: "NODE_NO_WARNINGS", Value: "1"},
			{Name: "NO_COLOR", Value: "1"},
			{Name: "TZ", Value: "UTC"},
		},
		SecretSlots:                []domain.SecretSlot{},
		FixtureRecipeDigest:        input.FixtureRecipeDigest.String(),
		Readiness:                  studySourceReadiness{Kind: string(domain.ReadinessNone)},
		CapturePolicyDigest:        input.CapturePolicy.Digest().String(),
		ProjectionDefinitionDigest: input.ProjectionDefinition.Digest().String(),
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: input.Repetitions, ConfirmationRepeats: input.Repetitions,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets:       budgets,
	}
	sourceBytes, err := canon.CanonicalizeTyped(sourceIdentity)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	// MUTATION_ANCHOR: study-plan-must-compile-strict-source-spec
	parsedSource, err := spec.ParseSource(sourceBytes)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	plan, err := spec.Compile(parsedSource, input.ProjectionDefinition)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	return plan, parsedSource.Digest(), parsedSource.CanonicalBytes(), nil
}

func behaviorArgv(behavior Behavior) []string {
	switch behavior {
	case BehaviorPrecedence, BehaviorMaterializeFailure:
		return []string{"--mode", "argv"}
	case BehaviorNonzeroExit:
		return []string{"--mode", "argv", "--behavior", "nonzero-exit"}
	case BehaviorOutputLimit:
		return []string{"--behavior", "output-limit", "--emit-bytes", "131072"}
	default:
		return []string{"--behavior", string(behavior)}
	}
}

func normalizeRoles(input []clifixture.CandidateRole) ([]clifixture.CandidateRole, error) {
	if len(input) != 3 {
		return nil, fmt.Errorf("CLI precedence study requires exactly three candidate roles")
	}
	seen := make(map[clifixture.CandidateRole]struct{}, 3)
	for _, role := range input {
		if !role.Valid() {
			return nil, fmt.Errorf("invalid CLI precedence role %q", role)
		}
		if _, duplicate := seen[role]; duplicate {
			return nil, fmt.Errorf("duplicate CLI precedence role %q", role)
		}
		seen[role] = struct{}{}
	}
	return append([]clifixture.CandidateRole(nil), input...), nil
}

func newPrivateDirectory(parent, pattern string) (string, error) {
	if !filepath.IsAbs(parent) {
		return "", fmt.Errorf("study root must be absolute")
	}
	canonicalParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	directory, err := os.MkdirTemp(canonicalParent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", err
	}
	if resolved != directory {
		return "", fmt.Errorf("study directory is not canonical")
	}
	return directory, nil
}

func digestBytes(kind string, bytes []byte) (domain.Digest, error) {
	digest, err := canon.DigestBytes(kind, bytes)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func cloneRoleMap(input map[domain.CandidateExecutionKey]clifixture.CandidateRole) map[domain.CandidateExecutionKey]clifixture.CandidateRole {
	result := make(map[domain.CandidateExecutionKey]clifixture.CandidateRole, len(input))
	for key, role := range input {
		result[key] = role
	}
	return result
}

func cloneLabels(input map[clifixture.CandidateRole]string) map[clifixture.CandidateRole]string {
	result := make(map[clifixture.CandidateRole]string, len(input))
	for role, label := range input {
		result[role] = label
	}
	return result
}

const (
	dimensionToolDigest         = "tool executable digest"
	dimensionToolMajor          = "tool major version"
	dimensionExecutionAuthority = "CLI execution authority"
	dimensionLogicalArgv        = "logical argv"
	dimensionPhysicalExecution  = "physical execution entered"
	dimensionCWDPolicy          = "working-directory policy"
	dimensionStdinPresence      = "stdin presence"
	dimensionStdinBytes         = "stdin byte count"
	dimensionWorldDigest        = "world instance digest"
	dimensionAttemptDigest      = "attempt artifact digest"
	dimensionScheduleOrdinal    = "schedule ordinal"
	dimensionPID                = "process id"
	dimensionWorkingDirectory   = "working directory"
	dimensionFixtureRoot        = "fixture root"
	dimensionFixtureOverlay     = "fixture overlay receipt"
	dimensionInvocationEvidence = "invocation evidence receipt"
	dimensionStdoutObserved     = "stdout observed bytes"
	dimensionStderrObserved     = "stderr observed bytes"
)

func newStudyEnvelope() (domain.ComparisonEnvelope, error) {
	required := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt},
		{dimensionToolMajor, domain.MeasuredToolReceipt},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt},
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt},
		{dimensionStdinPresence, domain.MeasuredProcessReceipt},
		{dimensionStdinBytes, domain.MeasuredProcessReceipt},
	}
	recorded := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionWorldDigest, domain.MeasuredWorldInstance},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance},
		{dimensionPID, domain.MeasuredProcessReceipt},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe},
		{dimensionFixtureOverlay, domain.MeasuredFilesystemProbe},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt},
		{dimensionStdoutObserved, domain.MeasuredProcessReceipt},
		{dimensionStderrObserved, domain.MeasuredProcessReceipt},
	}
	measured := make([]domain.MeasuredDimension, 0, len(required)+len(recorded))
	requiredEqual := make([]domain.RequiredEqualDimension, 0, len(required))
	tolerated := make([]domain.ToleratedDimension, 0, len(recorded))
	for _, dimension := range required {
		measured = append(measured, domain.MeasuredDimension{
			Name:       dimension.name,
			Source:     dimension.source,
			Comparison: domain.CompareExact,
		})
		requiredEqual = append(requiredEqual, domain.RequiredEqualDimension{Name: dimension.name})
	}
	for _, dimension := range recorded {
		measured = append(measured, domain.MeasuredDimension{
			Name:       dimension.name,
			Source:     dimension.source,
			Comparison: domain.CompareRecordedOnly,
		})
		tolerated = append(tolerated, domain.ToleratedDimension{
			Name:      dimension.name,
			Tolerance: domain.MayDifferRecorded,
		})
	}
	return domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version:       "cli-precedence-study/v1",
		Measured:      measured,
		RequiredEqual: requiredEqual,
		Tolerated:     tolerated,
		Rejected:      []domain.RejectedDimension{},
		Uncontrolled:  []string{"host network availability", "kernel scheduling and wall-clock timing"},
	})
}

func studyMeasurements(envelope domain.ComparisonEnvelope, result world.Result) (domain.InstanceMeasurements, error) {
	process := result.Process()
	overlay, hasOverlay := result.CLIFixtureOverlay()
	invocation, hasInvocation := result.CLIInvocationEvidence()
	if (hasOverlay && !overlay.Valid()) || (!hasOverlay && process.FixtureOverlayDigest().Valid()) ||
		(hasInvocation && !invocation.Valid()) ||
		(!hasInvocation && (process.InvocationEvidenceDigest().Valid() ||
			process.InvocationEvidenceStatus() != world.CLIInvocationNotInspected ||
			process.InvocationEvidencePresence() != world.CLIInvocationPresenceUnknown)) {
		return domain.InstanceMeasurements{}, fmt.Errorf("strict CLI result has contradictory optional artifact authority")
	}
	overlayMeasurement, err := canonicalOptionalReceipt(
		hasOverlay, overlay.Digest(), "", "",
	)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	invocationMeasurement, err := canonicalOptionalReceipt(
		hasInvocation, invocation.Digest(), string(process.InvocationEvidenceStatus()),
		string(process.InvocationEvidencePresence()),
	)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	logicalArgv, err := canonicalStrings(process.DeclaredLogicalArgv())
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
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, process.ExecutionAuthorityMarker()},
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt, logicalArgv},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt, process.CWDPolicy()},
		{dimensionStdinPresence, domain.MeasuredProcessReceipt, process.StdinPresence()},
		{dimensionStdinBytes, domain.MeasuredProcessReceipt, process.StdinBytes()},
		{dimensionWorldDigest, domain.MeasuredWorldInstance, result.World().Digest().String()},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance, result.FinalizedAttempt().ArtifactDigest().String()},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance, int64(result.World().ScheduleOrdinal())},
		{dimensionPID, domain.MeasuredProcessReceipt, int64(process.PID())},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt, process.PhysicalExecutionEntered()},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt, process.WorkingDirectory()},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe, result.Roots().Fixture()},
		{dimensionFixtureOverlay, domain.MeasuredFilesystemProbe, overlayMeasurement},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt, invocationMeasurement},
		{dimensionStdoutObserved, domain.MeasuredProcessReceipt, process.StdoutObservedBytes()},
		{dimensionStderrObserved, domain.MeasuredProcessReceipt, process.StderrObservedBytes()},
	}
	measurements := make([]domain.MeasurementValue, 0, len(values))
	for _, input := range values {
		var value canon.Value
		switch typed := input.value.(type) {
		case string:
			value, err = canon.String(typed)
		case int64:
			value, err = canon.Integer(typed)
		case bool:
			value = canon.Bool(typed)
		case canon.Value:
			value = typed
		default:
			err = fmt.Errorf("unsupported measurement value for %s", input.name)
		}
		if err != nil {
			return domain.InstanceMeasurements{}, err
		}
		measurements = append(measurements, domain.MeasurementValue{
			Name:   input.name,
			Source: input.source,
			Value:  value,
		})
	}
	return domain.NewInstanceMeasurements(envelope, result.World(), measurements)
}

func canonicalOptionalReceipt(
	present bool,
	digest domain.Digest,
	status, physicalPresence string,
) (canon.Value, error) {
	presence := "ABSENT"
	digestValue := ""
	if present {
		if !digest.Valid() {
			return canon.Value{}, fmt.Errorf("present optional receipt has no digest")
		}
		presence = "PRESENT"
		digestValue = digest.String()
	} else if digest.Valid() {
		return canon.Value{}, fmt.Errorf("absent optional receipt carries a digest")
	}
	presenceValue, err := canon.String(presence)
	if err != nil {
		return canon.Value{}, err
	}
	digestCanonical, err := canon.String(digestValue)
	if err != nil {
		return canon.Value{}, err
	}
	statusValue, err := canon.String(status)
	if err != nil {
		return canon.Value{}, err
	}
	physicalPresenceValue, err := canon.String(physicalPresence)
	if err != nil {
		return canon.Value{}, err
	}
	return canon.Object(
		canon.Member{Name: "digest", Value: digestCanonical},
		canon.Member{Name: "physical_presence", Value: physicalPresenceValue},
		canon.Member{Name: "presence", Value: presenceValue},
		canon.Member{Name: "status", Value: statusValue},
	)
}

func canonicalStrings(input []string) (canon.Value, error) {
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

// RolesByFingerprint is a display-only inspection helper. Semantic identity
// remains the complete candidate-key-to-fingerprint map owned by compare.
func (r StudyResult) RolesByFingerprint() map[clifixture.CandidateRole]domain.ProjectionFingerprint {
	result := make(map[clifixture.CandidateRole]domain.ProjectionFingerprint)
	if !r.HasOutcomeMap {
		return result
	}
	for _, entry := range r.OutcomeMap.Entries() {
		if role, present := r.CandidateRoles[entry.CandidateKey]; present {
			result[role] = entry.ProjectionFingerprint
		}
	}
	return result
}

// CanonicalCandidateRoster exposes a stable inspection order without granting
// caller order semantic authority.
func (r StudyResult) CanonicalCandidateRoster() []domain.CandidateExecutionKey {
	result := make([]domain.CandidateExecutionKey, 0, len(r.CandidateRoles))
	for key := range r.CandidateRoles {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}
