// Package clistudy owns the typed CLI reference workflow used by the U7
// application. It consumes the already-prepared outer fixture; it never
// creates a nested candidate repository.
package clistudy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

const (
	defaultDiscoveryRepeats = 3
	defaultWallBudget       = 45 * time.Second
)

// Config binds one run to app-admitted roots and executables. No field carries
// a requested semantic result.
type Config struct {
	RepositoryRoot string
	ScratchRoot    string
	EvidenceRoot   string
	GitExecutable  string
	NodeExecutable string
	Ordinal        int
}

type workflowCandidateAuthority struct {
	roles       []reference.CLIRole
	declaration gitobj.CandidateSetDeclaration
	roleByTree  map[domain.Digest]reference.CLIRole
}

type cliWorkflowScope struct {
	fixture               reference.CLIFixture
	repository            gitobj.Repository
	materializationPolicy gitobj.Policy
	main                  workflowCandidateAuthority
	auxiliary             workflowCandidateAuthority
	tools                 world.ToolRegistry
	closed                bool
}

type phaseRunSpec struct {
	label                   string
	purpose                 domain.AttemptPurpose
	repetitions             int
	planDiscoveryRepeats    int
	planConfirmationRepeats int
	roles                   []reference.CLIRole
	stimulus                *countercli.CLIStimulus
	projectionFields        []countercli.CLIFieldID
	stdoutBytes             int64
	reductionProposalLimit  int
	reductionCandidateLimit int
	reductionWallMS         int64
	confirmation            *confirmationRunInput
}

type confirmationRunInput struct {
	reducedBaseline compare.DivergentBaseline
	reductionRun    reducer.ReductionRun
	reductionResult grade.Result
}

// Trial retains the typed physical bridge into the generic observation core.
type Trial struct {
	Role                reference.CLIRole
	Slot                observe.ScheduledTrial
	Admitted            bool
	Result              world.Result
	Measurements        domain.InstanceMeasurements
	Observation         countercli.CLICapturedObservation
	Projection          countercli.CLIProjectionResult
	Projected           bool
	ProjectionRejection *countercli.ProjectionRejection
}

// Result contains the exact authorities produced by one physical phase.
type Result struct {
	Root                 string
	SourceSpecDigest     domain.Digest
	SourceSpecBytes      []byte
	Plan                 domain.WorldPlan
	Envelope             domain.ComparisonEnvelope
	Stimulus             countercli.CLIStimulus
	Binding              countercli.CLIExecutionBinding
	CapturePolicy        countercli.CLICapturePolicy
	ProjectionDefinition countercli.CLIProjectionDefinition
	Observation          observe.ObservationRun
	OutcomeMap           compare.CandidateOutcomeMap
	HasOutcomeMap        bool
	CandidateBindings    []domain.CandidateExecutionBinding
	CandidateRoles       map[domain.CandidateExecutionKey]reference.CLIRole
	Trials               []Trial
	Confirmation         confirmation.Completed
	HasConfirmation      bool
}

// Run executes a complete three-role discovery matrix against the admitted
// outer fixture.
func Run(ctx context.Context, config Config) (result Result, returnErr error) {
	scope, err := openCLIWorkflowScope(ctx, config)
	if err != nil {
		return Result{}, err
	}
	defer func() { returnErr = errors.Join(returnErr, scope.Close()) }()
	return runPhaseWithScope(ctx, config, scope, phaseRunSpec{
		label: "discovery", purpose: domain.AttemptDiscovery,
		repetitions: defaultDiscoveryRepeats,
	})
}

func openCLIWorkflowScope(ctx context.Context, config Config) (_ *cliWorkflowScope, returnErr error) {
	if ctx == nil || ctx.Err() != nil || config.Ordinal < 1 || config.Ordinal > 3 ||
		!cleanAbsolute(config.RepositoryRoot) || !cleanAbsolute(config.ScratchRoot) ||
		!cleanAbsolute(config.EvidenceRoot) || !cleanAbsolute(config.GitExecutable) ||
		!cleanAbsolute(config.NodeExecutable) {
		return nil, fmt.Errorf("CLI_STUDY_SCOPE_CONFIG_REFUSED")
	}
	scopeRoot, err := privateTempDirectory(config.ScratchRoot, fmt.Sprintf("u7c-%d-workflow-scope-", config.Ordinal))
	if err != nil {
		return nil, err
	}
	gitScratch, err := createPrivateDirectory(scopeRoot, "git-scratch")
	if err != nil {
		return nil, err
	}
	fixture, err := reference.OpenCLIFixture(ctx, reference.CLIFixtureConfig{
		Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
	})
	if err != nil {
		return nil, err
	}
	keepFixture := false
	defer func() {
		if !keepFixture {
			returnErr = errors.Join(returnErr, fixture.Close())
		}
	}()
	repository := fixture.Repository()
	if !repository.Valid() || fixture.Entrypoint() != reference.CLIEntrypoint {
		return nil, fmt.Errorf("CLI_STUDY_SCOPE_FIXTURE_REFUSED")
	}
	policy, err := gitobj.NewPolicy(16, 1<<20, 1<<19)
	if err != nil {
		return nil, err
	}
	mainInspection, err := createPrivateDirectory(scopeRoot, "main-inspection")
	if err != nil {
		return nil, err
	}
	main, err := inspectWorkflowCandidateAuthority(ctx, fixture, repository, reference.CLIRoles(), policy, mainInspection)
	if err != nil {
		return nil, err
	}
	auxiliaryInspection, err := createPrivateDirectory(scopeRoot, "auxiliary-inspection")
	if err != nil {
		return nil, err
	}
	auxiliary, err := inspectWorkflowCandidateAuthority(
		ctx, fixture, repository,
		[]reference.CLIRole{reference.ConfigFirst, reference.ArgvFirst},
		policy, auxiliaryInspection,
	)
	if err != nil {
		return nil, err
	}
	toolScratch, err := createPrivateDirectory(scopeRoot, "tools")
	if err != nil {
		return nil, err
	}
	tools, err := world.NewToolRegistry(ctx, toolScratch, world.ToolSpec{
		Name: "node", AbsolutePath: config.NodeExecutable,
		VersionConstraint: runnerprofile.NodeToolConstraintV1, VersionArgs: []string{"--version"},
	})
	if err != nil {
		return nil, err
	}
	scope := &cliWorkflowScope{
		fixture: fixture, repository: repository, materializationPolicy: policy,
		main: main, auxiliary: auxiliary, tools: tools,
	}
	if !scope.valid() {
		return nil, fmt.Errorf("CLI_STUDY_SCOPE_AUTHORITY_REFUSED")
	}
	keepFixture = true
	return scope, nil
}

func inspectWorkflowCandidateAuthority(
	ctx context.Context,
	fixture reference.CLIFixture,
	repository gitobj.Repository,
	roles []reference.CLIRole,
	policy gitobj.Policy,
	inspectionRoot string,
) (workflowCandidateAuthority, error) {
	normalized, err := normalizeRoles(roles)
	if err != nil || ctx == nil || !fixture.Valid() || !repository.Valid() || !policy.Valid() || !cleanAbsolute(inspectionRoot) {
		return workflowCandidateAuthority{}, errors.Join(err, fmt.Errorf("CLI_STUDY_SCOPE_CANDIDATES_REFUSED"))
	}
	pins := make([]gitobj.PinnedTree, 0, len(normalized))
	roleByTree := make(map[domain.Digest]reference.CLIRole, len(normalized))
	for _, role := range normalized {
		ref, refErr := fixture.Ref(role)
		if refErr != nil {
			return workflowCandidateAuthority{}, refErr
		}
		pinned, pinErr := repository.Pin(ctx, ref)
		if pinErr != nil {
			return workflowCandidateAuthority{}, pinErr
		}
		if _, duplicate := roleByTree[pinned.IdentityDigest()]; duplicate {
			return workflowCandidateAuthority{}, fmt.Errorf("CLI_STUDY_SCOPE_CANDIDATE_ALIAS_REFUSED")
		}
		pins = append(pins, pinned)
		roleByTree[pinned.IdentityDigest()] = role
	}
	selected, err := gitobj.SelectTrees(pins...)
	if err != nil {
		return workflowCandidateAuthority{}, err
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, policy, inspectionRoot)
	if err != nil || !declaration.Valid() || declaration.CandidateCount() != len(normalized) {
		return workflowCandidateAuthority{}, errors.Join(err, fmt.Errorf("CLI_STUDY_SCOPE_DECLARATION_REFUSED"))
	}
	return workflowCandidateAuthority{
		roles: append([]reference.CLIRole(nil), normalized...), declaration: declaration, roleByTree: roleByTree,
	}, nil
}

func (scope *cliWorkflowScope) valid() bool {
	if scope == nil || scope.closed || !scope.fixture.Valid() || !scope.repository.Valid() ||
		!scope.materializationPolicy.Valid() || !scope.main.valid() || !scope.auxiliary.valid() {
		return false
	}
	names := scope.tools.Names()
	return len(names) == 1 && names[0] == "node"
}

func (authority workflowCandidateAuthority) valid() bool {
	if !authority.declaration.Valid() || authority.declaration.CandidateCount() != len(authority.roles) ||
		len(authority.roleByTree) != len(authority.roles) {
		return false
	}
	seen := make(map[reference.CLIRole]struct{}, len(authority.roles))
	for _, role := range authority.roleByTree {
		seen[role] = struct{}{}
	}
	if len(seen) != len(authority.roles) {
		return false
	}
	for _, role := range authority.roles {
		if _, present := seen[role]; !present {
			return false
		}
	}
	return true
}

func (scope *cliWorkflowScope) candidateAuthority(roles []reference.CLIRole) (workflowCandidateAuthority, error) {
	if !scope.valid() {
		return workflowCandidateAuthority{}, fmt.Errorf("CLI_STUDY_SCOPE_CLOSED")
	}
	if sameCLIRoles(roles, scope.main.roles) {
		return scope.main, nil
	}
	if sameCLIRoles(roles, scope.auxiliary.roles) {
		return scope.auxiliary, nil
	}
	return workflowCandidateAuthority{}, fmt.Errorf("CLI_STUDY_SCOPE_ROLE_ROSTER_REFUSED")
}

func (scope *cliWorkflowScope) Close() error {
	if scope == nil || scope.closed {
		return nil
	}
	scope.closed = true
	return scope.fixture.Close()
}

func sameCLIRoles(left, right []reference.CLIRole) bool {
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

func runPhaseWithScope(ctx context.Context, config Config, scope *cliWorkflowScope, phase phaseRunSpec) (result Result, returnErr error) {
	if len(phase.roles) == 0 {
		phase.roles = reference.CLIRoles()
	}
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
	roles, err := normalizeRoles(phase.roles)
	if ctx == nil || config.Ordinal < 1 || config.Ordinal > 3 ||
		!cleanAbsolute(config.RepositoryRoot) || !cleanAbsolute(config.ScratchRoot) ||
		!cleanAbsolute(config.EvidenceRoot) || !cleanAbsolute(config.GitExecutable) ||
		!cleanAbsolute(config.NodeExecutable) || !phase.purpose.Valid() || !validRunLabel(phase.label) ||
		phase.repetitions < 1 || phase.repetitions > 5 || phase.repetitions != expectedRepeats ||
		phase.planDiscoveryRepeats < 1 || phase.planDiscoveryRepeats > 5 ||
		phase.planConfirmationRepeats < 1 || phase.planConfirmationRepeats > 5 || err != nil || !scope.valid() {
		return Result{}, fmt.Errorf("CLI_STUDY_CONFIG_REFUSED: %v", err)
	}
	hasReductionBudget := phase.reductionProposalLimit != 0 || phase.reductionCandidateLimit != 0 || phase.reductionWallMS != 0
	if hasReductionBudget && (phase.reductionProposalLimit < 1 || phase.reductionCandidateLimit < 1 || phase.reductionWallMS < 1) {
		return Result{}, fmt.Errorf("CLI_STUDY_REDUCTION_BUDGET_REFUSED")
	}
	if phase.confirmation != nil && (phase.purpose != domain.AttemptConfirmation ||
		!phase.confirmation.reducedBaseline.Valid() || !phase.confirmation.reductionRun.Valid() ||
		!phase.confirmation.reductionResult.Valid()) {
		return Result{}, fmt.Errorf("CLI_STUDY_CONFIRMATION_CONFIG_REFUSED")
	}

	runRoot, err := privateRunRoot(config.ScratchRoot, config.Ordinal, phase.label)
	if err != nil {
		return Result{}, err
	}
	allocationRoot, err := createPrivateDirectory(runRoot, "attempts")
	if err != nil {
		return Result{}, err
	}

	authority, err := scope.candidateAuthority(roles)
	if err != nil || !scope.repository.Valid() || scope.fixture.Entrypoint() != reference.CLIEntrypoint {
		return Result{}, fmt.Errorf("CLI_STUDY_FIXTURE_AUTHORITY_REFUSED")
	}

	stimulus, err := defaultCLIStimulus(scope.fixture.Entrypoint())
	if phase.stimulus != nil {
		stimulus = *phase.stimulus
	}
	if err != nil || !stimulus.Valid() {
		return Result{}, errors.Join(err, fmt.Errorf("CLI_STUDY_STIMULUS_REFUSED"))
	}
	stdoutBytes := phase.stdoutBytes
	if stdoutBytes == 0 {
		stdoutBytes = 64 << 10
	}
	capturePolicy, err := countercli.NewCLICapturePolicy(countercli.CLICapturePolicyConfig{
		StdoutBytes: stdoutBytes, StderrBytes: 64 << 10,
	})
	if err != nil {
		return Result{}, err
	}
	fixtureRecipe, err := countercli.NewCLIFixtureRecipe()
	if err != nil {
		return Result{}, err
	}
	projectionFields := phase.projectionFields
	if len(projectionFields) == 0 {
		projectionFields = []countercli.CLIFieldID{
			countercli.CLIFieldExitCode,
			countercli.CLIFieldStdoutJSONMode,
			countercli.CLIFieldStdoutJSONSource,
		}
	}
	projection, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{
		Fields: append([]countercli.CLIFieldID(nil), projectionFields...),
	})
	if err != nil {
		return Result{}, err
	}
	envelope, err := newStudyEnvelope()
	if err != nil {
		return Result{}, err
	}
	runnerDigest, err := runnerprofile.CLIDigest()
	if err != nil {
		return Result{}, err
	}
	plan, sourceDigest, sourceBytes, err := compileStudyPlan(studyPlanInput{
		candidateSetDigest: authority.declaration.Digest(), materializationPolicyDigest: scope.materializationPolicy.Digest(),
		comparisonEnvelopeDigest: envelope.Digest(), runnerDigest: runnerDigest,
		startArgv: stimulus.BaseLogicalArgv(), fixtureRecipeDigest: fixtureRecipe.Digest(),
		capturePolicy: capturePolicy, projectionDefinition: projection.Binding(),
		discoveryRepeats: phase.planDiscoveryRepeats, confirmationRepeats: phase.planConfirmationRepeats,
		candidateCount: len(roles), reductionProposalLimit: phase.reductionProposalLimit,
		reductionCandidateLimit: phase.reductionCandidateLimit, reductionWallMS: phase.reductionWallMS,
	})
	if err != nil {
		return Result{}, err
	}
	binding, err := countercli.BindExecution(plan, stimulus, capturePolicy, projection)
	if err != nil {
		return Result{}, err
	}
	bound, err := authority.declaration.Bind(plan)
	if err != nil {
		return Result{}, err
	}
	candidateByKey := make(map[domain.CandidateExecutionKey]gitobj.BoundCandidate, len(bound))
	roleByKey := make(map[domain.CandidateExecutionKey]reference.CLIRole, len(bound))
	bindings := make([]domain.CandidateExecutionBinding, 0, len(bound))
	roster := make([]domain.CandidateExecutionKey, 0, len(bound))
	for _, candidate := range bound {
		role, ok := authority.roleByTree[candidate.TreeIdentityDigest()]
		if !ok {
			return Result{}, fmt.Errorf("CLI_STUDY_UNKNOWN_TREE_REFUSED")
		}
		key := candidate.Binding().Key()
		candidateByKey[key] = candidate
		roleByKey[key] = role
		bindings = append(bindings, candidate.Binding())
		roster = append(roster, key)
	}
	trials := make([]Trial, 0, phase.repetitions*len(roster))
	execute := func(trialContext context.Context, slot observe.ScheduledTrial, nonce string) (world.Result, observe.PreparedTrial, error) {
		candidate, ok := candidateByKey[slot.CandidateKey()]
		if !ok {
			return world.Result{}, observe.PreparedTrial{}, fmt.Errorf("CLI_STUDY_SCHEDULE_REFUSED")
		}
		physical, executeErr := world.ExecuteCLI(trialContext, world.CLIRequest{
			Binding: binding, Candidate: candidate, Tools: scope.tools, AllocationRoot: allocationRoot,
			Purpose: phase.purpose, InstanceNonce: nonce, ScheduleOrdinal: slot.Ordinal(),
		})
		if executeErr != nil {
			return world.Result{}, observe.PreparedTrial{}, executeErr
		}
		measurements, measureErr := studyMeasurements(envelope, physical)
		if measureErr != nil {
			return world.Result{}, observe.PreparedTrial{}, measureErr
		}
		captured, captureErr := countercli.AdaptWorldResult(physical, binding, slot.Ordinal())
		if captureErr != nil {
			return world.Result{}, observe.PreparedTrial{}, captureErr
		}
		evidence := Trial{Role: roleByKey[slot.CandidateKey()], Slot: slot, Result: physical, Measurements: measurements, Observation: captured}
		if physical.FinalizedAttempt().HasControls() {
			prepared, prepareErr := observe.NewPreparedControlledTrial(slot, physical.World(), physical.FinalizedAttempt(), measurements)
			if prepareErr == nil {
				trials = append(trials, evidence)
			}
			return physical, prepared, prepareErr
		}
		projected, projectErr := projection.Project(captured)
		if projectErr != nil {
			var rejection *countercli.ProjectionRejection
			if !errors.As(projectErr, &rejection) {
				return world.Result{}, observe.PreparedTrial{}, projectErr
			}
			bridge, bridgeErr := countercli.PrepareProjectionRejectionEvidence(captured, rejection)
			if bridgeErr != nil {
				return world.Result{}, observe.PreparedTrial{}, bridgeErr
			}
			prepared, prepareErr := observe.NewPreparedProjectionRejectedTrial(slot, physical.World(), physical.FinalizedAttempt(), measurements, bridge)
			if prepareErr == nil {
				evidence.ProjectionRejection = rejection
				trials = append(trials, evidence)
			}
			return physical, prepared, prepareErr
		}
		structural, bridgeErr := countercli.PrepareStructuralCapture(physical.World(), physical.FinalizedAttempt(), captured, projected)
		if bridgeErr != nil {
			return world.Result{}, observe.PreparedTrial{}, bridgeErr
		}
		prepared, prepareErr := observe.NewPreparedCapturedTrial(slot, physical.World(), physical.FinalizedAttempt(), measurements, structural)
		if prepareErr == nil {
			evidence.Projection = projected
			evidence.Projected = true
			trials = append(trials, evidence)
		}
		return physical, prepared, prepareErr
	}

	var observation observe.ObservationRun
	var confirmed confirmation.Completed
	if phase.confirmation != nil {
		confirmed, err = confirmation.Run(ctx, confirmation.Request{
			Plan: plan, Envelope: envelope, ReducedBaseline: phase.confirmation.reducedBaseline,
			ReductionRun: phase.confirmation.reductionRun, ReductionResult: phase.confirmation.reductionResult,
			WallBudget: defaultWallBudget,
			Execute: func(runContext context.Context, request confirmation.TrialRequest) (world.Result, observe.PreparedTrial, error) {
				return execute(runContext, request.Slot(), request.InstanceNonce())
			},
		})
	} else {
		observation, err = observe.RunObservation(ctx, observe.ObservationConfig{
			Plan: plan, Purpose: phase.purpose, Envelope: envelope, CandidateRoster: roster,
			Repetitions: phase.repetitions,
			Budget:      observe.TrialBudget{MaxTotalTrials: phase.repetitions * len(roster), WallBudget: defaultWallBudget},
		}, func(runContext context.Context, slot observe.ScheduledTrial) (observe.PreparedTrial, error) {
			_, prepared, runErr := execute(runContext, slot, fmt.Sprintf("u7c-%d-%s-%s-%d-%d", config.Ordinal, phase.label, purposeToken(phase.purpose), slot.Repetition(), slot.Ordinal()))
			return prepared, runErr
		})
	}
	if err != nil {
		return Result{}, err
	}
	admitted := make(map[domain.Digest]struct{})
	if phase.confirmation != nil {
		for _, digest := range confirmed.ConfirmedOutcomeMap().OutcomeMap().EvidenceAttemptDigests() {
			admitted[digest] = struct{}{}
		}
	} else {
		for _, batch := range observation.Batches() {
			for _, digest := range batch.AttemptDigests() {
				admitted[digest] = struct{}{}
			}
		}
	}
	for index := range trials {
		_, trials[index].Admitted = admitted[trials[index].Result.FinalizedAttempt().ArtifactDigest()]
	}
	result = Result{
		Root: runRoot, SourceSpecDigest: sourceDigest, SourceSpecBytes: append([]byte(nil), sourceBytes...),
		Plan: plan, Envelope: envelope, Stimulus: stimulus, Binding: binding, CapturePolicy: capturePolicy,
		ProjectionDefinition: projection, Observation: observation, CandidateBindings: append([]domain.CandidateExecutionBinding(nil), bindings...),
		CandidateRoles: cloneRoleMap(roleByKey), Trials: append([]Trial(nil), trials...), Confirmation: confirmed,
		HasConfirmation: phase.confirmation != nil && confirmed.Valid(),
	}
	if phase.confirmation != nil {
		result.OutcomeMap = confirmed.ConfirmedOutcomeMap().OutcomeMap()
		result.HasOutcomeMap = true
	} else if mapInput, ok := observation.OutcomeMapInput(); ok {
		outcome, mapErr := compare.NewCandidateOutcomeMap(mapInput.StimulusDigest(), mapInput.EnvelopeDigest(), mapInput.Roster(), mapInput.Batches())
		if mapErr != nil {
			return Result{}, mapErr
		}
		result.OutcomeMap = outcome
		result.HasOutcomeMap = true
	}
	return result, nil
}

func defaultCLIStimulus(entrypoint string) (countercli.CLIStimulus, error) {
	environment, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		return countercli.CLIStimulus{}, err
	}
	fixture, err := countercli.NewFixtureFile("config.json", []byte(`{"mode":"config"}`), countercli.FixtureMode0644)
	if err != nil {
		return countercli.CLIStimulus{}, err
	}
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{entrypoint}, Argv: []string{"--mode", "argv"},
		Stdin: countercli.AbsentStdin(), Environment: []countercli.CLIEnvironmentBinding{environment},
		Fixtures: []countercli.CLIFixtureFile{fixture}, CWDPolicy: countercli.CWDMaterializedRoot,
	})
}

type studyPlanInput struct {
	candidateSetDigest, materializationPolicyDigest, comparisonEnvelopeDigest domain.Digest
	runnerDigest, fixtureRecipeDigest                                         domain.Digest
	startArgv                                                                 []string
	capturePolicy                                                             countercli.CLICapturePolicy
	projectionDefinition                                                      domain.ProjectionDefinitionBinding
	discoveryRepeats, confirmationRepeats, candidateCount                     int
	reductionProposalLimit, reductionCandidateLimit                           int
	reductionWallMS                                                           int64
}

type sourceAdapter struct {
	Domain         string `json:"domain"`
	AdapterVersion string `json:"adapter_version"`
	RunnerDigest   string `json:"runner_digest"`
}
type sourceReadiness struct {
	Kind string `json:"kind"`
}
type sourceBudgets struct {
	CandidateCount            int   `json:"candidate_count"`
	MaterializedEntryCount    int   `json:"materialized_entry_count"`
	MaterializedBytesPerWorld int64 `json:"materialized_bytes_per_world"`
	SingleBlobBytes           int64 `json:"single_blob_bytes"`
	ReadinessMS               int64 `json:"readiness_ms"`
	ProbeMS                   int64 `json:"probe_ms"`
	TeardownMS                int64 `json:"teardown_ms"`
	StdoutBytes               int64 `json:"stdout_bytes"`
	StderrBytes               int64 `json:"stderr_bytes"`
	BodyBytes                 int64 `json:"http_body_bytes"`
	ProposedShrinkStimuli     int   `json:"proposed_shrink_stimuli"`
	TotalCandidateTrials      int   `json:"total_candidate_trials"`
	ShrinkWallMS              int64 `json:"shrink_wall_ms"`
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
	Budgets                     sourceBudgets             `json:"budgets"`
}

func compileStudyPlan(input studyPlanInput) (domain.WorldPlan, domain.Digest, []byte, error) {
	proposalLimit := input.reductionProposalLimit
	candidateLimit := input.candidateCount * input.discoveryRepeats * 2
	wallMS := int64(1000)
	if proposalLimit > 0 {
		candidateLimit = input.reductionCandidateLimit
		wallMS = input.reductionWallMS
	}
	identity := sourceSpec{
		SchemaVersion: "countershape-source/v1", Kind: "SourceSpec",
		CandidateSetDigest: input.candidateSetDigest.String(), MaterializationPolicyDigest: input.materializationPolicyDigest.String(),
		ComparisonEnvelopeDigest: input.comparisonEnvelopeDigest.String(),
		Adapter:                  sourceAdapter{Domain: string(domain.AdapterCLI), AdapterVersion: runnerprofile.CLIAdapterVersionV1, RunnerDigest: input.runnerDigest.String()},
		ExecutionShape:           string(domain.OneCLIInvocation), StartArgv: append([]string(nil), input.startArgv...), SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}, {Name: "LC_ALL", Value: "C"}, {Name: "NODE_NO_WARNINGS", Value: "1"}, {Name: "NO_COLOR", Value: "1"}, {Name: "TZ", Value: "UTC"}},
		SecretSlots: []domain.SecretSlot{}, FixtureRecipeDigest: input.fixtureRecipeDigest.String(),
		Readiness: sourceReadiness{Kind: string(domain.ReadinessNone)}, CapturePolicyDigest: input.capturePolicy.Digest().String(),
		ProjectionDefinitionDigest: input.projectionDefinition.Digest().String(),
		RepeatSchedule:             domain.RepeatSchedule{DiscoveryRepeats: input.discoveryRepeats, ConfirmationRepeats: input.confirmationRepeats, Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools:              []domain.RequiredTool{{Name: "node", VersionConstraint: runnerprofile.NodeToolConstraintV1}},
		Budgets:                    sourceBudgets{CandidateCount: input.candidateCount, MaterializedEntryCount: 16, MaterializedBytesPerWorld: 1 << 20, SingleBlobBytes: 1 << 19, ProbeMS: 1500, TeardownMS: 800, StdoutBytes: input.capturePolicy.StdoutBytes(), StderrBytes: input.capturePolicy.StderrBytes(), BodyBytes: 64 << 10, ProposedShrinkStimuli: proposalLimit, TotalCandidateTrials: candidateLimit, ShrinkWallMS: wallMS},
	}
	bytes, err := canon.CanonicalizeTyped(identity)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	parsed, err := corespec.ParseSource(bytes)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	plan, err := corespec.Compile(parsed, input.projectionDefinition)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	return plan, parsed.Digest(), parsed.CanonicalBytes(), nil
}

func normalizeRoles(input []reference.CLIRole) ([]reference.CLIRole, error) {
	if len(input) < 1 || len(input) > len(reference.CLIRoles()) {
		return nil, fmt.Errorf("invalid CLI role count")
	}
	allowed := make(map[reference.CLIRole]struct{}, len(reference.CLIRoles()))
	for _, role := range reference.CLIRoles() {
		allowed[role] = struct{}{}
	}
	seen := make(map[reference.CLIRole]struct{}, len(input))
	for _, role := range input {
		if _, ok := allowed[role]; !ok {
			return nil, fmt.Errorf("invalid CLI role %q", role)
		}
		if _, duplicate := seen[role]; duplicate {
			return nil, fmt.Errorf("duplicate CLI role %q", role)
		}
		seen[role] = struct{}{}
	}
	return append([]reference.CLIRole(nil), input...), nil
}

func cleanAbsolute(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path
}

func validRunLabel(label string) bool {
	if label == "" {
		return true
	}
	if len(label) > 80 {
		return false
	}
	for _, value := range label {
		if !(value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == '-') {
			return false
		}
	}
	return true
}

func privateRunRoot(parent string, ordinal int, label string) (string, error) {
	if label == "" {
		label = "phase"
	}
	return privateTempDirectory(parent, fmt.Sprintf("u7c-%d-%s-", ordinal, label))
}

func privateTempDirectory(parent, pattern string) (string, error) {
	if !cleanAbsolute(parent) {
		return "", fmt.Errorf("CLI_STUDY_PRIVATE_ROOT_REFUSED")
	}
	canonical, err := filepath.EvalSymlinks(parent)
	if err != nil || canonical != parent {
		return "", errors.Join(err, fmt.Errorf("CLI_STUDY_PRIVATE_ROOT_REFUSED"))
	}
	path, err := os.MkdirTemp(canonical, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return "", errors.Join(err, fmt.Errorf("CLI_STUDY_PRIVATE_ROOT_REFUSED"))
	}
	return path, nil
}

func createPrivateDirectory(parent, name string) (string, error) {
	if !cleanAbsolute(parent) || filepath.Base(name) != name || name == "." {
		return "", fmt.Errorf("CLI_STUDY_PRIVATE_ROOT_REFUSED")
	}
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return "", errors.Join(err, fmt.Errorf("CLI_STUDY_PRIVATE_ROOT_REFUSED"))
	}
	return path, nil
}

func purposeToken(purpose domain.AttemptPurpose) string {
	switch purpose {
	case domain.AttemptDiscovery:
		return "discovery"
	case domain.AttemptReduction:
		return "reduction"
	case domain.AttemptFinalSweep:
		return "final-sweep"
	case domain.AttemptConfirmation:
		return "confirmation"
	case domain.AttemptConformance:
		return "conformance"
	default:
		return "invalid"
	}
}

func cloneRoleMap(input map[domain.CandidateExecutionKey]reference.CLIRole) map[domain.CandidateExecutionKey]reference.CLIRole {
	result := make(map[domain.CandidateExecutionKey]reference.CLIRole, len(input))
	for key, role := range input {
		result[key] = role
	}
	return result
}

// CanonicalCandidateRoster returns display-only stable ordering.
func (result Result) CanonicalCandidateRoster() []domain.CandidateExecutionKey {
	roster := make([]domain.CandidateExecutionKey, 0, len(result.CandidateRoles))
	for key := range result.CandidateRoles {
		roster = append(roster, key)
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i].String() < roster[j].String() })
	return roster
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
		{dimensionToolDigest, domain.MeasuredToolReceipt}, {dimensionToolMajor, domain.MeasuredToolReceipt},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt}, {dimensionStdinPresence, domain.MeasuredProcessReceipt},
		{dimensionStdinBytes, domain.MeasuredProcessReceipt},
	}
	recorded := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt},
		{dimensionWorldDigest, domain.MeasuredWorldInstance}, {dimensionAttemptDigest, domain.MeasuredWorldInstance},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance}, {dimensionPID, domain.MeasuredProcessReceipt},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt}, {dimensionWorkingDirectory, domain.MeasuredProcessReceipt},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe}, {dimensionFixtureOverlay, domain.MeasuredFilesystemProbe},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt}, {dimensionStdoutObserved, domain.MeasuredProcessReceipt},
		{dimensionStderrObserved, domain.MeasuredProcessReceipt},
	}
	measured := make([]domain.MeasuredDimension, 0, len(required)+len(recorded))
	equal := make([]domain.RequiredEqualDimension, 0, len(required))
	tolerated := make([]domain.ToleratedDimension, 0, len(recorded))
	for _, item := range required {
		measured = append(measured, domain.MeasuredDimension{Name: item.name, Source: item.source, Comparison: domain.CompareExact})
		equal = append(equal, domain.RequiredEqualDimension{Name: item.name})
	}
	for _, item := range recorded {
		measured = append(measured, domain.MeasuredDimension{Name: item.name, Source: item.source, Comparison: domain.CompareRecordedOnly})
		tolerated = append(tolerated, domain.ToleratedDimension{Name: item.name, Tolerance: domain.MayDifferRecorded})
	}
	return domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: "u7c-cli-study/v1", Measured: measured, RequiredEqual: equal, Tolerated: tolerated,
		Rejected: []domain.RejectedDimension{}, Uncontrolled: []string{"host network availability", "kernel scheduling and wall-clock timing"},
	})
}

func studyMeasurements(envelope domain.ComparisonEnvelope, result world.Result) (domain.InstanceMeasurements, error) {
	process := result.Process()
	overlay, hasOverlay := result.CLIFixtureOverlay()
	invocation, hasInvocation := result.CLIInvocationEvidence()
	if (hasOverlay && !overlay.Valid()) || (!hasOverlay && process.FixtureOverlayDigest().Valid()) ||
		(hasInvocation && !invocation.Valid()) || (!hasInvocation && (process.InvocationEvidenceDigest().Valid() ||
		process.InvocationEvidenceStatus() != world.CLIInvocationNotInspected || process.InvocationEvidencePresence() != world.CLIInvocationPresenceUnknown)) {
		return domain.InstanceMeasurements{}, fmt.Errorf("CLI_STUDY_OPTIONAL_AUTHORITY_REFUSED")
	}
	overlayValue, err := canonicalOptionalReceipt(hasOverlay, overlay.Digest(), "", "")
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	invocationValue, err := canonicalOptionalReceipt(hasInvocation, invocation.Digest(), string(process.InvocationEvidenceStatus()), string(process.InvocationEvidencePresence()))
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	argvValue, err := canonicalStrings(process.DeclaredLogicalArgv())
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	values := []struct {
		name   string
		source domain.MeasuredSource
		value  any
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt, process.ToolExecutableDigest().String()}, {dimensionToolMajor, domain.MeasuredToolReceipt, int64(process.ToolMajor())},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, process.ExecutionAuthorityMarker()}, {dimensionLogicalArgv, domain.MeasuredProcessReceipt, argvValue},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt, process.CWDPolicy()}, {dimensionStdinPresence, domain.MeasuredProcessReceipt, process.StdinPresence()},
		{dimensionStdinBytes, domain.MeasuredProcessReceipt, process.StdinBytes()}, {dimensionWorldDigest, domain.MeasuredWorldInstance, result.World().Digest().String()},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance, result.FinalizedAttempt().ArtifactDigest().String()}, {dimensionScheduleOrdinal, domain.MeasuredWorldInstance, int64(result.World().ScheduleOrdinal())},
		{dimensionPID, domain.MeasuredProcessReceipt, int64(process.PID())}, {dimensionPhysicalExecution, domain.MeasuredProcessReceipt, process.PhysicalExecutionEntered()},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt, process.WorkingDirectory()}, {dimensionFixtureRoot, domain.MeasuredFilesystemProbe, result.Roots().Fixture()},
		{dimensionFixtureOverlay, domain.MeasuredFilesystemProbe, overlayValue}, {dimensionInvocationEvidence, domain.MeasuredProcessReceipt, invocationValue},
		{dimensionStdoutObserved, domain.MeasuredProcessReceipt, process.StdoutObservedBytes()}, {dimensionStderrObserved, domain.MeasuredProcessReceipt, process.StderrObservedBytes()},
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
			err = fmt.Errorf("CLI_STUDY_MEASUREMENT_REFUSED")
		}
		if err != nil {
			return domain.InstanceMeasurements{}, err
		}
		measurements = append(measurements, domain.MeasurementValue{Name: input.name, Source: input.source, Value: value})
	}
	return domain.NewInstanceMeasurements(envelope, result.World(), measurements)
}

func canonicalOptionalReceipt(present bool, digest domain.Digest, status, physicalPresence string) (canon.Value, error) {
	presence, digestText := "ABSENT", ""
	if present {
		if !digest.Valid() {
			return canon.Value{}, fmt.Errorf("CLI_STUDY_OPTIONAL_DIGEST_REFUSED")
		}
		presence, digestText = "PRESENT", digest.String()
	} else if digest.Valid() {
		return canon.Value{}, fmt.Errorf("CLI_STUDY_OPTIONAL_DIGEST_REFUSED")
	}
	digestValue, err := canon.String(digestText)
	if err != nil {
		return canon.Value{}, err
	}
	physicalValue, err := canon.String(physicalPresence)
	if err != nil {
		return canon.Value{}, err
	}
	presenceValue, err := canon.String(presence)
	if err != nil {
		return canon.Value{}, err
	}
	statusValue, err := canon.String(status)
	if err != nil {
		return canon.Value{}, err
	}
	return canon.Object(canon.Member{Name: "digest", Value: digestValue}, canon.Member{Name: "physical_presence", Value: physicalValue}, canon.Member{Name: "presence", Value: presenceValue}, canon.Member{Name: "status", Value: statusValue})
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

// Keep app in this compilation unit as an architectural dependency edge. The
// concrete handler is defined alongside the completion seal in reduction.go.
var _ app.StudyHandler
