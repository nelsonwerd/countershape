package clistudy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

type completedCLIStudySeal struct{ marker byte }

var issuedCompletedCLIStudy = &completedCLIStudySeal{marker: 1}

type phaseFact struct {
	phase       string
	trial       int
	kind        string
	authority   domain.Digest
	world       domain.Digest
	attempt     domain.Digest
	measurement domain.Digest
	capture     domain.Digest
	projection  domain.Digest
}

type physicalControl struct {
	kind                string
	role                reference.CLIRole
	root                string
	plan                domain.WorldPlan
	envelope            domain.ComparisonEnvelope
	stimulus            countercli.CLIStimulus
	binding             countercli.CLIExecutionBinding
	definition          countercli.CLIProjectionDefinition
	result              world.Result
	measurements        domain.InstanceMeasurements
	observation         countercli.CLICapturedObservation
	projection          countercli.CLIProjectionResult
	projected           bool
	projectionRejection *countercli.ProjectionRejection
}

type semanticControlInput struct {
	kind   string
	label  string
	argv   []string
	stdin  countercli.CLIStdin
	env    []countercli.CLIEnvironmentBinding
	fields []countercli.CLIFieldID
}

type overlayControlAudit struct {
	refusalCode    string
	attemptsBefore int
	attemptsAfter  int
	worldsBefore   int
	worldsAfter    int
}

type reductionSequence struct {
	baseline     compare.DivergentBaseline
	run          reducer.ReductionRun
	grade        grade.Result
	minimized    Result
	evaluation   Result
	policyDigest domain.Digest
	sweepDigest  domain.Digest
}

type cliWorkflowTask func(context.Context) error

const (
	independentCLIPhaseWorkerLimit = 7
	semanticCLIControlWorkerLimit  = 16
)

// CompletedCLIStudy is the only public value from which harness evidence may
// be inspected. Its private seal binds the completed typed graph; publication
// additionally requires a separate private witness issued only after all ten
// official operations have closed and revalidated.
type CompletedCLIStudy struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	discovery            Result
	recoveryOne          Result
	recoveryTwo          Result
	shapeReference       Result
	shapeChanged         Result
	baselineCheckpoint   Result
	baseline             Result
	minimized            Result
	confirmed            Result
	mainEvaluation       Result
	auxiliaryBaseline    Result
	auxiliaryEvaluation  Result
	auxiliaryMinimized   Result
	auxiliaryConfirmed   Result
	weakRun              reducer.ReductionRun
	weakGrade            grade.Result
	strongRun            reducer.ReductionRun
	strongGrade          grade.Result
	choicepoint          choice.ChoicepointRecord
	decision             choice.DecisionRecord
	durableRuling        promotion.Ruling
	portableSource       contractsource.PortableSource
	bundle               emitmodel.ContractBundle
	residue              nodeemit.Residue
	choiceAudit          choiceAudit
	officialTrials       []officialTrial
	semanticControls     []physicalControl
	overlayAudit         overlayControlAudit
	phaseFacts           []phaseFact
	seal                 *completedCLIStudySeal
}

// Valid replays the construction joins; the private seal alone is never
// sufficient.
func (completed CompletedCLIStudy) Valid() bool {
	return validateCompletedCLIStudy(completed) == nil
}

// Handler is the app-owned machine-study edge. It cannot publish from a
// partial Result or caller-provided status.
func Handler() app.StudyHandler {
	return func(request app.StudyRequest) error {
		ready, err := completeCLIStudyForPublication(request.Context, Config{
			RepositoryRoot: request.WorkingDirectory,
			ScratchRoot:    request.ScratchRoot,
			EvidenceRoot:   request.EvidenceRoot,
			GitExecutable:  request.GitExecutable,
			NodeExecutable: request.NodeExecutable,
			Ordinal:        request.Ordinal,
		})
		if err != nil {
			return err
		}
		if request.Workspace == nil {
			return fmt.Errorf("CLI_STUDY_COMPLETION_REFUSED")
		}
		if err := request.Context.Err(); err != nil {
			return fmt.Errorf("CLI_STUDY_CONTEXT_ENDED_BEFORE_PUBLICATION: %w", err)
		}
		_, err = publishCompletedEvidence(request.Context, ready, request.Workspace)
		return err
	}
}

// completeCLIStudyForPublication is the sole issuer of the private
// publication witness. completeCLIWorkflow returns only after the completed
// graph has passed its full static replay and all ten official graphs have
// passed their terminal live replay.
func completeCLIStudyForPublication(ctx context.Context, config Config) (publicationReadyCLIStudy, error) {
	completed, err := completeCLIWorkflow(ctx, config)
	if err != nil {
		return publicationReadyCLIStudy{}, err
	}
	inspection, err := projectCompletedInspection(completed)
	if err != nil {
		return publicationReadyCLIStudy{}, err
	}
	input := evidenceInputForCompleted(completed)
	boundInput, err := cloneEvidencePublicationInput(input)
	if err != nil {
		return publicationReadyCLIStudy{}, err
	}
	ready := publicationReadyCLIStudy{
		ordinal: completed.ordinal, physicalRunAuthority: completed.physicalRunAuthority,
		inspection:      cloneStudyInspection(inspection),
		boundInspection: cloneStudyInspection(inspection),
		input:           input, boundInput: boundInput,
		seal: issuedPublicationReadyCLIStudy,
	}
	if !ready.bound() {
		return publicationReadyCLIStudy{}, fmt.Errorf("CLI_STUDY_PUBLICATION_WITNESS_REFUSED")
	}
	return ready, nil
}

// CompleteCLIStudy is implemented as a single closed orchestration so no
// intermediate phase can authorize publication.
func CompleteCLIStudy(ctx context.Context, config Config) (CompletedCLIStudy, error) {
	if ctx == nil {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_CONFIG_REFUSED")
	}
	return completeCLIWorkflow(ctx, config)
}

func completeCLIWorkflow(ctx context.Context, config Config) (completed CompletedCLIStudy, returnErr error) {
	scope, err := openCLIWorkflowScope(ctx, config)
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	scopeOpen := true
	defer func() {
		if scopeOpen {
			returnErr = errors.Join(returnErr, scope.Close())
		}
	}()
	mainPhase := func(label string, purpose domain.AttemptPurpose, repetitions int, stimulus *countercli.CLIStimulus) phaseRunSpec {
		return phaseRunSpec{
			label: label, purpose: purpose, repetitions: repetitions,
			planDiscoveryRepeats: 3, planConfirmationRepeats: 2, stimulus: stimulus,
			reductionProposalLimit: 2, reductionCandidateLimit: 25,
			reductionWallMS: (3 * time.Minute).Milliseconds(),
		}
	}
	auxRoles := []reference.CLIRole{reference.ConfigFirst, reference.ArgvFirst}
	auxPhase := func(label string, purpose domain.AttemptPurpose, stimulus *countercli.CLIStimulus) phaseRunSpec {
		return phaseRunSpec{
			label: label, purpose: purpose, repetitions: 1, planDiscoveryRepeats: 1, planConfirmationRepeats: 1,
			roles: auxRoles, stimulus: stimulus, reductionProposalLimit: 2, reductionCandidateLimit: 6,
			reductionWallMS: (3 * time.Minute).Milliseconds(),
		}
	}
	discovery, err := runPhaseWithScope(ctx, config, scope, mainPhase("main-decisive", domain.AttemptDiscovery, 3, nil))
	if err != nil || validateStableDivergence(discovery, 3, 3, domain.AttemptDiscovery) != nil {
		return CompletedCLIStudy{}, errors.Join(err, fmt.Errorf("CLI_STUDY_MAIN_DECISIVE_REFUSED"))
	}
	shortStimulus, err := shortEligibilityStimulus(discovery.Stimulus)
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	eligibilityReferenceSpec := mainPhase("main-eligibility-reference", domain.AttemptDiscovery, 3, &shortStimulus)
	eligibilityReferenceSpec.stdoutBytes = 30
	eligibilityDriftSpec := mainPhase("main-eligibility-drift", domain.AttemptDiscovery, 3, &discovery.Stimulus)
	eligibilityDriftSpec.stdoutBytes = 30
	shapeReferenceSpec := phaseRunSpec{
		label: "main-shape-reference", purpose: domain.AttemptDiscovery, repetitions: 1,
		planDiscoveryRepeats: 1, planConfirmationRepeats: 1,
	}
	shapeStimulus, err := stimulusWithConfigMode(discovery.Stimulus, "shape-changed")
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	shapeChangedSpec := phaseRunSpec{
		label: "main-shape-changed", purpose: domain.AttemptReduction, repetitions: 1,
		planDiscoveryRepeats: 1, planConfirmationRepeats: 1, stimulus: &shapeStimulus,
	}
	noisy, err := noisyCLIStimulus(discovery.Stimulus, "z-main-irrelevant.txt")
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	baselineCheckpointSpec := mainPhase("main-baseline-checkpoint", domain.AttemptDiscovery, 3, &noisy)
	baselineSpec := mainPhase("main-divergent-baseline", domain.AttemptDiscovery, 3, &noisy)
	auxiliaryNoisy, err := noisyCLIStimulus(discovery.Stimulus, "z-auxiliary-irrelevant.txt")
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	auxiliaryBaselineSpec := auxPhase("auxiliary-divergent-baseline", domain.AttemptDiscovery, &auxiliaryNoisy)

	var recoveryOne, recoveryTwo Result
	var shapeReference, shapeChanged Result
	var baselineCheckpoint, baselineStudy, auxiliaryBaseline Result
	independentPhases := []cliWorkflowTask{
		func(taskContext context.Context) error {
			var runErr error
			recoveryOne, runErr = runPhaseWithScope(taskContext, config, scope, eligibilityReferenceSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			recoveryTwo, runErr = runPhaseWithScope(taskContext, config, scope, eligibilityDriftSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			shapeReference, runErr = runPhaseWithScope(taskContext, config, scope, shapeReferenceSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			shapeChanged, runErr = runPhaseWithScope(taskContext, config, scope, shapeChangedSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			baselineCheckpoint, runErr = runPhaseWithScope(taskContext, config, scope, baselineCheckpointSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			baselineStudy, runErr = runPhaseWithScope(taskContext, config, scope, baselineSpec)
			return runErr
		},
		func(taskContext context.Context) error {
			var runErr error
			auxiliaryBaseline, runErr = runPhaseWithScope(taskContext, config, scope, auxiliaryBaselineSpec)
			return runErr
		},
	}
	if err := runBoundedCLIWorkflowTasks(ctx, independentCLIPhaseWorkerLimit, independentPhases); err != nil {
		return CompletedCLIStudy{}, errors.Join(err, fmt.Errorf("CLI_STUDY_INDEPENDENT_PHASES_REFUSED"))
	}
	if validateEligibilityPair(recoveryOne, recoveryTwo) != nil ||
		validateFreshDisjoint(discovery, recoveryOne, recoveryTwo) != nil {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_ELIGIBILITY_DRIFT_REFUSED")
	}
	shapeAssessment := compare.AssessPreservation(shapeReference.OutcomeMap, shapeChanged.OutcomeMap)
	if validateStableDivergence(shapeReference, 3, 1, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(shapeChanged, 3, 1, domain.AttemptReduction) != nil ||
		!shapeAssessment.Valid() || shapeAssessment.Relation() != compare.PreservationDifferent ||
		shapeReference.OutcomeMap.DistinctProjectionCount() != shapeChanged.OutcomeMap.DistinctProjectionCount() {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_SHAPE_TRAP_REFUSED")
	}
	if validateStableDivergence(baselineCheckpoint, 3, 3, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(baselineStudy, 3, 3, domain.AttemptDiscovery) != nil ||
		baselineCheckpoint.Plan.Digest() != baselineStudy.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineStudy.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(baselineCheckpoint.OutcomeMap.CanonicalBytes(), baselineStudy.OutcomeMap.CanonicalBytes()) {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_MAIN_BASELINE_REFUSED")
	}
	if validateStableDivergence(auxiliaryBaseline, 2, 1, domain.AttemptDiscovery) != nil {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_AUXILIARY_BASELINE_REFUSED")
	}

	var mainReduction, auxiliaryReduction reductionSequence
	var confirmed, auxiliaryConfirmed Result
	reductionBranches := []cliWorkflowTask{
		func(taskContext context.Context) error {
			var branchErr error
			mainReduction, branchErr = runWeakPhysicalReduction(
				taskContext, config, scope, baselineStudy, mainPhase("", domain.AttemptReduction, 3, nil),
			)
			if branchErr != nil {
				return branchErr
			}
			mainReduced, branchErr := compare.RequireDivergence(mainReduction.minimized.OutcomeMap)
			if branchErr != nil {
				return branchErr
			}
			confirmationSpec := mainPhase("main-confirmation", domain.AttemptConfirmation, 2, &mainReduction.minimized.Stimulus)
			confirmationSpec.confirmation = &confirmationRunInput{
				reducedBaseline: mainReduced, reductionRun: mainReduction.run, reductionResult: mainReduction.grade,
			}
			confirmed, branchErr = runPhaseWithScope(taskContext, config, scope, confirmationSpec)
			if branchErr != nil || validateConfirmation(mainReduction.minimized, confirmed, 3, 2) != nil {
				return errors.Join(branchErr, fmt.Errorf("CLI_STUDY_MAIN_CONFIRMATION_REFUSED"))
			}
			return nil
		},
		func(taskContext context.Context) error {
			var branchErr error
			auxiliaryReduction, branchErr = runStrongPhysicalReduction(
				taskContext, config, scope, auxiliaryBaseline, auxPhase("", domain.AttemptReduction, nil),
			)
			if branchErr != nil {
				return branchErr
			}
			auxReduced, branchErr := compare.RequireDivergence(auxiliaryReduction.minimized.OutcomeMap)
			if branchErr != nil {
				return branchErr
			}
			confirmationSpec := auxPhase("auxiliary-confirmation", domain.AttemptConfirmation, &auxiliaryReduction.minimized.Stimulus)
			confirmationSpec.confirmation = &confirmationRunInput{
				reducedBaseline: auxReduced, reductionRun: auxiliaryReduction.run, reductionResult: auxiliaryReduction.grade,
			}
			auxiliaryConfirmed, branchErr = runPhaseWithScope(taskContext, config, scope, confirmationSpec)
			if branchErr != nil || validateConfirmation(auxiliaryReduction.minimized, auxiliaryConfirmed, 2, 1) != nil {
				return errors.Join(branchErr, fmt.Errorf("CLI_STUDY_AUXILIARY_CONFIRMATION_REFUSED"))
			}
			return nil
		},
	}
	if err := runBoundedCLIWorkflowTasks(ctx, 2, reductionBranches); err != nil {
		return CompletedCLIStudy{}, errors.Join(err, fmt.Errorf("CLI_STUDY_REDUCTION_BRANCH_REFUSED"))
	}

	controls, overlayFacts, err := runSemanticControls(ctx, config, scope, discovery.Stimulus)
	if err != nil || len(controls) != 16 {
		return CompletedCLIStudy{}, errors.Join(err, fmt.Errorf("CLI_STUDY_CONTROL_ROSTER_REFUSED"))
	}
	if err := scope.Close(); err != nil {
		return CompletedCLIStudy{}, fmt.Errorf("CLI_STUDY_SCOPE_CLOSE_REFUSED: %w", err)
	}
	scopeOpen = false

	authorityRoot, err := privateTempDirectory(config.ScratchRoot, fmt.Sprintf("u7c-%d-authorities-", config.Ordinal))
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	choicepoint, decision, ruling, source, bundle, residue, choiceFacts, objectStore, err := completeChoiceAndBundle(
		ctx, authorityRoot, baselineCheckpoint, mainReduction.baseline, baselineStudy, mainReduction.run, confirmed,
	)
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	officialTrials, err := executeOfficialTrials(ctx, config, authorityRoot, objectStore, residue, bundle)
	if err != nil {
		return CompletedCLIStudy{}, err
	}

	phaseFacts, err := buildPhaseFacts(
		discovery, recoveryOne, recoveryTwo, shapeReference, shapeChanged,
		baselineCheckpoint, baselineStudy, mainReduction.evaluation,
		auxiliaryBaseline, auxiliaryReduction.evaluation, controls,
		confirmed, auxiliaryConfirmed, officialTrials,
	)
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	completed = CompletedCLIStudy{
		ordinal: config.Ordinal, discovery: discovery, recoveryOne: recoveryOne, recoveryTwo: recoveryTwo,
		shapeReference: shapeReference, shapeChanged: shapeChanged,
		baselineCheckpoint: baselineCheckpoint, baseline: baselineStudy,
		minimized: mainReduction.minimized, confirmed: confirmed, mainEvaluation: mainReduction.evaluation,
		auxiliaryBaseline: auxiliaryBaseline, auxiliaryEvaluation: auxiliaryReduction.evaluation,
		auxiliaryMinimized: auxiliaryReduction.minimized, auxiliaryConfirmed: auxiliaryConfirmed,
		weakRun: mainReduction.run, weakGrade: mainReduction.grade,
		strongRun: auxiliaryReduction.run, strongGrade: auxiliaryReduction.grade,
		choicepoint: choicepoint, decision: decision, durableRuling: ruling, portableSource: source,
		bundle: bundle, residue: residue, choiceAudit: choiceFacts, officialTrials: officialTrials,
		semanticControls: controls, overlayAudit: overlayFacts, phaseFacts: phaseFacts,
	}
	completed.physicalRunAuthority, err = physicalRunAuthority(completed)
	if err != nil {
		return CompletedCLIStudy{}, err
	}
	completed.seal = issuedCompletedCLIStudy
	if err := validateCompletedCLIStudyConstruction(completed); err != nil {
		return CompletedCLIStudy{}, err
	}
	if err := revalidateCompletedOfficialTrials(
		ctx, config, authorityRoot, objectStore, residue, bundle, completed,
	); err != nil {
		return CompletedCLIStudy{}, err
	}
	return completed, nil
}

func runBoundedCLIWorkflowTasks(ctx context.Context, limit int, tasks []cliWorkflowTask) error {
	if ctx == nil || limit < 1 || len(tasks) < 1 {
		return fmt.Errorf("CLI_STUDY_TASK_SET_REFUSED")
	}
	for _, task := range tasks {
		if task == nil {
			return fmt.Errorf("CLI_STUDY_TASK_SET_REFUSED")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	taskContext, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	outcomes := make([]error, len(tasks))
	jobs := make(chan int, len(tasks))
	for index := range tasks {
		jobs <- index
	}
	close(jobs)

	workerCount := limit
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				if taskContext.Err() != nil {
					continue
				}
				outcomes[index] = tasks[index](taskContext)
				if outcomes[index] != nil {
					cancel(outcomes[index])
				}
			}
		}()
	}
	workers.Wait()

	for _, outcome := range outcomes {
		if outcome != nil && !errors.Is(outcome, context.Canceled) && !errors.Is(outcome, context.DeadlineExceeded) {
			return outcome
		}
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	if cause := context.Cause(taskContext); cause != nil {
		return cause
	}
	for _, outcome := range outcomes {
		if outcome != nil {
			return outcome
		}
	}
	return nil
}

func noisyCLIStimulus(base countercli.CLIStimulus, paths ...string) (countercli.CLIStimulus, error) {
	fixtures := base.Fixtures()
	for _, path := range paths {
		noise, err := countercli.NewFixtureFile(path, []byte("not-consumed\n"), countercli.FixtureMode0644)
		if err != nil {
			return countercli.CLIStimulus{}, err
		}
		fixtures = append(fixtures, noise)
	}
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: base.Executable(), BaseArgv: base.BaseArgv(), Argv: base.Argv(), Stdin: base.Stdin(),
		Environment: base.Environment(), Fixtures: fixtures, CWDPolicy: base.CWDPolicy(),
	})
}

func shortEligibilityStimulus(base countercli.CLIStimulus) (countercli.CLIStimulus, error) {
	environment, err := countercli.PresentEnvironment("APP_MODE", "e")
	if err != nil {
		return countercli.CLIStimulus{}, err
	}
	fixture, err := countercli.NewFixtureFile("config.json", []byte(`{"mode":"c"}`), countercli.FixtureMode0644)
	if err != nil {
		return countercli.CLIStimulus{}, err
	}
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: base.Executable(), BaseArgv: base.BaseArgv(), Argv: []string{"--mode", "a"},
		Stdin: base.Stdin(), Environment: []countercli.CLIEnvironmentBinding{environment},
		Fixtures: []countercli.CLIFixtureFile{fixture}, CWDPolicy: base.CWDPolicy(),
	})
}

func stimulusWithArgv(base countercli.CLIStimulus, argv []string) (countercli.CLIStimulus, error) {
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: base.Executable(), BaseArgv: base.BaseArgv(), Argv: append([]string(nil), argv...),
		Stdin: base.Stdin(), Environment: base.Environment(), Fixtures: base.Fixtures(), CWDPolicy: base.CWDPolicy(),
	})
}

func stimulusWithConfigMode(base countercli.CLIStimulus, mode string) (countercli.CLIStimulus, error) {
	fixtures := base.Fixtures()
	if len(fixtures) != 1 || fixtures[0].Path() != "config.json" || mode == "" {
		return countercli.CLIStimulus{}, fmt.Errorf("CLI_STUDY_SHAPE_STIMULUS_REFUSED")
	}
	config, err := countercli.NewFixtureFile(
		"config.json", []byte(fmt.Sprintf(`{"mode":%q}`, mode)), countercli.FixtureMode0644,
	)
	if err != nil {
		return countercli.CLIStimulus{}, err
	}
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: base.Executable(), BaseArgv: base.BaseArgv(), Argv: base.Argv(), Stdin: base.Stdin(),
		Environment: base.Environment(), Fixtures: []countercli.CLIFixtureFile{config}, CWDPolicy: base.CWDPolicy(),
	})
}

func stimulusWithInputs(base countercli.CLIStimulus, argv []string, stdin countercli.CLIStdin, environment []countercli.CLIEnvironmentBinding) (countercli.CLIStimulus, error) {
	return countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: base.Executable(), BaseArgv: base.BaseArgv(), Argv: append([]string(nil), argv...),
		Stdin: stdin, Environment: append([]countercli.CLIEnvironmentBinding(nil), environment...),
		Fixtures: base.Fixtures(), CWDPolicy: base.CWDPolicy(),
	})
}

func runWeakPhysicalReduction(ctx context.Context, config Config, scope *cliWorkflowScope, baselineStudy Result, evaluationPhase phaseRunSpec) (reductionSequence, error) {
	return runPhysicalReduction(ctx, config, scope, "main", baselineStudy, evaluationPhase, false)
}

func runStrongPhysicalReduction(ctx context.Context, config Config, scope *cliWorkflowScope, baselineStudy Result, evaluationPhase phaseRunSpec) (reductionSequence, error) {
	return runPhysicalReduction(ctx, config, scope, "auxiliary", baselineStudy, evaluationPhase, true)
}

func runPhysicalReduction(
	ctx context.Context,
	config Config,
	scope *cliWorkflowScope,
	label string,
	baselineStudy Result,
	evaluationPhase phaseRunSpec,
	wantStrong bool,
) (reductionSequence, error) {
	baseline, err := compare.RequireDivergence(baselineStudy.OutcomeMap)
	if err != nil {
		return reductionSequence{}, err
	}
	rules := []countercli.CLIReducerID{countercli.CLIFixtureRemove}
	if !wantStrong {
		rules = append(rules, countercli.CLIEnvironmentRemove)
	}
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baselineStudy.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: rules,
	})
	if err != nil {
		return reductionSequence{}, err
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineStudy.Plan)
	if err != nil {
		return reductionSequence{}, err
	}
	evaluations := make([]Result, 0, 1)
	run, err := reducer.Run(ctx, reducer.RunInput[countercli.CLIStimulus]{
		Original: baselineStudy.Stimulus,
		Reference: func(stimulus countercli.CLIStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := countercli.MeasureCLIStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus countercli.CLIStimulus) ([]reducer.TypedProposal[countercli.CLIStimulus], error) {
			neighbors, enumerateErr := countercli.EnumerateCLINeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[countercli.CLIStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[countercli.CLIStimulus]{Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor()}
			}
			return proposals, nil
		},
		Evaluate: func(runContext context.Context, stimulus countercli.CLIStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			required := uint64(evaluationPhase.repetitions * len(evaluationPhase.roles))
			if len(evaluationPhase.roles) == 0 {
				required = uint64(evaluationPhase.repetitions * len(reference.CLIRoles()))
			}
			if allowance.RemainingCandidateTrials < required || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			phase := evaluationPhase
			phase.label = label + "-accepted-reduction"
			phase.purpose = purpose
			phase.stimulus = &stimulus
			evaluationContext, cancel := context.WithDeadline(runContext, allowance.WallDeadline)
			defer cancel()
			result, runErr := runPhaseWithScope(evaluationContext, config, scope, phase)
			if runErr != nil {
				return reducer.EvaluationObservation{}, runErr
			}
			evaluations = append(evaluations, result)
			outcome := result.OutcomeMap
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil || len(evaluations) != 1 || !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown ||
		!run.HasAcceptedReduction() || len(run.Transcript().AcceptedPath()) != 1 ||
		evaluations[0].Stimulus.Digest() != run.MinimizedStimulusDigest() {
		return reductionSequence{}, errors.Join(err, fmt.Errorf("CLI_STUDY_REDUCTION_REFUSED"))
	}
	entries := run.Transcript().Entries()
	if len(entries) == 0 {
		return reductionSequence{}, fmt.Errorf("CLI_STUDY_REDUCTION_TRANSCRIPT_REFUSED")
	}
	evaluation := entries[0].Evaluation()
	if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != reducer.Preserves ||
		!evaluation.LogicalNonReuseWithBaseline() || len(evaluation.ObservedAttemptDigests()) != len(evaluations[0].Trials) {
		return reductionSequence{}, fmt.Errorf("CLI_STUDY_REDUCTION_EVIDENCE_REFUSED")
	}
	if !wantStrong {
		if len(entries) != 2 || entries[1].Evaluation().Decision() != reducer.Unresolved ||
			entries[1].Evaluation().ReasonCode() != string(reducer.ReasonEvaluatorError) ||
			entries[0].Evaluation().Neighbor().Rule().Name() != string(countercli.CLIFixtureRemove) ||
			entries[1].Evaluation().Neighbor().Rule().Name() != string(countercli.CLIEnvironmentRemove) ||
			entries[0].CandidateTrials() != 9 || entries[0].TrialCount() != 9 ||
			entries[1].CandidateTrials() != 0 || entries[1].TrialCount() != 9 ||
			run.Budget().ProposalLimit() != 2 || run.Budget().CandidateTrialLimit() != 10 ||
			run.Transcript().FinalSweepState() == reducer.FinalSweepComplete {
			return reductionSequence{}, fmt.Errorf("CLI_STUDY_WEAK_UNRESOLVED_REFUSED")
		}
		if _, present, draftErr := run.CompletedSweepDraft(); draftErr != nil || present {
			return reductionSequence{}, errors.Join(draftErr, fmt.Errorf("CLI_STUDY_WEAK_SWEEP_REFUSED"))
		}
		weak, finalizeErr := grade.Finalize(ctx, run, nil)
		if finalizeErr != nil || !weak.Valid() || weak.Grade().Status() != grade.StatusBestKnown ||
			weak.RunDigest() != run.Digest() {
			return reductionSequence{}, errors.Join(finalizeErr, fmt.Errorf("CLI_STUDY_WEAK_GRADE_REFUSED"))
		}
		return reductionSequence{
			baseline: baseline, run: run, grade: weak, minimized: evaluations[0], evaluation: evaluations[0],
			policyDigest: policy.Digest(),
		}, nil
	}
	if len(entries) != 1 || run.Transcript().FinalSweepState() != reducer.FinalSweepComplete {
		return reductionSequence{}, fmt.Errorf("CLI_STUDY_STRONG_TRANSCRIPT_REFUSED")
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 {
		return reductionSequence{}, errors.Join(err, fmt.Errorf("CLI_STUDY_SWEEP_DRAFT_REFUSED"))
	}
	sweepRoot, err := privateTempDirectory(config.ScratchRoot, fmt.Sprintf("u7c-%d-%s-sweep-", config.Ordinal, label))
	if err != nil {
		return reductionSequence{}, err
	}
	sweepStore, err := store.OpenReductionSweepStore(sweepRoot)
	if err != nil {
		return reductionSequence{}, err
	}
	authority, err := sweepStore.Publish(ctx, draft)
	if err != nil {
		return reductionSequence{}, err
	}
	strong, err := grade.Finalize(ctx, run, &grade.SweepCompletion{Store: sweepStore, Draft: draft, Authority: authority})
	if err != nil || !strong.Valid() || strong.Grade().Status() != grade.StatusOneMinimalUnder {
		return reductionSequence{}, errors.Join(err, fmt.Errorf("CLI_STUDY_STRONG_GRADE_REFUSED"))
	}
	return reductionSequence{
		baseline: baseline, run: run, grade: strong,
		minimized: evaluations[0], evaluation: evaluations[0], policyDigest: policy.Digest(), sweepDigest: draft.Digest(),
	}, nil
}

func runSemanticControls(ctx context.Context, config Config, scope *cliWorkflowScope, base countercli.CLIStimulus) ([]physicalControl, overlayControlAudit, error) {
	overlay, err := validateOverlayConstructorRefusal()
	if err != nil {
		return nil, overlayControlAudit{}, err
	}
	inputs, err := expectedSemanticControlInputs(base)
	if err != nil {
		return nil, overlayControlAudit{}, err
	}
	controls := make([]physicalControl, len(inputs))
	tasks := make([]cliWorkflowTask, len(inputs))
	for index := range inputs {
		index := index
		tasks[index] = func(taskContext context.Context) error {
			input := inputs[index]
			stimulus, stimulusErr := stimulusWithInputs(base, input.argv, input.stdin, input.env)
			if stimulusErr != nil {
				return stimulusErr
			}
			control, runErr := runSingleControl(
				taskContext, config, scope, input.label, input.kind, reference.ArgvFirst, stimulus, input.fields,
			)
			if runErr != nil {
				return runErr
			}
			if validationErr := validatePhysicalControl(control); validationErr != nil {
				return validationErr
			}
			controls[index] = control
			return nil
		}
	}
	if err := runBoundedCLIWorkflowTasks(ctx, semanticCLIControlWorkerLimit, tasks); err != nil {
		return nil, overlayControlAudit{}, err
	}
	return controls, overlay, nil
}

func expectedSemanticControlInputs(base countercli.CLIStimulus) ([]semanticControlInput, error) {
	if !base.Valid() {
		return nil, fmt.Errorf("CLI_STUDY_CONTROL_BASE_REFUSED")
	}
	appAbsent, err := countercli.AbsentEnvironment("APP_MODE")
	if err != nil {
		return nil, err
	}
	appEmpty, err := countercli.PresentEnvironment("APP_MODE", "")
	if err != nil {
		return nil, err
	}
	appMode, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		return nil, err
	}
	unusedAlpha, err := countercli.PresentEnvironment("A_UNUSED", "alpha")
	if err != nil {
		return nil, err
	}
	unusedOmega, err := countercli.PresentEnvironment("Z_UNUSED", "omega")
	if err != nil {
		return nil, err
	}
	presentEmpty, err := countercli.PresentStdin([]byte{})
	if err != nil {
		return nil, err
	}
	standardFields := []countercli.CLIFieldID{
		countercli.CLIFieldExitCode, countercli.CLIFieldStdoutJSONMode, countercli.CLIFieldStdoutJSONSource,
	}
	exitFields := []countercli.CLIFieldID{countercli.CLIFieldExitCode}
	jsonFields := []countercli.CLIFieldID{countercli.CLIFieldStdoutJSONMode, countercli.CLIFieldStdoutJSONSource}
	return []semanticControlInput{
		{"CONTROL_STDIN_ABSENT", "control-stdin-absent", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_STDIN_PRESENT_EMPTY", "control-stdin-present-empty", []string{"--mode", "argv"}, presentEmpty, []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_APP_MODE_ABSENT", "control-app-mode-absent", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appAbsent}, standardFields},
		{"CONTROL_APP_MODE_PRESENT_EMPTY", "control-app-mode-present-empty", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appEmpty}, standardFields},
		{"CONTROL_ORDERED_ARGV_MODE_THEN_STDERR", "control-ordered-argv-mode-stderr", []string{"--mode", "argv", "--stderr", "ordered"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_ORDERED_ARGV_STDERR_THEN_MODE", "control-ordered-argv-stderr-mode", []string{"--stderr", "ordered", "--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_SPARSE_IRRELEVANT_ENV_ALPHA", "control-sparse-env-alpha", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode, unusedAlpha}, standardFields},
		{"CONTROL_SPARSE_IRRELEVANT_ENV_OMEGA", "control-sparse-env-omega", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode, unusedOmega}, standardFields},
		{"CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION", "control-default-exit-zero-home", []string{"--mode", "argv"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, exitFields},
		{"CONTROL_UNSELECTED_STDERR_ALPHA", "control-stderr-alpha", []string{"--mode", "argv", "--stderr", "alpha"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_UNSELECTED_STDERR_BETA", "control-stderr-beta", []string{"--mode", "argv", "--stderr", "beta"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, standardFields},
		{"CONTROL_EXIT_SEVEN_SUBSTRATE", "control-exit-seven-substrate", []string{"--mode", "argv", "--behavior", "nonzero-exit"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, exitFields},
		{"CONTROL_SIGNAL_SELECTED_EXIT_MISSING", "control-signal-exit-missing", []string{"--behavior", "signal"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, exitFields},
		{"CONTROL_TIMEOUT", "control-timeout", []string{"--behavior", "timeout"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, exitFields},
		{"CONTROL_MALFORMED_SELECTED_JSON", "control-malformed-json", []string{"--behavior", "malformed-projection"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, jsonFields},
		{"CONTROL_OUTPUT_LIMIT", "control-output-limit", []string{"--behavior", "output-limit", "--emit-bytes", "131072"}, countercli.AbsentStdin(), []countercli.CLIEnvironmentBinding{appMode}, exitFields},
	}, nil
}

func validateOverlayConstructorRefusal() (overlayControlAudit, error) {
	_, overlayErr := countercli.NewFixtureFile("../escape", []byte("refused\n"), countercli.FixtureMode0644)
	var adapterErr *climodel.Refusal
	if !errors.As(overlayErr, &adapterErr) || adapterErr.Code != countercli.CodeFixturePath {
		return overlayControlAudit{}, fmt.Errorf("CLI_STUDY_OVERLAY_REFUSAL_MISSING: %v", overlayErr)
	}
	return overlayControlAudit{refusalCode: adapterErr.Code}, nil
}

func runSingleControl(
	ctx context.Context,
	config Config,
	scope *cliWorkflowScope,
	label, kind string,
	role reference.CLIRole,
	stimulus countercli.CLIStimulus,
	fields []countercli.CLIFieldID,
) (control physicalControl, returnErr error) {
	runRoot, err := privateRunRoot(config.ScratchRoot, config.Ordinal, label)
	if err != nil {
		return physicalControl{}, err
	}
	authority, err := scope.candidateAuthority(reference.CLIRoles())
	if err != nil {
		return physicalControl{}, err
	}
	capture, err := countercli.NewCLICapturePolicy(countercli.CLICapturePolicyConfig{StdoutBytes: 64 << 10, StderrBytes: 64 << 10})
	if err != nil {
		return physicalControl{}, err
	}
	projection, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{Fields: append([]countercli.CLIFieldID(nil), fields...)})
	if err != nil {
		return physicalControl{}, err
	}
	fixtureRecipe, err := countercli.NewCLIFixtureRecipe()
	if err != nil {
		return physicalControl{}, err
	}
	envelope, err := newStudyEnvelope()
	if err != nil {
		return physicalControl{}, err
	}
	runnerDigest, err := runnerprofile.CLIDigest()
	if err != nil {
		return physicalControl{}, err
	}
	plan, _, _, err := compileStudyPlan(studyPlanInput{
		candidateSetDigest: authority.declaration.Digest(), materializationPolicyDigest: scope.materializationPolicy.Digest(),
		comparisonEnvelopeDigest: envelope.Digest(), runnerDigest: runnerDigest,
		startArgv: stimulus.BaseLogicalArgv(), fixtureRecipeDigest: fixtureRecipe.Digest(), capturePolicy: capture,
		projectionDefinition: projection.Binding(), discoveryRepeats: 1, confirmationRepeats: 1,
		candidateCount: len(reference.CLIRoles()),
	})
	if err != nil {
		return physicalControl{}, err
	}
	binding, err := countercli.BindExecution(plan, stimulus, capture, projection)
	if err != nil {
		return physicalControl{}, err
	}
	bound, err := authority.declaration.Bind(plan)
	if err != nil || len(bound) != len(reference.CLIRoles()) {
		return physicalControl{}, errors.Join(err, fmt.Errorf("CLI_STUDY_CONTROL_BINDING_REFUSED"))
	}
	var selectedCandidate gitobj.BoundCandidate
	for _, candidate := range bound {
		if authority.roleByTree[candidate.TreeIdentityDigest()] == role {
			selectedCandidate = candidate
		}
	}
	if !selectedCandidate.Valid() {
		return physicalControl{}, fmt.Errorf("CLI_STUDY_CONTROL_ROLE_REFUSED")
	}
	allocationRoot, err := createPrivateDirectory(runRoot, "attempts")
	if err != nil {
		return physicalControl{}, err
	}
	physical, err := world.ExecuteCLI(ctx, world.CLIRequest{
		Binding: binding, Candidate: selectedCandidate, Tools: scope.tools, AllocationRoot: allocationRoot,
		Purpose: domain.AttemptDiscovery, InstanceNonce: fmt.Sprintf("u7c-%d-control-%s", config.Ordinal, label), ScheduleOrdinal: 0,
	})
	if err != nil {
		return physicalControl{}, err
	}
	measurements, err := studyMeasurements(envelope, physical)
	if err != nil {
		return physicalControl{}, err
	}
	observation, err := countercli.AdaptWorldResult(physical, binding, 0)
	if err != nil {
		return physicalControl{}, err
	}
	control = physicalControl{
		kind: kind, role: role, root: runRoot, plan: plan, envelope: envelope, stimulus: stimulus, binding: binding,
		definition: projection, result: physical, measurements: measurements, observation: observation,
	}
	if physical.FinalizedAttempt().HasControls() {
		return control, nil
	}
	projected, projectErr := projection.Project(observation)
	if projectErr == nil {
		control.projection = projected
		control.projected = true
		return control, nil
	}
	var rejection *countercli.ProjectionRejection
	if !errors.As(projectErr, &rejection) {
		return physicalControl{}, projectErr
	}
	control.projectionRejection = rejection
	return control, nil
}

func validateStudyEnvelope(envelope domain.ComparisonEnvelope) error {
	expected, err := newStudyEnvelope()
	if err != nil || !envelope.Digest().Valid() || envelope.Digest() != expected.Digest() {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_ENVELOPE_AUTHORITY_REFUSED"))
	}
	config := envelope.Config()
	measuredMatches, toleratedMatches := 0, 0
	for _, dimension := range config.Measured {
		if dimension.Name == dimensionLogicalArgv {
			measuredMatches++
			if dimension.Source != domain.MeasuredProcessReceipt || dimension.Comparison != domain.CompareRecordedOnly {
				return fmt.Errorf("CLI_STUDY_LOGICAL_ARGV_DISPOSITION_REFUSED")
			}
		}
	}
	for _, dimension := range config.RequiredEqual {
		if dimension.Name == dimensionLogicalArgv {
			return fmt.Errorf("CLI_STUDY_LOGICAL_ARGV_DISPOSITION_REFUSED")
		}
	}
	for _, dimension := range config.Tolerated {
		if dimension.Name == dimensionLogicalArgv {
			toleratedMatches++
			if dimension.Tolerance != domain.MayDifferRecorded {
				return fmt.Errorf("CLI_STUDY_LOGICAL_ARGV_DISPOSITION_REFUSED")
			}
		}
	}
	if measuredMatches != 1 || toleratedMatches != 1 {
		return fmt.Errorf("CLI_STUDY_LOGICAL_ARGV_DISPOSITION_REFUSED")
	}
	return nil
}

func validatePhysicalControl(control physicalControl) error {
	if control.kind == "" || control.role != reference.ArgvFirst || !control.plan.Digest().Valid() ||
		validateStudyEnvelope(control.envelope) != nil || !control.stimulus.Valid() ||
		!control.binding.Valid() || !control.definition.Valid() ||
		control.binding.PlanDigest() != control.plan.Digest() ||
		control.binding.StimulusDigest() != control.stimulus.Digest() ||
		control.binding.CapturePolicyDigest() != control.plan.CapturePolicyDigest() ||
		control.binding.ProjectionDefinitionBinding().Digest() != control.plan.ProjectionDefinitionDigest() ||
		control.definition.Binding().Digest() != control.plan.ProjectionDefinitionDigest() {
		return fmt.Errorf("CLI_STUDY_CONTROL_SHAPE_REFUSED: %s", control.kind)
	}
	instance := control.result.World()
	attempt := control.result.FinalizedAttempt()
	if !instance.Digest().Valid() || !attempt.ArtifactDigest().Valid() ||
		instance.PlanDigest() != control.plan.Digest() || instance.EnvelopeDigest() != control.envelope.Digest() ||
		instance.StimulusDigest() != control.stimulus.Digest() ||
		instance.CapturePolicyDigest() != control.binding.CapturePolicyDigest() ||
		instance.ProjectionDefinitionDigest() != control.definition.Binding().Digest() ||
		instance.AttemptArtifactDigest() != attempt.ArtifactDigest() || instance.Purpose() != domain.AttemptDiscovery ||
		instance.RequiredFreshTrials() != 1 || instance.ScheduleOrdinal() != 0 ||
		!control.measurements.Digest().Valid() || control.measurements.SubjectDigest() != instance.Digest() ||
		!control.observation.Valid() || control.observation.WorldDigest() != instance.Digest() ||
		control.observation.PlanDigest() != control.plan.Digest() ||
		control.observation.CandidateKey() != instance.CandidateKey() ||
		control.observation.StimulusDigest() != control.stimulus.Digest() ||
		control.observation.ExecutionPayloadDigest() != control.binding.ExecutionPayloadDigest() ||
		control.observation.AttemptDigest() != attempt.ArtifactDigest() || control.observation.TrialIndex() != 0 ||
		control.observation.CapturePolicyDigest() != control.binding.CapturePolicyDigest() ||
		control.observation.ProjectionDefinitionDigest() != control.definition.Binding().Digest() {
		return fmt.Errorf("CLI_STUDY_CONTROL_AUTHORITY_REFUSED: %s", control.kind)
	}
	if err := validateProjectionLineage(
		control.definition, control.observation, control.projected, control.projection, control.projectionRejection,
	); err != nil && !attempt.HasControls() {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_CONTROL_PROJECTION_LINEAGE_REFUSED: %s", control.kind))
	}
	switch control.kind {
	case "CONTROL_TIMEOUT", "CONTROL_OUTPUT_LIMIT":
		if control.projected || control.projectionRejection != nil ||
			!control.result.Process().PhysicalExecutionEntered() || !control.result.FinalizedAttempt().HasControls() {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	case "CONTROL_MALFORMED_SELECTED_JSON":
		if control.projected || control.projectionRejection == nil || control.result.FinalizedAttempt().HasControls() {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	case "CONTROL_SIGNAL_SELECTED_EXIT_MISSING":
		if !control.projected || control.result.FinalizedAttempt().HasControls() || !control.observation.ProjectionEligible() ||
			!projectedExitMatches(control.projection, countercli.ExactMissing, 0) {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	case "CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION":
		if !control.projected || !projectedExitMatches(control.projection, countercli.ExactInteger, 0) ||
			!controlRootsIsolated(control) {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	case "CONTROL_EXIT_SEVEN_SUBSTRATE":
		if !control.projected || !projectedExitMatches(control.projection, countercli.ExactInteger, 7) {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	default:
		if !control.projected || control.result.FinalizedAttempt().HasControls() {
			return fmt.Errorf("CLI_STUDY_CONTROL_EXPECTATION_REFUSED: %s", control.kind)
		}
	}
	return nil
}

func validateProjectionLineage(
	definition countercli.CLIProjectionDefinition,
	observation countercli.CLICapturedObservation,
	projected bool,
	projection countercli.CLIProjectionResult,
	rejection *countercli.ProjectionRejection,
) error {
	if !definition.Valid() || !observation.Valid() {
		return fmt.Errorf("CLI_STUDY_PROJECTION_LINEAGE_REFUSED")
	}
	if projected {
		derivation := projection.Derivation()
		replayed, err := definition.Project(observation)
		if err != nil || rejection != nil || !derivation.Valid() ||
			derivation.ObservationDigest() != observation.Digest() ||
			derivation.DefinitionDigest() != definition.Binding().Digest() ||
			derivation.AdapterDefinitionDigest() != definition.Digest() ||
			replayed.DerivationDigest() != projection.DerivationDigest() ||
			!bytes.Equal(replayed.ProjectionBytes(), projection.ProjectionBytes()) ||
			!bytes.Equal(replayed.CanonicalBytes(), projection.CanonicalBytes()) {
			return errors.Join(err, fmt.Errorf("CLI_STUDY_PROJECTED_LINEAGE_REFUSED"))
		}
		return nil
	}
	if rejection == nil {
		return fmt.Errorf("CLI_STUDY_PROJECTION_AUTHORITY_ABSENT")
	}
	_, err := definition.Project(observation)
	var replayed *countercli.ProjectionRejection
	if !errors.As(err, &replayed) || !rejection.Valid() || !replayed.Valid() ||
		rejection.ObservationDigest() != observation.Digest() ||
		rejection.DefinitionDigest() != definition.Binding().Digest() ||
		replayed.EvidenceDigest() != rejection.EvidenceDigest() ||
		!bytes.Equal(replayed.CanonicalBytes(), rejection.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_REJECTION_LINEAGE_REFUSED"))
	}
	return nil
}

func controlRootsIsolated(control physicalControl) bool {
	roots := control.result.Roots()
	want := map[string]string{
		"HOME": roots.Home(), "TMPDIR": roots.Temporary(),
		"XDG_CONFIG_HOME": roots.XDGConfig(), "XDG_CACHE_HOME": roots.XDGCache(),
		"XDG_DATA_HOME": roots.XDGData(), "XDG_STATE_HOME": roots.XDGState(),
	}
	seenRoots := make(map[string]struct{}, len(want))
	for _, root := range want {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return false
		}
		if _, duplicate := seenRoots[root]; duplicate {
			return false
		}
		seenRoots[root] = struct{}{}
	}
	seenNames := make(map[string]struct{}, len(want))
	for _, binding := range control.result.Process().Environment() {
		name, value, present := strings.Cut(binding, "=")
		if !present {
			return false
		}
		expected, tracked := want[name]
		if !tracked {
			continue
		}
		if value != expected {
			return false
		}
		if _, duplicate := seenNames[name]; duplicate {
			return false
		}
		seenNames[name] = struct{}{}
	}
	return len(seenNames) == len(want)
}

func projectedExitMatches(projection countercli.CLIProjectionResult, tag countercli.CLIExactValueTag, integer int64) bool {
	fields := projection.Fields()
	if len(fields) != 1 || fields[0].ID() != countercli.CLIFieldExitCode || fields[0].Value().Tag() != tag {
		return false
	}
	if tag == countercli.ExactMissing {
		return true
	}
	value, ok := fields[0].Value().Integer()
	return ok && value == integer
}

func validateEligibilityPair(referenceStudy, driftStudy Result) error {
	if validateStableDivergence(referenceStudy, 3, 3, domain.AttemptDiscovery) != nil ||
		!driftStudy.HasOutcomeMap || driftStudy.OutcomeMap.Phase() != domain.AttemptDiscovery ||
		len(driftStudy.Trials) != 9 || len(driftStudy.OutcomeMap.Entries()) != 1 ||
		len(driftStudy.OutcomeMap.Exclusions()) != 2 ||
		referenceStudy.Plan.Digest() != driftStudy.Plan.Digest() ||
		referenceStudy.CapturePolicy.Digest() != driftStudy.CapturePolicy.Digest() ||
		referenceStudy.ProjectionDefinition.Digest() != driftStudy.ProjectionDefinition.Digest() ||
		referenceStudy.Envelope.Digest() != driftStudy.Envelope.Digest() ||
		referenceStudy.OutcomeMap.ComparisonBasisDigest() != driftStudy.OutcomeMap.ComparisonBasisDigest() {
		return fmt.Errorf("CLI_STUDY_ELIGIBILITY_PAIR_SHAPE_REFUSED")
	}
	assessment := compare.AssessPreservation(referenceStudy.OutcomeMap, driftStudy.OutcomeMap)
	if !assessment.Valid() || assessment.Relation() != compare.PreservationUnresolved ||
		assessment.ReasonCode() != "CANDIDATE_ELIGIBILITY_OR_ADMISSION_CHANGED" {
		return fmt.Errorf("CLI_STUDY_ELIGIBILITY_ASSESSMENT_REFUSED")
	}
	counts := make(map[reference.CLIRole]int, 3)
	for _, trial := range driftStudy.Trials {
		counts[trial.Role]++
		if !trial.Admitted || !trial.Result.World().Digest().Valid() ||
			!trial.Result.FinalizedAttempt().ArtifactDigest().Valid() || !trial.Observation.Valid() {
			return fmt.Errorf("CLI_STUDY_ELIGIBILITY_TRIAL_REFUSED")
		}
		if trial.Role == reference.EnvironmentFirst {
			if !trial.Projected || trial.ProjectionRejection != nil ||
				trial.Result.FinalizedAttempt().HasControls() || !trial.Observation.ProjectionEligible() {
				return fmt.Errorf("CLI_STUDY_ELIGIBILITY_ENV_REFUSED")
			}
		} else {
			primary, primaryPresent := trial.Result.FinalizedAttempt().PrimaryControl()
			if trial.Projected || trial.ProjectionRejection != nil ||
				!trial.Result.FinalizedAttempt().HasControls() || !primaryPresent || primary != domain.ControlOutputLimit ||
				trial.Observation.ProjectionEligible() {
				return fmt.Errorf("CLI_STUDY_ELIGIBILITY_CONTROL_EXCLUSION_REFUSED")
			}
		}
	}
	if counts[reference.ConfigFirst] != 3 || counts[reference.EnvironmentFirst] != 3 || counts[reference.ArgvFirst] != 3 {
		return fmt.Errorf("CLI_STUDY_ELIGIBILITY_REPEAT_REFUSED")
	}
	for _, exclusion := range driftStudy.OutcomeMap.Exclusions() {
		role, rolePresent := driftStudy.CandidateRoles[exclusion.CandidateKey]
		if !rolePresent || role == reference.EnvironmentFirst || exclusion.Classification != observe.Uncomparable {
			return fmt.Errorf("CLI_STUDY_ELIGIBILITY_CLASSIFICATION_REFUSED")
		}
	}
	return nil
}

func validateStableDivergence(result Result, candidates, repetitions int, purpose domain.AttemptPurpose) error {
	if !result.HasOutcomeMap || result.OutcomeMap.Phase() != purpose || len(result.CandidateBindings) != candidates ||
		len(result.CandidateRoles) != candidates || len(result.Trials) != candidates*repetitions ||
		len(result.OutcomeMap.Entries()) != candidates || len(result.OutcomeMap.Exclusions()) != 0 ||
		result.OutcomeMap.DistinctProjectionCount() != candidates || !result.OutcomeMap.Divergence() {
		return fmt.Errorf("CLI_STUDY_DIVERGENCE_SHAPE_REFUSED")
	}
	seenWorlds := make(map[domain.Digest]struct{}, len(result.Trials))
	seenAttempts := make(map[domain.Digest]struct{}, len(result.Trials))
	seenCaptures := make(map[domain.Digest]struct{}, len(result.Trials))
	stable := make(map[reference.CLIRole][]byte, candidates)
	counts := make(map[reference.CLIRole]int, candidates)
	for _, trial := range result.Trials {
		if !trial.Admitted || !trial.Projected || trial.ProjectionRejection != nil ||
			!trial.Result.World().Digest().Valid() || !trial.Result.FinalizedAttempt().ArtifactDigest().Valid() ||
			!trial.Measurements.Digest().Valid() || !trial.Observation.Valid() || !trial.Projection.DerivationDigest().Valid() ||
			!projectionHasNormalRoleSource(trial.Role, trial.Projection) {
			return fmt.Errorf("CLI_STUDY_DIVERGENCE_TRIAL_REFUSED")
		}
		for digest, ledger := range map[domain.Digest]map[domain.Digest]struct{}{
			trial.Result.World().Digest():                    seenWorlds,
			trial.Result.FinalizedAttempt().ArtifactDigest(): seenAttempts,
			trial.Observation.Digest():                       seenCaptures,
		} {
			if _, duplicate := ledger[digest]; duplicate {
				return fmt.Errorf("CLI_STUDY_PHYSICAL_REUSE_REFUSED")
			}
			ledger[digest] = struct{}{}
		}
		counts[trial.Role]++
		projection := trial.Projection.ProjectionBytes()
		if prior, ok := stable[trial.Role]; ok && !bytes.Equal(prior, projection) {
			return fmt.Errorf("CLI_STUDY_UNSTABLE_ROLE_REFUSED")
		}
		stable[trial.Role] = append([]byte(nil), projection...)
	}
	for role := range stable {
		if counts[role] != repetitions {
			return fmt.Errorf("CLI_STUDY_REPEAT_COUNT_REFUSED")
		}
	}
	return nil
}

func projectionHasNormalRoleSource(role reference.CLIRole, projection countercli.CLIProjectionResult) bool {
	fields := projection.Fields()
	if len(fields) != 3 || fields[0].ID() != countercli.CLIFieldExitCode ||
		fields[1].ID() != countercli.CLIFieldStdoutJSONMode || fields[2].ID() != countercli.CLIFieldStdoutJSONSource {
		return false
	}
	exit, exitOK := fields[0].Value().Integer()
	mode, modeOK := fields[1].Value().String()
	source, sourceOK := fields[2].Value().String()
	wantSource := map[reference.CLIRole]string{
		reference.ConfigFirst: "config", reference.EnvironmentFirst: "env", reference.ArgvFirst: "argv",
	}[role]
	return exitOK && exit == 0 && modeOK && sourceOK && mode != "" && source == wantSource
}

func validateFreshDisjoint(studies ...Result) error {
	ledgers := []map[domain.Digest]struct{}{{}, {}, {}}
	for _, study := range studies {
		if !study.HasOutcomeMap {
			return fmt.Errorf("CLI_STUDY_FRESH_MAP_REFUSED")
		}
		for index, values := range [][]domain.Digest{
			study.OutcomeMap.EvidenceWorldDigests(), study.OutcomeMap.EvidenceAttemptDigests(),
			study.OutcomeMap.EvidenceObservationDigests(),
		} {
			for _, digest := range values {
				if !digest.Valid() {
					return fmt.Errorf("CLI_STUDY_FRESH_DIGEST_REFUSED")
				}
				if _, duplicate := ledgers[index][digest]; duplicate {
					return fmt.Errorf("CLI_STUDY_FRESH_REUSE_REFUSED")
				}
				ledgers[index][digest] = struct{}{}
			}
		}
	}
	return nil
}

func validateConfirmation(minimized, confirmed Result, candidates, repetitions int) error {
	if !confirmed.HasConfirmation || !confirmed.Confirmation.Valid() || !confirmed.HasOutcomeMap ||
		confirmed.OutcomeMap.Phase() != domain.AttemptConfirmation || confirmed.OutcomeMap.ScheduleStartOffset() != 1%candidates ||
		confirmed.OutcomeMap.ScheduleDigest() == minimized.OutcomeMap.ScheduleDigest() ||
		len(confirmed.OutcomeMap.Entries()) != candidates || len(confirmed.OutcomeMap.Exclusions()) != 0 ||
		len(confirmed.Trials) != candidates*repetitions || len(confirmed.Confirmation.Draft().PhysicalFacts()) != candidates*repetitions ||
		confirmed.Confirmation.Assessment().Relation() != compare.PreservationEqual {
		return fmt.Errorf("CLI_STUDY_CONFIRMATION_REFUSED")
	}
	return validateFreshDisjoint(minimized, confirmed)
}

func buildPhaseFacts(
	discovery, recoveryOne, recoveryTwo, shapeReference, shapeChanged Result,
	baselineCheckpoint, baseline, mainEvaluation Result,
	auxiliaryBaseline, auxiliaryEvaluation Result,
	controls []physicalControl,
	confirmed, auxiliaryConfirmed Result,
	official []officialTrial,
) ([]phaseFact, error) {
	facts := make([]phaseFact, 0, 98)
	searchTrial := 0
	appendResult := func(kind string, result Result) error {
		for _, trial := range result.Trials {
			if !trial.Admitted {
				return fmt.Errorf("CLI_STUDY_PHASE_RESULT_REFUSED: %s", kind)
			}
			projection := domain.Digest("")
			if trial.Projected {
				projection = trial.Projection.DerivationDigest()
			} else if trial.ProjectionRejection != nil {
				projection = trial.ProjectionRejection.EvidenceDigest()
			}
			controlledEligibilityDrift := false
			if kind == "MAIN_ELIGIBILITY_DRIFT_WORLD" && !trial.Projected && trial.ProjectionRejection == nil {
				primary, primaryPresent := trial.Result.FinalizedAttempt().PrimaryControl()
				controlledEligibilityDrift = primaryPresent && primary == domain.ControlOutputLimit
			}
			if !projection.Valid() && !controlledEligibilityDrift {
				return fmt.Errorf("CLI_STUDY_PHASE_PROJECTION_REFUSED: %s", kind)
			}
			searchTrial++
			facts = append(facts, phaseFact{
				phase: "search", trial: searchTrial, kind: kind,
				authority: trial.Result.FinalizedAttempt().ArtifactDigest(), world: trial.Result.World().Digest(),
				attempt: trial.Result.FinalizedAttempt().ArtifactDigest(), measurement: trial.Measurements.Digest(),
				capture: trial.Observation.Digest(), projection: projection,
			})
		}
		return nil
	}
	main := []struct {
		kind   string
		result Result
	}{
		{"MAIN_DECISIVE_WORLD", discovery}, {"MAIN_ELIGIBILITY_REFERENCE_WORLD", recoveryOne},
		{"MAIN_ELIGIBILITY_DRIFT_WORLD", recoveryTwo}, {"MAIN_SHAPE_REFERENCE_WORLD", shapeReference},
		{"MAIN_SHAPE_CHANGED_WORLD", shapeChanged}, {"MAIN_BASELINE_CHECKPOINT_WORLD", baselineCheckpoint},
		{"MAIN_DIVERGENT_BASELINE_WORLD", baseline}, {"MAIN_ACCEPTED_REDUCTION_WORLD", mainEvaluation},
		{"AUXILIARY_BASELINE_WORLD", auxiliaryBaseline}, {"AUXILIARY_PRESERVING_EVALUATION_WORLD", auxiliaryEvaluation},
	}
	for _, entry := range main {
		if err := appendResult(entry.kind, entry.result); err != nil {
			return nil, err
		}
	}
	for _, control := range controls {
		projection := domain.Digest("")
		if control.projected {
			projection = control.projection.DerivationDigest()
		} else if control.projectionRejection != nil {
			projection = control.projectionRejection.EvidenceDigest()
		}
		searchTrial++
		facts = append(facts, phaseFact{
			phase: "search", trial: searchTrial, kind: control.kind,
			authority: control.result.FinalizedAttempt().ArtifactDigest(), world: control.result.World().Digest(),
			attempt: control.result.FinalizedAttempt().ArtifactDigest(), measurement: control.measurements.Digest(),
			capture: control.observation.Digest(), projection: projection,
		})
	}
	if searchTrial != 80 {
		return nil, fmt.Errorf("CLI_STUDY_SEARCH_FACT_COUNT_REFUSED: %d", searchTrial)
	}
	confirmTrial := 0
	appendConfirmation := func(kind string, result Result) error {
		physical := result.Confirmation.Draft().PhysicalFacts()
		if len(physical) != len(result.Trials) {
			return fmt.Errorf("CLI_STUDY_CONFIRMATION_FACT_JOIN_REFUSED")
		}
		byAttempt := make(map[domain.Digest]Trial, len(result.Trials))
		for _, trial := range result.Trials {
			byAttempt[trial.Result.FinalizedAttempt().ArtifactDigest()] = trial
		}
		for _, fact := range physical {
			trial, ok := byAttempt[fact.AttemptArtifactDigest()]
			if !ok || fact.WorldDigest() != trial.Result.World().Digest() {
				return fmt.Errorf("CLI_STUDY_CONFIRMATION_FACT_JOIN_REFUSED")
			}
			confirmTrial++
			facts = append(facts, phaseFact{
				phase: "confirm", trial: confirmTrial, kind: kind, authority: fact.Digest(),
				world: fact.WorldDigest(), attempt: fact.AttemptArtifactDigest(), measurement: trial.Measurements.Digest(),
				capture: trial.Observation.Digest(), projection: trial.Projection.DerivationDigest(),
			})
		}
		return nil
	}
	if err := appendConfirmation("MAIN_CONFIRMATION_FACT", confirmed); err != nil {
		return nil, err
	}
	if err := appendConfirmation("AUXILIARY_CONFIRMATION_FACT", auxiliaryConfirmed); err != nil {
		return nil, err
	}
	if confirmTrial != 8 {
		return nil, fmt.Errorf("CLI_STUDY_CONFIRM_FACT_COUNT_REFUSED: %d", confirmTrial)
	}
	for index, trial := range official {
		facts = append(facts, phaseFact{
			phase: "contract", trial: index + 1, kind: "OFFICIAL_DEFAULT_EXIT_ZERO_CLASSIFICATION",
			authority: trial.classificationDigest, attempt: trial.attemptDigest,
		})
	}
	if len(facts) != 98 {
		return nil, fmt.Errorf("CLI_STUDY_PHASE_FACT_COUNT_REFUSED: %d", len(facts))
	}
	seen := make(map[domain.Digest]struct{}, len(facts))
	for _, fact := range facts {
		if fact.kind == "" || !fact.authority.Valid() {
			return nil, fmt.Errorf("CLI_STUDY_PHASE_FACT_REFUSED")
		}
		if _, duplicate := seen[fact.authority]; duplicate {
			return nil, fmt.Errorf("CLI_STUDY_PHASE_FACT_ALIAS_REFUSED")
		}
		seen[fact.authority] = struct{}{}
	}
	return facts, nil
}

func physicalRunAuthority(completed CompletedCLIStudy) (domain.Digest, error) {
	sourceResults := []struct {
		name   string
		result Result
	}{
		{name: "main-decisive", result: completed.discovery},
		{name: "main-eligibility-reference", result: completed.recoveryOne},
		{name: "main-eligibility-drift", result: completed.recoveryTwo},
		{name: "main-shape-reference", result: completed.shapeReference},
		{name: "main-shape-changed", result: completed.shapeChanged},
		{name: "main-baseline-checkpoint", result: completed.baselineCheckpoint},
		{name: "main-divergent-baseline", result: completed.baseline},
		{name: "main-accepted-reduction", result: completed.mainEvaluation},
		{name: "auxiliary-divergent-baseline", result: completed.auxiliaryBaseline},
		{name: "auxiliary-accepted-reduction", result: completed.auxiliaryEvaluation},
		{name: "main-confirmation", result: completed.confirmed},
		{name: "auxiliary-confirmation", result: completed.auxiliaryConfirmed},
	}
	sourceAuthorities := make([]string, len(sourceResults))
	for index, entry := range sourceResults {
		if err := validateResultSourceSpec(entry.result); err != nil {
			return "", errors.Join(err, fmt.Errorf("CLI_STUDY_SOURCE_AUTHORITY_REFUSED: %s", entry.name))
		}
		sourceAuthorities[index] = strings.Join([]string{
			entry.name,
			entry.result.SourceSpecDigest.String(),
			sha256Hex(entry.result.SourceSpecBytes),
			entry.result.Plan.Digest().String(),
		}, ":")
	}
	phase := make([]string, len(completed.phaseFacts))
	for index, fact := range completed.phaseFacts {
		phase[index] = strings.Join([]string{
			fact.phase, fmt.Sprintf("%03d", fact.trial), fact.kind,
			fact.authority.String(), fact.world.String(), fact.attempt.String(),
			fact.measurement.String(), fact.capture.String(), fact.projection.String(),
		}, ":")
	}
	targets := make([]string, len(completed.officialTrials))
	for index, trial := range completed.officialTrials {
		targets[index] = strings.Join([]string{
			trial.attemptDigest.String(), trial.targetDigest.String(), trial.targetCanonicalSHA256,
			trial.bundleDigest.String(), trial.residueHeadDigest.String(), trial.runDigest.String(),
			trial.runCanonicalSHA256, trial.classificationDigest.String(), trial.classificationCanonicalSHA256,
			trial.result, fmt.Sprintf("%d", trial.exitCode), fmt.Sprintf("%t", trial.recoveryEqual),
		}, ":")
	}
	if !completed.weakRun.Valid() || !completed.weakGrade.Valid() ||
		!completed.strongRun.Valid() || !completed.strongGrade.Valid() {
		return "", fmt.Errorf("CLI_STUDY_REDUCTION_AUTHORITY_REFUSED")
	}
	weakGrade := completed.weakGrade.Grade()
	strongGrade := completed.strongGrade.Grade()
	weakSweep, weakSweepPresent := weakGrade.CompletedSweepDigest()
	strongSweep, strongSweepPresent := strongGrade.CompletedSweepDigest()
	quantifiedSet, quantifiedSetPresent := strongGrade.QuantifiedReducerSetDigest()
	if weakSweepPresent || weakSweep.Valid() || !strongSweepPresent || !strongSweep.Valid() ||
		!quantifiedSetPresent || quantifiedSet != completed.strongRun.ReducerSet().Digest() {
		return "", fmt.Errorf("CLI_STUDY_REDUCTION_AUTHORITY_REFUSED")
	}
	reductions := []string{
		strings.Join([]string{
			"main-weak", completed.weakRun.Digest().String(), completed.weakRun.Transcript().Digest().String(),
			weakGrade.Digest().String(), string(weakGrade.Status()), completed.weakRun.ReducerSet().Digest().String(),
			completed.weakRun.MinimizedStimulusDigest().String(), string(completed.weakRun.Transcript().FinalSweepState()),
			weakSweep.String(), strings.Join(weakGrade.Limitations(), ","),
		}, ":"),
		strings.Join([]string{
			"auxiliary-strong", completed.strongRun.Digest().String(), completed.strongRun.Transcript().Digest().String(),
			strongGrade.Digest().String(), string(strongGrade.Status()), completed.strongRun.ReducerSet().Digest().String(),
			completed.strongRun.MinimizedStimulusDigest().String(), string(completed.strongRun.Transcript().FinalSweepState()),
			strongSweep.String(), strings.Join(strongGrade.Limitations(), ","),
		}, ":"),
	}
	exact, err := canon.CanonicalizeTyped(struct {
		SchemaVersion      string   `json:"schema_version"`
		Kind               string   `json:"kind"`
		Ordinal            int      `json:"ordinal"`
		PlanDigest         string   `json:"world_plan_digest"`
		ConfirmationDigest string   `json:"confirmation_digest"`
		AuxiliaryDigest    string   `json:"auxiliary_confirmation_digest"`
		DecisionDigest     string   `json:"decision_digest"`
		BundleDigest       string   `json:"bundle_digest"`
		SourceAuthorities  []string `json:"source_authorities"`
		ReductionAuthority []string `json:"reduction_authority"`
		PhaseAuthorities   []string `json:"phase_authorities"`
		OfficialGraphs     []string `json:"official_graphs"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "CLICompletedPhysicalRun", Ordinal: completed.ordinal,
		PlanDigest: completed.confirmed.Plan.Digest().String(), ConfirmationDigest: completed.confirmed.Confirmation.Draft().Digest().String(),
		AuxiliaryDigest: completed.auxiliaryConfirmed.Confirmation.Draft().Digest().String(), DecisionDigest: completed.decision.Digest().String(),
		BundleDigest: completed.bundle.Digest().String(), SourceAuthorities: sourceAuthorities, ReductionAuthority: reductions,
		PhaseAuthorities: phase, OfficialGraphs: targets,
	})
	if err != nil {
		return "", err
	}
	digest, err := canon.DigestBytes("CLICompletedPhysicalRun", exact)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func validateResultOwnership(result Result) error {
	if err := validateResultSourceSpec(result); err != nil {
		return err
	}
	if !result.Plan.Digest().Valid() || !result.Envelope.Digest().Valid() || !result.Stimulus.Valid() ||
		!result.CapturePolicy.Valid() || !result.ProjectionDefinition.Valid() || !result.Binding.Valid() ||
		result.Binding.PlanDigest() != result.Plan.Digest() || result.Binding.StimulusDigest() != result.Stimulus.Digest() ||
		result.Binding.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
		result.Binding.ProjectionDefinitionBinding().Digest() != result.ProjectionDefinition.Binding().Digest() ||
		!result.HasOutcomeMap || len(result.CandidateBindings) < 2 ||
		result.OutcomeMap.PlanDigest() != result.Plan.Digest() ||
		result.OutcomeMap.StimulusDigest() != result.Stimulus.Digest() ||
		result.OutcomeMap.EnvelopeDigest() != result.Envelope.Digest() ||
		result.OutcomeMap.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
		result.OutcomeMap.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
		len(result.CandidateBindings) != len(result.CandidateRoles) || len(result.Trials) == 0 {
		return fmt.Errorf("CLI_STUDY_RESULT_AUTHORITY_REFUSED")
	}
	bindings := make(map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding, len(result.CandidateBindings))
	for _, binding := range result.CandidateBindings {
		if !binding.Valid() || !binding.Key().Valid() {
			return fmt.Errorf("CLI_STUDY_RESULT_BINDING_REFUSED")
		}
		if _, duplicate := bindings[binding.Key()]; duplicate {
			return fmt.Errorf("CLI_STUDY_RESULT_BINDING_ALIAS_REFUSED")
		}
		if _, rolePresent := result.CandidateRoles[binding.Key()]; !rolePresent {
			return fmt.Errorf("CLI_STUDY_RESULT_ROLE_REFUSED")
		}
		bindings[binding.Key()] = binding
	}
	entries := make(map[domain.CandidateExecutionKey]domain.ProjectionFingerprint)
	for _, entry := range result.OutcomeMap.Entries() {
		entries[entry.CandidateKey] = entry.ProjectionFingerprint
	}
	exclusions := make(map[domain.CandidateExecutionKey]struct{})
	for _, exclusion := range result.OutcomeMap.Exclusions() {
		exclusions[exclusion.CandidateKey] = struct{}{}
	}
	worlds := make([]domain.Digest, 0, len(result.Trials))
	attempts := make([]domain.Digest, 0, len(result.Trials))
	captures := make([]domain.Digest, 0, len(result.Trials))
	for _, trial := range result.Trials {
		key := trial.Slot.CandidateKey()
		role, knownRole := result.CandidateRoles[key]
		if _, bound := bindings[key]; !bound || !knownRole || trial.Role != role || !trial.Admitted ||
			!trial.Result.World().Digest().Valid() || !trial.Result.FinalizedAttempt().ArtifactDigest().Valid() ||
			!trial.Measurements.Digest().Valid() || !trial.Observation.Valid() {
			return fmt.Errorf("CLI_STUDY_RESULT_TRIAL_OWNER_REFUSED")
		}
		world := trial.Result.World()
		attempt := trial.Result.FinalizedAttempt()
		if world.CandidateKey() != key || world.PlanDigest() != result.Plan.Digest() ||
			world.StimulusDigest() != result.Stimulus.Digest() || world.AttemptArtifactDigest() != attempt.ArtifactDigest() ||
			world.EnvelopeDigest() != result.Envelope.Digest() ||
			world.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
			world.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
			world.Purpose() != result.OutcomeMap.Phase() || world.ScheduleOrdinal() != trial.Slot.Ordinal() ||
			trial.Measurements.SubjectDigest() != world.Digest() || trial.Observation.WorldDigest() != world.Digest() ||
			trial.Observation.AttemptDigest() != attempt.ArtifactDigest() || trial.Observation.CandidateKey() != key ||
			trial.Observation.PlanDigest() != result.Plan.Digest() || trial.Observation.StimulusDigest() != result.Stimulus.Digest() {
			return fmt.Errorf("CLI_STUDY_RESULT_TRIAL_LINEAGE_REFUSED")
		}
		if trial.Observation.ExecutionPayloadDigest() != result.Binding.ExecutionPayloadDigest() ||
			trial.Observation.TrialIndex() != trial.Slot.Ordinal() ||
			trial.Observation.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
			trial.Observation.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() {
			return fmt.Errorf("CLI_STUDY_RESULT_CAPTURE_LINEAGE_REFUSED")
		}
		if trial.Projected {
			fingerprint, fingerprintErr := domain.NewProjectionFingerprint(trial.Projection.ProjectionBytes())
			if fingerprintErr != nil || entries[key] != fingerprint || trial.ProjectionRejection != nil ||
				trial.Result.FinalizedAttempt().HasControls() {
				return fmt.Errorf("CLI_STUDY_RESULT_PROJECTION_JOIN_REFUSED")
			}
			if err := validateProjectionLineage(
				result.ProjectionDefinition, trial.Observation, true, trial.Projection, nil,
			); err != nil {
				return err
			}
		} else if trial.Result.FinalizedAttempt().HasControls() {
			primary, primaryPresent := trial.Result.FinalizedAttempt().PrimaryControl()
			if _, excluded := exclusions[key]; !excluded || trial.ProjectionRejection != nil ||
				!primaryPresent || primary != domain.ControlOutputLimit {
				return fmt.Errorf("CLI_STUDY_RESULT_CONTROL_JOIN_REFUSED")
			}
		} else {
			if _, excluded := exclusions[key]; !excluded || trial.ProjectionRejection == nil ||
				!trial.ProjectionRejection.EvidenceDigest().Valid() {
				return fmt.Errorf("CLI_STUDY_RESULT_EXCLUSION_JOIN_REFUSED")
			}
			if err := validateProjectionLineage(
				result.ProjectionDefinition, trial.Observation, false, countercli.CLIProjectionResult{}, trial.ProjectionRejection,
			); err != nil {
				return err
			}
		}
		worlds = append(worlds, world.Digest())
		attempts = append(attempts, attempt.ArtifactDigest())
		if trial.Projected || trial.ProjectionRejection != nil {
			captures = append(captures, trial.Observation.Digest())
		}
	}
	if !sameDigestSet(worlds, result.OutcomeMap.EvidenceWorldDigests()) ||
		!sameDigestSet(attempts, result.OutcomeMap.EvidenceAttemptDigests()) ||
		!sameDigestSet(captures, result.OutcomeMap.EvidenceObservationDigests()) {
		return fmt.Errorf("CLI_STUDY_RESULT_MAP_EVIDENCE_REFUSED")
	}
	return validateResultScheduleAndMap(result)
}

func validateResultSourceSpec(result Result) error {
	if !result.SourceSpecDigest.Valid() || len(result.SourceSpecBytes) == 0 ||
		!result.ProjectionDefinition.Binding().Valid() {
		return fmt.Errorf("CLI_STUDY_RESULT_SOURCE_REFUSED")
	}
	parsed, err := corespec.ParseSource(result.SourceSpecBytes)
	if err != nil || parsed.Digest() != result.SourceSpecDigest ||
		!bytes.Equal(parsed.CanonicalBytes(), result.SourceSpecBytes) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_RESULT_SOURCE_PARSE_REFUSED"))
	}
	recompiled, err := corespec.Compile(parsed, result.ProjectionDefinition.Binding())
	if err != nil || recompiled.Digest() != result.Plan.Digest() ||
		!bytes.Equal(recompiled.CanonicalBytes(), result.Plan.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_RESULT_SOURCE_PLAN_REFUSED"))
	}
	return nil
}

func validateResultScheduleAndMap(result Result) error {
	roster := result.CanonicalCandidateRoster()
	if len(roster) < 2 || len(result.Trials)%len(roster) != 0 {
		return fmt.Errorf("CLI_STUDY_RESULT_SCHEDULE_SHAPE_REFUSED")
	}
	repetitions := len(result.Trials) / len(roster)
	phase := result.OutcomeMap.Phase()
	if result.HasConfirmation != (phase == domain.AttemptConfirmation) ||
		!sameCandidateKeysInOrder(result.OutcomeMap.CandidateRoster(), roster) {
		return fmt.Errorf("CLI_STUDY_RESULT_PHASE_ROSTER_REFUSED")
	}
	schedule, err := observe.NewPhaseRotatedSchedule(roster, repetitions, phase)
	if err != nil || !schedule.Valid() || schedule.Digest() != result.OutcomeMap.ScheduleDigest() ||
		schedule.StartOffset() != result.OutcomeMap.ScheduleStartOffset() ||
		string(schedule.Rotation()) != result.OutcomeMap.Rotation() ||
		!sameCandidateKeysInOrder(schedule.Roster(), roster) || schedule.TotalTrials() != len(result.Trials) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_RESULT_SCHEDULE_REFUSED"))
	}
	slots := schedule.Trials()
	for index, trial := range result.Trials {
		if trial.Slot != slots[index] || trial.Result.World().ScheduleOrdinal() != slots[index].Ordinal() ||
			trial.Result.World().CandidateKey() != slots[index].CandidateKey() {
			return fmt.Errorf("CLI_STUDY_RESULT_TRIAL_ORDER_REFUSED")
		}
	}
	if !result.HasConfirmation {
		observedSchedule := result.Observation.Schedule()
		input, inputPresent := result.Observation.OutcomeMapInput()
		if !observedSchedule.Valid() || observedSchedule.Digest() != schedule.Digest() ||
			!bytes.Equal(observedSchedule.CanonicalBytes(), schedule.CanonicalBytes()) ||
			result.Observation.Status() != observe.ObservationComplete ||
			result.Observation.CompletedMatrices() != repetitions || !inputPresent ||
			input.StimulusDigest() != result.Stimulus.Digest() || input.EnvelopeDigest() != result.Envelope.Digest() ||
			!sameCandidateKeysInOrder(input.Roster(), roster) {
			return fmt.Errorf("CLI_STUDY_OBSERVATION_ORDER_REFUSED")
		}
		observedBatches := result.Observation.Batches()
		inputBatches := input.Batches()
		if len(observedBatches) != len(inputBatches) || len(inputBatches) != len(roster) {
			return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_COUNT_REFUSED")
		}
		for index := range observedBatches {
			if observedBatches[index].Digest() != inputBatches[index].Digest() ||
				!bytes.Equal(observedBatches[index].CanonicalBytes(), inputBatches[index].CanonicalBytes()) {
				return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_ALIAS_REFUSED")
			}
		}
		if err := validateObservationBatchOrder(result, schedule, inputBatches); err != nil {
			return err
		}
		rebuilt, rebuildErr := compare.NewCandidateOutcomeMap(
			input.StimulusDigest(), input.EnvelopeDigest(), input.Roster(), inputBatches,
		)
		if rebuildErr != nil || rebuilt.ArtifactDigest() != result.OutcomeMap.ArtifactDigest() ||
			!bytes.Equal(rebuilt.CanonicalBytes(), result.OutcomeMap.CanonicalBytes()) {
			return errors.Join(rebuildErr, fmt.Errorf("CLI_STUDY_OBSERVATION_MAP_REPLAY_REFUSED"))
		}
		return nil
	}
	if result.Observation.Schedule().Valid() || len(result.Observation.Batches()) != 0 {
		return fmt.Errorf("CLI_STUDY_CONFIRMATION_OBSERVATION_ALIAS_REFUSED")
	}
	if _, present := result.Observation.OutcomeMapInput(); present {
		return fmt.Errorf("CLI_STUDY_CONFIRMATION_OBSERVATION_ALIAS_REFUSED")
	}
	draft := result.Confirmation.Draft()
	physical := draft.PhysicalFacts()
	record := draft.Record()
	factDigests := record.PhysicalFactDigests()
	executionBindings := record.ExecutionBindingDigests()
	if !draft.Valid() || !record.Valid() || len(physical) != len(result.Trials) ||
		len(factDigests) != len(physical) || len(executionBindings) != len(physical) {
		return fmt.Errorf("CLI_STUDY_CONFIRMATION_ORDER_REFUSED")
	}
	for index, fact := range physical {
		trial := result.Trials[index]
		slot := slots[index]
		if !fact.Valid() || factDigests[index] != fact.Digest() ||
			executionBindings[index] != result.Binding.Digest() ||
			fact.PlanDigest() != result.Plan.Digest() || fact.StimulusDigest() != result.Stimulus.Digest() ||
			fact.Purpose() != domain.AttemptConfirmation || fact.ScheduleOrdinal() != slot.Ordinal() ||
			fact.CandidateKey() != slot.CandidateKey() || fact.WorldDigest() != trial.Result.World().Digest() ||
			fact.AttemptArtifactDigest() != trial.Result.FinalizedAttempt().ArtifactDigest() {
			return fmt.Errorf("CLI_STUDY_CONFIRMATION_PHYSICAL_ORDER_REFUSED")
		}
	}
	return nil
}

func validateObservationBatchOrder(result Result, schedule observe.RotatedSchedule, batches []observe.StableBatch) error {
	roster := schedule.Roster()
	for batchIndex, candidate := range roster {
		batch := batches[batchIndex]
		worlds := batch.WorldDigests()
		attempts := batch.AttemptDigests()
		observations := batch.ObservationDigests()
		ordinals := batch.ScheduleOrdinals()
		if batch.CandidateKey() != candidate || batch.ScheduleDigest() != schedule.Digest() ||
			batch.ScheduleStartOffset() != schedule.StartOffset() || batch.Rotation() != string(schedule.Rotation()) ||
			batch.RequiredFreshTrials() != schedule.Repetitions() || len(worlds) != schedule.Repetitions() ||
			len(attempts) != len(worlds) || len(ordinals) != len(worlds) {
			return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_ORDER_REFUSED")
		}
		expectedObservations := make([]domain.Digest, 0, len(worlds))
		for repetition := 0; repetition < schedule.Repetitions(); repetition++ {
			slot, present := schedule.Slot(candidate, repetition)
			if !present || slot.Ordinal() < 0 || slot.Ordinal() >= len(result.Trials) {
				return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_SLOT_REFUSED")
			}
			trial := result.Trials[slot.Ordinal()]
			if trial.Slot != slot || ordinals[repetition] != slot.Ordinal() ||
				worlds[repetition] != trial.Result.World().Digest() ||
				attempts[repetition] != trial.Result.FinalizedAttempt().ArtifactDigest() {
				return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_TRIAL_REFUSED")
			}
			if trial.Projected || trial.ProjectionRejection != nil {
				expectedObservations = append(expectedObservations, trial.Observation.Digest())
			}
		}
		if !sameDigestsInOrder(observations, expectedObservations) {
			return fmt.Errorf("CLI_STUDY_OBSERVATION_BATCH_CAPTURE_REFUSED")
		}
	}
	return nil
}

func sameCandidateKeysInOrder(left, right []domain.CandidateExecutionKey) bool {
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

func sameDigestsInOrder(left, right []domain.Digest) bool {
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

func sameDigestSet(left, right []domain.Digest) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[domain.Digest]struct{}, len(left))
	for _, digest := range left {
		if !digest.Valid() {
			return false
		}
		if _, duplicate := seen[digest]; duplicate {
			return false
		}
		seen[digest] = struct{}{}
	}
	rightSeen := make(map[domain.Digest]struct{}, len(right))
	for _, digest := range right {
		if !digest.Valid() {
			return false
		}
		if _, duplicate := rightSeen[digest]; duplicate {
			return false
		}
		rightSeen[digest] = struct{}{}
		if _, present := seen[digest]; !present {
			return false
		}
	}
	return true
}

func validateWeakReductionFacts(baseline, minimized Result, run reducer.ReductionRun, weak grade.Result) error {
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baseline.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIFixtureRemove, countercli.CLIEnvironmentRemove},
	})
	weakValue := weak.Grade()
	weakSweep, weakSweepPresent := weakValue.CompletedSweepDigest()
	if err != nil || !run.Valid() || !weak.Valid() || run.Baseline().OutcomeMap().ArtifactDigest() != baseline.OutcomeMap.ArtifactDigest() ||
		run.OriginalStimulusDigest() != baseline.Stimulus.Digest() || run.ReducerSet().Digest() != policy.ReducerSet().Digest() ||
		run.MinimizedStimulusDigest() != minimized.Stimulus.Digest() || weak.RunDigest() != run.Digest() ||
		weak.TranscriptDigest() != run.Transcript().Digest() || weakValue.Status() != grade.StatusBestKnown ||
		weakValue.RunDigest() != run.Digest() || weakValue.ReducerSetDigest() != run.ReducerSet().Digest() ||
		weakSweepPresent || weakSweep.Valid() ||
		run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Budget().ProposalLimit() != 2 || run.Budget().CandidateTrialLimit() != 10 ||
		compare.AssessPreservation(baseline.OutcomeMap, minimized.OutcomeMap).Relation() != compare.PreservationEqual {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_WEAK_REDUCTION_JOIN_REFUSED"))
	}
	initial, err := countercli.EnumerateCLINeighbors(baseline.Stimulus, policy)
	if err != nil || len(initial) != 2 {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_WEAK_INITIAL_NEIGHBORS_REFUSED"))
	}
	firstRecipe := initial[0].ReplayRecipe()
	if firstRecipe.Rule != countercli.CLIFixtureRemove || firstRecipe.Transform != countercli.CLITransformRemove ||
		firstRecipe.Index < 0 || firstRecipe.Index >= len(baseline.Stimulus.Fixtures()) ||
		baseline.Stimulus.Fixtures()[firstRecipe.Index].Path() != "z-main-irrelevant.txt" ||
		initial[0].LogicalNeighbor().Locus() != fmt.Sprintf("fixture[%d]", firstRecipe.Index) ||
		initial[0].Stimulus().Digest() != minimized.Stimulus.Digest() {
		return fmt.Errorf("CLI_STUDY_WEAK_FIXTURE_NEIGHBOR_REFUSED")
	}
	replayedFirst, err := countercli.ReplayCLINeighbor(baseline.Stimulus, policy, firstRecipe)
	if err != nil || replayedFirst.Digest() != minimized.Stimulus.Digest() ||
		!bytes.Equal(replayedFirst.CanonicalBytes(), minimized.Stimulus.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_WEAK_FIXTURE_REPLAY_REFUSED"))
	}
	remaining, err := countercli.EnumerateCLINeighbors(minimized.Stimulus, policy)
	if err != nil || len(remaining) != 1 {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_WEAK_REMAINING_NEIGHBOR_REFUSED"))
	}
	secondRecipe := remaining[0].ReplayRecipe()
	if secondRecipe.Rule != countercli.CLIEnvironmentRemove || secondRecipe.Transform != countercli.CLITransformRemove ||
		secondRecipe.Index < 0 || secondRecipe.Index >= len(minimized.Stimulus.Environment()) ||
		minimized.Stimulus.Environment()[secondRecipe.Index].Name() != "APP_MODE" ||
		remaining[0].LogicalNeighbor().Locus() != fmt.Sprintf("environment[%d]", secondRecipe.Index) {
		return fmt.Errorf("CLI_STUDY_WEAK_ENVIRONMENT_NEIGHBOR_REFUSED")
	}
	replayedSecond, err := countercli.ReplayCLINeighbor(minimized.Stimulus, policy, secondRecipe)
	entries := run.Transcript().Entries()
	accepted := run.Transcript().AcceptedPath()
	if err != nil || len(entries) != 2 || len(accepted) != 1 || accepted[0] != initial[0].LogicalNeighbor().Digest() ||
		entries[0].Evaluation().Neighbor().Digest() != initial[0].LogicalNeighbor().Digest() ||
		entries[1].Evaluation().Neighbor().Digest() != remaining[0].LogicalNeighbor().Digest() ||
		entries[1].Evaluation().Neighbor().StimulusDigest() != replayedSecond.Digest() ||
		entries[0].Evaluation().Purpose() != domain.AttemptReduction ||
		entries[1].Evaluation().Purpose() != domain.AttemptReduction ||
		entries[0].Evaluation().Decision() != reducer.Preserves || entries[0].CandidateTrials() != 9 ||
		entries[0].ProposalCount() != 1 || entries[0].TrialCount() != 9 ||
		entries[1].Evaluation().Decision() != reducer.Unresolved ||
		entries[1].Evaluation().ReasonCode() != string(reducer.ReasonEvaluatorError) || entries[1].CandidateTrials() != 0 ||
		entries[1].ProposalCount() != 2 || entries[1].TrialCount() != 9 ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepIncomplete ||
		!sameStringsInOrder(run.Transcript().Limitations(), []string{
			"UNRESOLVED_EVALUATOR_ERROR", "PROPOSAL_BUDGET_EXHAUSTED",
		}) ||
		!sameStringsInOrder(weakValue.Limitations(), []string{
			"UNRESOLVED_EVALUATOR_ERROR", "PROPOSAL_BUDGET_EXHAUSTED", "DURABLE_SWEEP_AUTHORITY_ABSENT",
		}) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_WEAK_TRANSCRIPT_REFUSED"))
	}
	if _, present, draftErr := run.CompletedSweepDraft(); draftErr != nil || present {
		return errors.Join(draftErr, fmt.Errorf("CLI_STUDY_WEAK_DRAFT_REFUSED"))
	}
	return nil
}

func validateStrongReductionFacts(baseline, minimized Result, run reducer.ReductionRun, strong grade.Result) error {
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baseline.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIFixtureRemove},
	})
	strongValue := strong.Grade()
	strongSweep, strongSweepPresent := strongValue.CompletedSweepDigest()
	quantifiedSet, quantifiedSetPresent := strongValue.QuantifiedReducerSetDigest()
	if err != nil || !run.Valid() || !strong.Valid() || strong.RunDigest() != run.Digest() ||
		run.Baseline().OutcomeMap().ArtifactDigest() != baseline.OutcomeMap.ArtifactDigest() ||
		run.OriginalStimulusDigest() != baseline.Stimulus.Digest() || run.ReducerSet().Digest() != policy.ReducerSet().Digest() ||
		strong.TranscriptDigest() != run.Transcript().Digest() || strongValue.Status() != grade.StatusOneMinimalUnder ||
		strongValue.RunDigest() != run.Digest() || strongValue.ReducerSetDigest() != run.ReducerSet().Digest() ||
		!strongSweepPresent || !strongSweep.Valid() || !quantifiedSetPresent || quantifiedSet != run.ReducerSet().Digest() ||
		len(strongValue.Limitations()) != 0 || len(run.Transcript().Limitations()) != 0 ||
		run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.MinimizedStimulusDigest() != minimized.Stimulus.Digest() ||
		run.Budget().ProposalLimit() != 2 || run.Budget().CandidateTrialLimit() != 2 ||
		compare.AssessPreservation(baseline.OutcomeMap, minimized.OutcomeMap).Relation() != compare.PreservationEqual {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_STRONG_REDUCTION_JOIN_REFUSED"))
	}
	neighbors, err := countercli.EnumerateCLINeighbors(baseline.Stimulus, policy)
	entries := run.Transcript().Entries()
	if err != nil || len(neighbors) != 1 || len(entries) != 1 ||
		neighbors[0].ReplayRecipe().Rule != countercli.CLIFixtureRemove ||
		neighbors[0].ReplayRecipe().Transform != countercli.CLITransformRemove ||
		neighbors[0].ReplayRecipe().Index < 0 ||
		neighbors[0].ReplayRecipe().Index >= len(baseline.Stimulus.Fixtures()) ||
		baseline.Stimulus.Fixtures()[neighbors[0].ReplayRecipe().Index].Path() != "z-auxiliary-irrelevant.txt" ||
		neighbors[0].LogicalNeighbor().Locus() != fmt.Sprintf("fixture[%d]", neighbors[0].ReplayRecipe().Index) ||
		neighbors[0].Stimulus().Digest() != minimized.Stimulus.Digest() ||
		entries[0].Evaluation().Neighbor().Digest() != neighbors[0].LogicalNeighbor().Digest() ||
		entries[0].Evaluation().Purpose() != domain.AttemptReduction ||
		entries[0].Evaluation().Decision() != reducer.Preserves || entries[0].CandidateTrials() != 2 ||
		entries[0].ProposalCount() != 1 || entries[0].TrialCount() != 2 ||
		len(run.Transcript().AcceptedPath()) != 1 ||
		run.Transcript().AcceptedPath()[0] != neighbors[0].LogicalNeighbor().Digest() ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepComplete {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_STRONG_TRANSCRIPT_REFUSED"))
	}
	replayed, replayErr := countercli.ReplayCLINeighbor(baseline.Stimulus, policy, neighbors[0].ReplayRecipe())
	if replayErr != nil || replayed.Digest() != minimized.Stimulus.Digest() ||
		!bytes.Equal(replayed.CanonicalBytes(), minimized.Stimulus.CanonicalBytes()) {
		return errors.Join(replayErr, fmt.Errorf("CLI_STUDY_STRONG_FIXTURE_REPLAY_REFUSED"))
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 ||
		draft.RunDigest() != run.Digest() || draft.TranscriptDigest() != run.Transcript().Digest() ||
		draft.CurrentStimulusDigest() != minimized.Stimulus.Digest() ||
		draft.ReducerSetDigest() != run.ReducerSet().Digest() || strongSweep != draft.Digest() {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_STRONG_DRAFT_REFUSED"))
	}
	return nil
}

func sameStringsInOrder(left, right []string) bool {
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

func validateGlobalPhysicalNonalias(completed CompletedCLIStudy) error {
	results := []Result{
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint, completed.baseline,
		completed.mainEvaluation, completed.auxiliaryBaseline, completed.auxiliaryEvaluation,
		completed.confirmed, completed.auxiliaryConfirmed,
	}
	sets := []map[domain.Digest]struct{}{{}, {}, {}, {}, {}}
	add := func(index int, digest domain.Digest) error {
		if !digest.Valid() {
			return fmt.Errorf("CLI_STUDY_GLOBAL_PHYSICAL_DIGEST_REFUSED")
		}
		if _, duplicate := sets[index][digest]; duplicate {
			return fmt.Errorf("CLI_STUDY_GLOBAL_PHYSICAL_ALIAS_REFUSED")
		}
		sets[index][digest] = struct{}{}
		return nil
	}
	for _, result := range results {
		for _, trial := range result.Trials {
			projection := trial.Projection.DerivationDigest()
			if trial.ProjectionRejection != nil {
				projection = trial.ProjectionRejection.EvidenceDigest()
			}
			for index, digest := range []domain.Digest{
				trial.Result.World().Digest(), trial.Result.FinalizedAttempt().ArtifactDigest(),
				trial.Measurements.Digest(), trial.Observation.Digest(), projection,
			} {
				if index == 4 && !digest.Valid() && trial.Result.FinalizedAttempt().HasControls() &&
					trial.ProjectionRejection == nil {
					primary, primaryPresent := trial.Result.FinalizedAttempt().PrimaryControl()
					if primaryPresent && primary == domain.ControlOutputLimit {
						continue
					}
				}
				if err := add(index, digest); err != nil {
					return err
				}
			}
		}
	}
	for _, control := range completed.semanticControls {
		projection := control.projection.DerivationDigest()
		if control.projectionRejection != nil {
			projection = control.projectionRejection.EvidenceDigest()
		}
		values := []domain.Digest{
			control.result.World().Digest(), control.result.FinalizedAttempt().ArtifactDigest(),
			control.measurements.Digest(), control.observation.Digest(), projection,
		}
		for index, digest := range values {
			if index == 4 && !digest.Valid() && (control.kind == "CONTROL_TIMEOUT" || control.kind == "CONTROL_OUTPUT_LIMIT") {
				continue
			}
			if err := add(index, digest); err != nil {
				return err
			}
		}
	}
	if len(sets[0]) != 88 || len(sets[1]) != 88 || len(sets[2]) != 88 || len(sets[3]) != 88 || len(sets[4]) != 80 {
		return fmt.Errorf("CLI_STUDY_GLOBAL_PHYSICAL_COUNT_REFUSED")
	}
	return nil
}

var expectedControlKinds = []string{
	"CONTROL_STDIN_ABSENT",
	"CONTROL_STDIN_PRESENT_EMPTY",
	"CONTROL_APP_MODE_ABSENT",
	"CONTROL_APP_MODE_PRESENT_EMPTY",
	"CONTROL_ORDERED_ARGV_MODE_THEN_STDERR",
	"CONTROL_ORDERED_ARGV_STDERR_THEN_MODE",
	"CONTROL_SPARSE_IRRELEVANT_ENV_ALPHA",
	"CONTROL_SPARSE_IRRELEVANT_ENV_OMEGA",
	"CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION",
	"CONTROL_UNSELECTED_STDERR_ALPHA",
	"CONTROL_UNSELECTED_STDERR_BETA",
	"CONTROL_EXIT_SEVEN_SUBSTRATE",
	"CONTROL_SIGNAL_SELECTED_EXIT_MISSING",
	"CONTROL_TIMEOUT",
	"CONTROL_MALFORMED_SELECTED_JSON",
	"CONTROL_OUTPUT_LIMIT",
}

func validateControlRoster(base countercli.CLIStimulus, controls []physicalControl) error {
	if len(controls) != len(expectedControlKinds) {
		return fmt.Errorf("CLI_STUDY_COMPLETED_CONTROL_COUNT_REFUSED")
	}
	inputs, err := expectedSemanticControlInputs(base)
	if err != nil || len(inputs) != len(controls) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_CONTROL_INPUT_REFUSED"))
	}
	for index, control := range controls {
		input := inputs[index]
		expectedStimulus, stimulusErr := stimulusWithInputs(base, input.argv, input.stdin, input.env)
		expectedDefinition, definitionErr := countercli.NewCLIProjectionDefinition(
			countercli.CLIProjectionDefinitionConfig{Fields: append([]countercli.CLIFieldID(nil), input.fields...)},
		)
		if stimulusErr != nil || definitionErr != nil || control.kind != expectedControlKinds[index] ||
			control.kind != input.kind || control.stimulus.Digest() != expectedStimulus.Digest() ||
			!bytes.Equal(control.stimulus.CanonicalBytes(), expectedStimulus.CanonicalBytes()) ||
			control.definition.Digest() != expectedDefinition.Digest() ||
			control.definition.Binding().Digest() != expectedDefinition.Binding().Digest() ||
			!bytes.Equal(control.definition.CanonicalBytes(), expectedDefinition.CanonicalBytes()) ||
			!bytes.Equal(control.definition.Binding().CanonicalBytes(), expectedDefinition.Binding().CanonicalBytes()) {
			return errors.Join(stimulusErr, definitionErr, fmt.Errorf("CLI_STUDY_COMPLETED_CONTROL_IDENTITY_REFUSED"))
		}
		if control.kind != expectedControlKinds[index] {
			return fmt.Errorf("CLI_STUDY_COMPLETED_CONTROL_ORDER_REFUSED")
		}
		if err := validatePhysicalControl(control); err != nil {
			return err
		}
	}
	for _, pair := range [][2]int{{0, 1}, {2, 3}, {4, 5}, {6, 7}, {9, 10}} {
		if err := validateSelectedControlPair(controls[pair[0]], controls[pair[1]], pair[0] == 9); err != nil {
			return err
		}
	}
	if controls[0].result.Process().StdinPresence() == controls[1].result.Process().StdinPresence() {
		return fmt.Errorf("CLI_STUDY_STDIN_CONTROL_PHYSICAL_REFUSED")
	}
	defaultExit := controls[8]
	signalExit := controls[12]
	if !defaultExit.projected || !signalExit.projected ||
		!projectedExitMatches(defaultExit.projection, countercli.ExactInteger, 0) ||
		!projectedExitMatches(signalExit.projection, countercli.ExactMissing, 0) ||
		bytes.Equal(defaultExit.projection.ProjectionBytes(), signalExit.projection.ProjectionBytes()) ||
		defaultExit.projection.DerivationDigest() == signalExit.projection.DerivationDigest() {
		return fmt.Errorf("CLI_STUDY_SIGNAL_CONTRADICTION_REFUSED")
	}
	return nil
}

func validateSelectedControlPair(left, right physicalControl, requireStderrDifference bool) error {
	if !left.projected || !right.projected || left.projectionRejection != nil || right.projectionRejection != nil ||
		left.stimulus.Digest() == right.stimulus.Digest() ||
		bytes.Equal(left.stimulus.CanonicalBytes(), right.stimulus.CanonicalBytes()) ||
		left.definition.Digest() != right.definition.Digest() ||
		!bytes.Equal(left.definition.CanonicalBytes(), right.definition.CanonicalBytes()) ||
		!bytes.Equal(left.projection.ProjectionBytes(), right.projection.ProjectionBytes()) ||
		left.projection.DerivationDigest() == right.projection.DerivationDigest() ||
		left.observation.Digest() == right.observation.Digest() {
		return fmt.Errorf("CLI_STUDY_CONTROL_PAIR_REFUSED: %s/%s", left.kind, right.kind)
	}
	if !requireStderrDifference {
		return nil
	}
	leftStderr := left.observation.Stderr()
	rightStderr := right.observation.Stderr()
	if leftStderr.State() != countercli.ChannelPresent || rightStderr.State() != countercli.ChannelPresent ||
		leftStderr.RetainedDigest() == rightStderr.RetainedDigest() ||
		bytes.Equal(leftStderr.Bytes(), rightStderr.Bytes()) {
		return fmt.Errorf("CLI_STUDY_CONTROL_STDERR_PAIR_REFUSED")
	}
	return nil
}

func validateReductionCohortTopology(
	weakRun, strongRun domain.Digest,
	weakTranscript, strongTranscript domain.Digest,
	weakGrade, strongGrade domain.Digest,
	weakMinimized, strongMinimized, expectedMinimized domain.Digest,
	weakCanonical, strongCanonical, expectedCanonical []byte,
) error {
	for _, authority := range []domain.Digest{
		weakRun, strongRun, weakTranscript, strongTranscript, weakGrade, strongGrade,
		weakMinimized, strongMinimized, expectedMinimized,
	} {
		if !authority.Valid() {
			return fmt.Errorf("CLI_STUDY_COMPLETED_REDUCTION_CONVERGENCE_REFUSED")
		}
	}
	if weakRun == strongRun || weakTranscript == strongTranscript || weakGrade == strongGrade {
		return fmt.Errorf("CLI_STUDY_COMPLETED_REDUCTION_ALIAS_REFUSED")
	}
	if weakMinimized != strongMinimized || weakMinimized != expectedMinimized ||
		len(weakCanonical) == 0 || len(strongCanonical) == 0 || len(expectedCanonical) == 0 ||
		!bytes.Equal(weakCanonical, strongCanonical) || !bytes.Equal(weakCanonical, expectedCanonical) {
		return fmt.Errorf("CLI_STUDY_COMPLETED_REDUCTION_CONVERGENCE_REFUSED")
	}
	return nil
}

func validateConfirmationReduction(
	minimized, confirmed Result,
	run reducer.ReductionRun,
	reductionResult grade.Result,
	candidates, repetitions int,
) error {
	if err := validateConfirmation(minimized, confirmed, candidates, repetitions); err != nil {
		return err
	}
	draft := confirmed.Confirmation.Draft()
	record := draft.Record()
	gradeValue := reductionResult.Grade()
	if !run.Valid() || !reductionResult.Valid() || !draft.Valid() || !record.Valid() ||
		confirmed.Plan.Digest() != minimized.Plan.Digest() || confirmed.Stimulus.Digest() != minimized.Stimulus.Digest() ||
		run.MinimizedStimulusDigest() != minimized.Stimulus.Digest() || reductionResult.RunDigest() != run.Digest() ||
		reductionResult.TranscriptDigest() != run.Transcript().Digest() || gradeValue.RunDigest() != run.Digest() ||
		record.PlanDigest() != confirmed.Plan.Digest() || record.ReductionRunDigest() != run.Digest() ||
		record.ReductionGradeDigest() != gradeValue.Digest() ||
		!bytes.Equal(record.ReductionGradeCanonicalBytes(), gradeValue.CanonicalBytes()) ||
		record.ReductionGradeStatus() != string(gradeValue.Status()) ||
		record.OriginalBaselineMap().ArtifactDigest() != run.Baseline().OutcomeMap().ArtifactDigest() ||
		record.ReducedBaselineMap().ArtifactDigest() != minimized.OutcomeMap.ArtifactDigest() ||
		record.ConfirmedMap().ArtifactDigest() != confirmed.OutcomeMap.ArtifactDigest() ||
		draft.ReducedBaseline().OutcomeMap().ArtifactDigest() != minimized.OutcomeMap.ArtifactDigest() ||
		draft.ConfirmedOutcomeMap().OutcomeMap().ArtifactDigest() != confirmed.OutcomeMap.ArtifactDigest() {
		return fmt.Errorf("CLI_STUDY_CONFIRMATION_REDUCTION_JOIN_REFUSED")
	}
	return nil
}

func validateChoicePublicationJoins(completed CompletedCLIStudy, constructionProof bool) error {
	if err := validateChoicepointCompletionJoin(completed); err != nil {
		return err
	}
	confirmed := completed.confirmed
	source := completed.portableSource
	baseArgv := confirmed.Stimulus.BaseArgv()
	view, viewPresent := source.CLIView()
	expectedExecution, executionErr := countercli.BindExecution(
		confirmed.Plan, confirmed.Stimulus, confirmed.CapturePolicy, confirmed.ProjectionDefinition,
	)
	if executionErr != nil || len(baseArgv) == 0 || !source.Valid() || source.Adapter() != domain.AdapterCLI ||
		source.StimulusKind() != "CLIStimulus" || source.Entrypoint() != baseArgv[0] ||
		source.StartProfile() != contractsource.CLIStartProfileV1 ||
		source.Plan().Digest() != confirmed.Plan.Digest() ||
		!bytes.Equal(source.Plan().CanonicalBytes(), confirmed.Plan.CanonicalBytes()) ||
		source.StimulusDigest() != confirmed.Stimulus.Digest() ||
		!bytes.Equal(source.StimulusCanonicalBytes(), confirmed.Stimulus.CanonicalBytes()) ||
		source.ExecutionBindingDigest() != confirmed.Binding.Digest() ||
		!bytes.Equal(source.ExecutionBindingCanonicalBytes(), confirmed.Binding.CanonicalBytes()) ||
		expectedExecution.Digest() != confirmed.Binding.Digest() ||
		!bytes.Equal(expectedExecution.CanonicalBytes(), confirmed.Binding.CanonicalBytes()) ||
		source.ProjectionBinding().Digest() != confirmed.ProjectionDefinition.Binding().Digest() ||
		!bytes.Equal(source.ProjectionBinding().CanonicalBytes(), confirmed.ProjectionDefinition.Binding().CanonicalBytes()) ||
		!viewPresent || !view.Valid() || view.SourceDigest() != source.Digest() ||
		view.Stimulus().Digest() != confirmed.Stimulus.Digest() ||
		!bytes.Equal(view.Stimulus().CanonicalBytes(), confirmed.Stimulus.CanonicalBytes()) ||
		view.Capture().Digest() != confirmed.CapturePolicy.Digest() ||
		!bytes.Equal(view.Capture().CanonicalBytes(), confirmed.CapturePolicy.CanonicalBytes()) ||
		view.Projection().Digest() != confirmed.ProjectionDefinition.Digest() ||
		view.Projection().Binding().Digest() != confirmed.ProjectionDefinition.Binding().Digest() ||
		!bytes.Equal(view.Projection().CanonicalBytes(), confirmed.ProjectionDefinition.CanonicalBytes()) ||
		!bytes.Equal(view.Projection().Binding().CanonicalBytes(), confirmed.ProjectionDefinition.Binding().CanonicalBytes()) {
		return errors.Join(executionErr, fmt.Errorf("CLI_STUDY_COMPLETED_SOURCE_REFUSED"))
	}

	bundle := completed.bundle
	bundleSource := bundle.PortableSource()
	predicate := bundle.Predicate()
	allowed := predicate.AllowedTuples()
	residueBundle := completed.residue.Bundle()
	if !completed.choicepoint.Valid() || !completed.decision.Valid() || !completed.durableRuling.Record().Valid() ||
		!bundle.Valid() || !completed.residue.Valid() ||
		bundleSource.Digest() != source.Digest() ||
		!bytes.Equal(bundleSource.CanonicalBytes(), source.CanonicalBytes()) ||
		!bundle.SourceProfile().ValidFor(source) || !predicate.ValidFor(source.Profile(), source.StimulusDigest()) ||
		predicate.StimulusDigest() != confirmed.Stimulus.Digest() ||
		!sameStringsInOrder(predicate.SelectedFields(), completed.decision.SelectedFields()) ||
		bundle.DecisionAction() != string(completed.decision.Action()) ||
		bundle.ChoicepointDigest() != completed.choicepoint.Digest() ||
		bundle.DecisionRecordDigest() != completed.decision.Digest() ||
		completed.durableRuling.Record().Digest() != completed.decision.Digest() ||
		!bytes.Equal(completed.durableRuling.Record().CanonicalBytes(), completed.decision.CanonicalBytes()) ||
		completed.residue.BundleDigest() != bundle.Digest() ||
		completed.residue.DecisionRecordDigest() != completed.decision.Digest() ||
		completed.residue.ChoicepointDigest() != completed.choicepoint.Digest() ||
		!residueBundle.Valid() || residueBundle.Digest() != bundle.Digest() ||
		!bytes.Equal(residueBundle.CanonicalBytes(), bundle.CanonicalBytes()) ||
		len(allowed) != len(completed.choiceAudit.allowedTupleBytes) {
		return fmt.Errorf("CLI_STUDY_COMPLETED_CHOICE_REFUSED")
	}
	for index, tuple := range allowed {
		if !tuple.Valid() || !bytes.Equal(tuple.CanonicalBytes(), completed.choiceAudit.allowedTupleBytes[index]) {
			return fmt.Errorf("CLI_STUDY_COMPLETED_PREDICATE_TUPLE_REFUSED")
		}
	}
	if constructionProof {
		if err := validateChoiceAuditProof(completed.choiceAudit, completed.choicepoint, completed.decision, bundle); err != nil {
			return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_CHOICE_REFUSED"))
		}
	} else if err := validateChoiceAuditStatic(completed.choiceAudit, completed.choicepoint, completed.decision, bundle); err != nil {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_CHOICE_REFUSED"))
	}
	return nil
}

func validateChoicepointCompletionJoin(completed CompletedCLIStudy) error {
	choicepoint := completed.choicepoint
	confirmed := completed.confirmed
	draft := confirmed.Confirmation.Draft()
	completedRecord := draft.Record()
	choiceRecord := choicepoint.ConfirmationRecord()
	choicePlan := choicepoint.WorldPlan()
	minimized := choicepoint.MinimizedStimulus()
	rebuiltMinimized, minimizedErr := choice.NewCanonicalArtifact(
		minimized.Kind(), minimized.Digest(), minimized.CanonicalBytes(),
	)
	originalMap := choiceRecord.OriginalBaselineMap()
	reducedMap := choiceRecord.ReducedBaselineMap()
	confirmedMap := choiceRecord.ConfirmedMap()
	weakBaseline := completed.weakRun.Baseline().OutcomeMap()
	if minimizedErr != nil || !choicepoint.Valid() || !confirmed.HasConfirmation || !draft.Valid() ||
		!completedRecord.Valid() || !choiceRecord.Valid() || !choicePlan.Digest().Valid() ||
		choicePlan.Digest() != confirmed.Plan.Digest() ||
		!bytes.Equal(choicePlan.CanonicalBytes(), confirmed.Plan.CanonicalBytes()) ||
		choicePlan.ComparisonEnvelopeDigest() != confirmed.Envelope.Digest() ||
		choicepoint.ConfirmationDigest() != draft.Digest() ||
		choicepoint.ConfirmationDigest() != completedRecord.Digest() ||
		choiceRecord.Digest() != completedRecord.Digest() ||
		!bytes.Equal(choiceRecord.CanonicalBytes(), completedRecord.CanonicalBytes()) ||
		choiceRecord.PlanDigest() != confirmed.Plan.Digest() ||
		choiceRecord.ReductionRunDigest() != completed.weakRun.Digest() ||
		minimized.Kind() != "CLIStimulus" || rebuiltMinimized.Digest() != minimized.Digest() ||
		!bytes.Equal(rebuiltMinimized.CanonicalBytes(), minimized.CanonicalBytes()) ||
		minimized.Digest() != confirmed.Stimulus.Digest() ||
		!bytes.Equal(minimized.CanonicalBytes(), confirmed.Stimulus.CanonicalBytes()) ||
		minimized.Digest() != completed.minimized.Stimulus.Digest() ||
		!bytes.Equal(minimized.CanonicalBytes(), completed.minimized.Stimulus.CanonicalBytes()) ||
		choicepoint.ConfirmedOutcomeMapDigest() != confirmed.OutcomeMap.ArtifactDigest() ||
		choiceRecord.ConfirmedArtifactDigest() != confirmed.OutcomeMap.ArtifactDigest() ||
		confirmedMap.ArtifactDigest() != confirmed.OutcomeMap.ArtifactDigest() ||
		!bytes.Equal(confirmedMap.CanonicalBytes(), confirmed.OutcomeMap.CanonicalBytes()) ||
		reducedMap.ArtifactDigest() != completed.minimized.OutcomeMap.ArtifactDigest() ||
		!bytes.Equal(reducedMap.CanonicalBytes(), completed.minimized.OutcomeMap.CanonicalBytes()) ||
		originalMap.ArtifactDigest() != completed.baseline.OutcomeMap.ArtifactDigest() ||
		!bytes.Equal(originalMap.CanonicalBytes(), completed.baseline.OutcomeMap.CanonicalBytes()) ||
		originalMap.ArtifactDigest() != weakBaseline.ArtifactDigest() ||
		!bytes.Equal(originalMap.CanonicalBytes(), weakBaseline.CanonicalBytes()) ||
		originalMap.StimulusDigest() != completed.baseline.Stimulus.Digest() {
		return errors.Join(minimizedErr, fmt.Errorf("CLI_STUDY_CHOICEPOINT_COMPLETION_JOIN_REFUSED"))
	}
	return nil
}

func validateChoiceAuditStatic(
	audit choiceAudit,
	record choice.ChoicepointRecord,
	decision choice.DecisionRecord,
	bundle emitmodel.ContractBundle,
) error {
	reopened, err := choice.ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || reopened.Digest() != record.Digest() ||
		!bytes.Equal(reopened.CanonicalBytes(), record.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_CHOICEPOINT_STATIC_REOPEN_REFUSED"))
	}
	reopenedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), reopened)
	if err != nil || reopenedDecision.Digest() != decision.Digest() ||
		!bytes.Equal(reopenedDecision.CanonicalBytes(), decision.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_DECISION_STATIC_REOPEN_REFUSED"))
	}
	replayedDecision, err := replayCLIChoiceDecision(reopened)
	if err != nil || replayedDecision.Digest() != reopenedDecision.Digest() ||
		!bytes.Equal(replayedDecision.CanonicalBytes(), reopenedDecision.CanonicalBytes()) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_CHOICE_STATIC_REPLAY_REFUSED"))
	}
	if !bundle.Valid() || !exactCLIAllowDecision(reopenedDecision) || !exactCLIBundlePredicate(bundle) ||
		audit.decisionDigest != reopenedDecision.Digest() || audit.earlyReveal ||
		audit.blindState != choice.SessionBlindOpen || audit.provisionalState != choice.SessionProvisionalRecorded ||
		audit.revealedState != choice.SessionRevealed || audit.finalState != choice.SessionFinalized ||
		audit.ambiguityCode != choice.CodeAmbiguousScope ||
		audit.noncompilableCode != string(choice.CodeNoncompilablePredicate) ||
		len(audit.destinationBefore) != 0 || len(audit.destinationAfter) != 0 ||
		!sameStringsInOrder(audit.destinationBefore, audit.destinationAfter) {
		return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_AUDIT_REFUSED")
	}
	allowed := bundle.Predicate().AllowedTuples()
	if len(allowed) != 2 || len(audit.allowedTupleBytes) != len(allowed) {
		return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_TUPLE_COUNT_REFUSED")
	}
	for index, tuple := range allowed {
		if !tuple.Valid() || !bytes.Equal(tuple.CanonicalBytes(), audit.allowedTupleBytes[index]) {
			return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_TUPLE_REFUSED")
		}
	}
	wantActions := []choice.Action{
		choice.ActionAllowObserved, choice.ActionCustomExpectation,
		choice.ActionRejectAll, choice.ActionDefer, choice.ActionRefine,
	}
	if len(audit.actions) != len(wantActions) {
		return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_ACTION_COUNT_REFUSED")
	}
	for index, fact := range audit.actions {
		if fact.action != wantActions[index] || !fact.decision.Valid() || fact.decision.Action() != fact.action {
			return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_ACTION_REFUSED: %d", index)
		}
		parsed, parseErr := choice.ParseDecisionRecord(fact.decision.CanonicalBytes(), reopened)
		if parseErr != nil || parsed.Digest() != fact.decision.Digest() ||
			!bytes.Equal(parsed.CanonicalBytes(), fact.decision.CanonicalBytes()) {
			return errors.Join(parseErr, fmt.Errorf("CLI_STUDY_CHOICE_STATIC_ACTION_PARSE_REFUSED: %d", index))
		}
		switch fact.action {
		case choice.ActionAllowObserved:
			if parsed.Digest() != reopenedDecision.Digest() || parsed.Digest() != replayedDecision.Digest() ||
				!bytes.Equal(parsed.CanonicalBytes(), reopenedDecision.CanonicalBytes()) ||
				!exactCLIAllowDecision(parsed) {
				return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_ALLOW_REFUSED")
			}
			if _, inspectErr := choice.InspectPortableRuling(parsed); inspectErr != nil {
				return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_ALLOW_INSPECTION_REFUSED: %v", inspectErr)
			}
		case choice.ActionCustomExpectation:
			if !exactCLICustomDecision(reopened, parsed) || fact.customReviewer != cliCustomReviewer ||
				fact.customReviewEvidence != reopened.ConfirmationDigest() {
				return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_CUSTOM_REFUSED")
			}
			if _, inspectErr := choice.InspectPortableRuling(parsed); inspectErr != nil {
				return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_CUSTOM_INSPECTION_REFUSED: %v", inspectErr)
			}
		default:
			expected, expectedErr := finalizeEarlyChoice(
				reopened,
				choice.RulingDraftInput{Action: fact.action, SelectedFields: []string{}, AllowedAliases: []string{}},
				"Noncompilable control action "+string(fact.action)+".",
			)
			_, replayErr := choice.InspectPortableRuling(parsed)
			_, compilable := parsed.CompilableRuling()
			wantBase := "noncompilable-" + strings.ToLower(string(fact.action))
			if expectedErr != nil || expected.Digest() != parsed.Digest() ||
				!bytes.Equal(expected.CanonicalBytes(), parsed.CanonicalBytes()) ||
				compilable || !parsed.EarlyReveal() ||
				fact.preparationRefusalCode != string(choice.CodeNoncompilablePredicate) ||
				!choice.IsRefusal(fact.preparationErr, choice.CodeNoncompilablePredicate) ||
				!choice.IsRefusal(replayErr, choice.CodeNoncompilablePredicate) ||
				fact.preparationErr.Error() != replayErr.Error() ||
				!filepath.IsAbs(fact.destinationPath) || filepath.Clean(fact.destinationPath) != fact.destinationPath ||
				filepath.Base(fact.destinationPath) != wantBase ||
				len(fact.destinationBefore) != 0 || len(fact.destinationAfter) != 0 ||
				!sameStringsInOrder(fact.destinationBefore, fact.destinationAfter) {
				return fmt.Errorf("CLI_STUDY_CHOICE_STATIC_NONCOMPILABLE_REFUSED: %s", fact.action)
			}
		}
	}
	return nil
}

func validateCompletedCLIStudy(completed CompletedCLIStudy) error {
	return validateCompletedCLIStudyWithChoiceProof(completed, false)
}

func validateCompletedCLIStudyConstruction(completed CompletedCLIStudy) error {
	return validateCompletedCLIStudyWithChoiceProof(completed, true)
}

func validateCompletedCLIStudyWithChoiceProof(completed CompletedCLIStudy, constructionProof bool) error {
	if completed.seal != issuedCompletedCLIStudy || completed.ordinal < 1 || completed.ordinal > 3 ||
		!completed.physicalRunAuthority.Valid() {
		return fmt.Errorf("CLI_STUDY_COMPLETED_CORE_REFUSED")
	}
	results := []Result{
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged,
		completed.baselineCheckpoint, completed.baseline, completed.mainEvaluation,
		completed.auxiliaryBaseline, completed.auxiliaryEvaluation,
		completed.confirmed, completed.auxiliaryConfirmed,
	}
	for _, result := range results {
		if err := validateStudyEnvelope(result.Envelope); err != nil {
			return err
		}
		if err := validateResultOwnership(result); err != nil {
			return err
		}
	}
	if validateStableDivergence(completed.discovery, 3, 3, domain.AttemptDiscovery) != nil ||
		validateEligibilityPair(completed.recoveryOne, completed.recoveryTwo) != nil ||
		validateStableDivergence(completed.shapeReference, 3, 1, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(completed.shapeChanged, 3, 1, domain.AttemptReduction) != nil ||
		validateStableDivergence(completed.baselineCheckpoint, 3, 3, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(completed.baseline, 3, 3, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(completed.mainEvaluation, 3, 3, domain.AttemptReduction) != nil ||
		validateStableDivergence(completed.auxiliaryBaseline, 2, 1, domain.AttemptDiscovery) != nil ||
		validateStableDivergence(completed.auxiliaryEvaluation, 2, 1, domain.AttemptReduction) != nil {
		return fmt.Errorf("CLI_STUDY_COMPLETED_PHASE_REFUSED")
	}
	shortStimulus, err := shortEligibilityStimulus(completed.discovery.Stimulus)
	if err != nil || completed.recoveryOne.Stimulus.Digest() != shortStimulus.Digest() ||
		completed.recoveryTwo.Stimulus.Digest() != completed.discovery.Stimulus.Digest() ||
		completed.recoveryOne.OutcomeMap.ComparisonBasisDigest() != completed.recoveryTwo.OutcomeMap.ComparisonBasisDigest() ||
		completed.recoveryOne.CapturePolicy.StdoutBytes() != 30 || completed.recoveryTwo.CapturePolicy.StdoutBytes() != 30 ||
		completed.recoveryOne.CapturePolicy.StderrBytes() != 64<<10 ||
		completed.recoveryTwo.CapturePolicy.StderrBytes() != 64<<10 ||
		validateFreshDisjoint(completed.discovery, completed.recoveryOne, completed.recoveryTwo) != nil {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_ELIGIBILITY_REFUSED"))
	}
	shapeAssessment := compare.AssessPreservation(completed.shapeReference.OutcomeMap, completed.shapeChanged.OutcomeMap)
	expectedShapeStimulus, shapeStimulusErr := stimulusWithConfigMode(completed.discovery.Stimulus, "shape-changed")
	if shapeStimulusErr != nil || completed.shapeReference.Stimulus.Digest() != completed.discovery.Stimulus.Digest() ||
		completed.shapeChanged.Stimulus.Digest() != expectedShapeStimulus.Digest() ||
		!shapeAssessment.Valid() || shapeAssessment.Relation() != compare.PreservationDifferent ||
		shapeAssessment.ReasonCode() != "EXACT_PRESERVATION_MAP_CHANGED" ||
		completed.shapeReference.Plan.Digest() != completed.shapeChanged.Plan.Digest() ||
		completed.shapeReference.OutcomeMap.DistinctProjectionCount() != completed.shapeChanged.OutcomeMap.DistinctProjectionCount() ||
		validateFreshDisjoint(completed.shapeReference, completed.shapeChanged) != nil {
		return errors.Join(shapeStimulusErr, fmt.Errorf("CLI_STUDY_COMPLETED_SHAPE_REFUSED"))
	}
	baselineAssessment := compare.AssessPreservation(completed.baselineCheckpoint.OutcomeMap, completed.baseline.OutcomeMap)
	if !baselineAssessment.Valid() || baselineAssessment.Relation() != compare.PreservationEqual ||
		baselineAssessment.ReasonCode() != "EXACT_PRESERVATION_MAP_MATCH" ||
		completed.baselineCheckpoint.Plan.Digest() != completed.baseline.Plan.Digest() ||
		bytes.Equal(completed.baselineCheckpoint.OutcomeMap.CanonicalBytes(), completed.baseline.OutcomeMap.CanonicalBytes()) ||
		validateFreshDisjoint(completed.baselineCheckpoint, completed.baseline) != nil {
		return fmt.Errorf("CLI_STUDY_COMPLETED_BASELINE_RECOVERY_REFUSED")
	}
	if completed.minimized.OutcomeMap.ArtifactDigest() != completed.mainEvaluation.OutcomeMap.ArtifactDigest() ||
		completed.minimized.Stimulus.Digest() != completed.mainEvaluation.Stimulus.Digest() ||
		completed.auxiliaryMinimized.OutcomeMap.ArtifactDigest() != completed.auxiliaryEvaluation.OutcomeMap.ArtifactDigest() ||
		completed.auxiliaryMinimized.Stimulus.Digest() != completed.auxiliaryEvaluation.Stimulus.Digest() ||
		completed.baseline.Plan.Digest() != completed.minimized.Plan.Digest() ||
		completed.minimized.Plan.Digest() != completed.confirmed.Plan.Digest() ||
		completed.auxiliaryBaseline.Plan.Digest() != completed.auxiliaryMinimized.Plan.Digest() ||
		completed.auxiliaryMinimized.Plan.Digest() != completed.auxiliaryConfirmed.Plan.Digest() {
		return fmt.Errorf("CLI_STUDY_COMPLETED_PLAN_JOIN_REFUSED")
	}
	if err := validateWeakReductionFacts(completed.baseline, completed.mainEvaluation, completed.weakRun, completed.weakGrade); err != nil {
		return err
	}
	if err := validateStrongReductionFacts(completed.auxiliaryBaseline, completed.auxiliaryEvaluation, completed.strongRun, completed.strongGrade); err != nil {
		return err
	}
	if err := validateReductionCohortTopology(
		completed.weakRun.Digest(), completed.strongRun.Digest(),
		completed.weakRun.Transcript().Digest(), completed.strongRun.Transcript().Digest(),
		completed.weakGrade.Grade().Digest(), completed.strongGrade.Grade().Digest(),
		completed.weakRun.MinimizedStimulusDigest(), completed.strongRun.MinimizedStimulusDigest(),
		completed.discovery.Stimulus.Digest(),
		completed.minimized.Stimulus.CanonicalBytes(),
		completed.auxiliaryMinimized.Stimulus.CanonicalBytes(),
		completed.discovery.Stimulus.CanonicalBytes(),
	); err != nil {
		return err
	}
	if err := validateConfirmationReduction(completed.minimized, completed.confirmed, completed.weakRun, completed.weakGrade, 3, 2); err != nil {
		return err
	}
	if err := validateConfirmationReduction(completed.auxiliaryMinimized, completed.auxiliaryConfirmed, completed.strongRun, completed.strongGrade, 2, 1); err != nil {
		return err
	}
	if err := validateControlRoster(completed.discovery.Stimulus, completed.semanticControls); err != nil {
		return err
	}
	if err := validateGlobalPhysicalNonalias(completed); err != nil {
		return err
	}
	if completed.overlayAudit.refusalCode != countercli.CodeFixturePath ||
		completed.overlayAudit.attemptsBefore != 0 || completed.overlayAudit.attemptsAfter != 0 ||
		completed.overlayAudit.worldsBefore != 0 || completed.overlayAudit.worldsAfter != 0 ||
		completed.overlayAudit.attemptsBefore != completed.overlayAudit.attemptsAfter ||
		completed.overlayAudit.worldsBefore != completed.overlayAudit.worldsAfter {
		return fmt.Errorf("CLI_STUDY_COMPLETED_OVERLAY_REFUSED")
	}
	if err := validateChoicePublicationJoins(completed, constructionProof); err != nil {
		return err
	}
	if err := validateOfficialTrialRoster(
		completed.officialTrials, completed.bundle.Digest(), completed.residue.HeadDigest(),
	); err != nil {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_OFFICIAL_STATIC_REFUSED"))
	}
	rebuiltFacts, err := buildPhaseFacts(
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint, completed.baseline,
		completed.mainEvaluation, completed.auxiliaryBaseline, completed.auxiliaryEvaluation,
		completed.semanticControls, completed.confirmed, completed.auxiliaryConfirmed, completed.officialTrials,
	)
	if err != nil || !samePhaseFacts(rebuiltFacts, completed.phaseFacts) {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_PHASE_FACTS_REFUSED"))
	}
	authority, err := physicalRunAuthority(completed)
	if err != nil || authority != completed.physicalRunAuthority {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_COMPLETED_AUTHORITY_REFUSED"))
	}
	return nil
}

func revalidateCompletedOfficialTrials(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
	completed CompletedCLIStudy,
) error {
	if ctx == nil || objectStore == nil || !residue.Valid() || !bundle.Valid() ||
		residue.BundleDigest() != bundle.Digest() {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_TERMINAL_CONTEXT_REFUSED")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_CONTEXT_ENDED_BEFORE_OFFICIAL_REVALIDATION: %w", err)
	}
	if err := revalidateOfficialTrialRosterShared(
		ctx, config, authorityRoot, objectStore, residue, bundle, completed.officialTrials,
	); err != nil {
		return errors.Join(err, fmt.Errorf("CLI_STUDY_OFFICIAL_TERMINAL_REVALIDATION_REFUSED"))
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_CONTEXT_ENDED_AFTER_OFFICIAL_REVALIDATION: %w", err)
	}
	return nil
}

func samePhaseFacts(left, right []phaseFact) bool {
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

func publishCompletedEvidence(
	ctx context.Context,
	ready publicationReadyCLIStudy,
	workspace *app.EvidenceWorkspace,
) (StudyInspection, error) {
	if ctx == nil || workspace == nil || !ready.valid() {
		return StudyInspection{}, fmt.Errorf("CLI_STUDY_PUBLICATION_REFUSED")
	}
	if err := ctx.Err(); err != nil {
		return StudyInspection{}, fmt.Errorf("CLI_STUDY_PUBLICATION_CONTEXT_REFUSED: %w", err)
	}
	publication, err := publishEvidenceInput(ctx, ready.input, workspace)
	if err != nil {
		return StudyInspection{}, err
	}
	return mergeReadyStudyPublicationAfterValidation(ready, publication)
}

func deterministicChoiceEvidencePayloads(
	decision choice.DecisionRecord,
	bundle emitmodel.ContractBundle,
	allowedTupleSHA []string,
) (evidencePayload, evidencePayload, evidencePayload) {
	selectedFields := decision.SelectedFields()
	predicateSHA := sha256Hex(bundle.Predicate().CanonicalBytes())
	decisionSemanticSHA := semanticProjectionSHA256(
		"decision-record",
		string(decision.Action()),
		strings.Join(selectedFields, "\x00"),
		predicateSHA,
		fmt.Sprintf("early-reveal=%t", decision.EarlyReveal()),
	)
	bundleSemanticSHA := semanticProjectionSHA256(
		"contract-bundle",
		bundle.PortableSource().Digest().String(),
		predicateSHA,
		"six-files",
		string(decision.Action()),
	)
	return evidencePayload{
			"action":               string(decision.Action()),
			"allowed_tuple_sha256": append([]string(nil), allowedTupleSHA...),
			"authority":            "DETERMINISTIC_CORRELATED_TUPLE_RULING_PROJECTION_FROM_SEALED_DECISION",
			"predicate_sha256":     predicateSHA,
			"projection_kind":      "SEMANTIC_REGRESSION_PROJECTION",
			"selected_fields":      append([]string(nil), selectedFields...),
		}, evidencePayload{
			"authority":              "DETERMINISTIC_DECISION_SEMANTIC_PROJECTION_FROM_STRICT_RECORD",
			"canonical_record_valid": decision.Valid(),
			"early_reveal":           decision.EarlyReveal(),
			"projection_kind":        "SEMANTIC_REGRESSION_PROJECTION",
			"selected_fields":        append([]string(nil), selectedFields...),
			"semantic_sha256":        decisionSemanticSHA,
		}, evidencePayload{
			"authority":              "DETERMINISTIC_CONTRACT_SEMANTIC_PROJECTION_FROM_STRICT_SIX_FILE_BUNDLE",
			"canonical_bundle_valid": bundle.Valid(),
			"member_count":           len(bundle.Files()),
			"predicate_sha256":       predicateSHA,
			"projection_kind":        "SEMANTIC_REGRESSION_PROJECTION",
			"selected_fields":        append([]string(nil), selectedFields...),
			"semantic_sha256":        bundleSemanticSHA,
		}
}

func semanticProjectionSHA256(kind string, parts ...string) string {
	framed := append([]string{"countershape/u7c/semantic-regression-projection/v1", kind}, parts...)
	return sha256Hex([]byte(strings.Join(framed, "\x00") + "\x00"))
}

func evidenceInputForCompleted(completed CompletedCLIStudy) evidencePublicationInput {
	physical := completed.physicalRunAuthority.String()
	phaseReceipts := make([]phaseEvidenceReceipt, len(completed.phaseFacts))
	phaseKinds := make([]string, len(completed.phaseFacts))
	worlds := make([]string, 0, 88)
	attempts := make([]string, 0, 88)
	measurements := make([]string, 0, 88)
	captures := make([]string, 0, 88)
	projections := make([]string, 0, 80)
	for index, fact := range completed.phaseFacts {
		phaseReceipts[index] = phaseEvidenceReceipt{phase: fact.phase, trial: fact.trial, authority: fact.authority}
		phaseKinds[index] = fact.phase + ":" + fmt.Sprintf("%03d", fact.trial) + ":" + fact.kind
		if fact.world.Valid() {
			worlds = append(worlds, fact.world.String())
		}
		if fact.attempt.Valid() {
			attempts = append(attempts, fact.attempt.String())
		}
		if fact.measurement.Valid() {
			measurements = append(measurements, fact.measurement.String())
		}
		if fact.capture.Valid() {
			captures = append(captures, fact.capture.String())
		}
		if fact.projection.Valid() {
			projections = append(projections, fact.projection.String())
		}
	}
	allowedTupleSHA := make([]string, len(completed.choiceAudit.allowedTupleBytes))
	for index, exact := range completed.choiceAudit.allowedTupleBytes {
		allowedTupleSHA[index] = sha256Hex(exact)
	}
	rulingPayload, decisionPayload, bundlePayload := deterministicChoiceEvidencePayloads(
		completed.decision,
		completed.bundle,
		allowedTupleSHA,
	)
	targetDigests := make([]string, len(completed.officialTrials))
	targetSHA := make([]string, len(completed.officialTrials))
	runDigests := make([]string, len(completed.officialTrials))
	runSHA := make([]string, len(completed.officialTrials))
	classificationDigests := make([]string, len(completed.officialTrials))
	classificationSHA := make([]string, len(completed.officialTrials))
	results := make([]string, len(completed.officialTrials))
	for index, trial := range completed.officialTrials {
		targetDigests[index], targetSHA[index] = trial.targetDigest.String(), trial.targetCanonicalSHA256
		runDigests[index], runSHA[index] = trial.runDigest.String(), trial.runCanonicalSHA256
		classificationDigests[index], classificationSHA[index] = trial.classificationDigest.String(), trial.classificationCanonicalSHA256
		results[index] = trial.result
	}
	return evidencePublicationInput{
		ordinal: completed.ordinal, physicalRunAuthority: completed.physicalRunAuthority, phaseTrials: phaseReceipts,
		deterministic: deterministicEvidencePayloads{
			sourceSpec:     evidencePayload{"authority": "typed-source-spec", "digest": completed.confirmed.SourceSpecDigest.String(), "canonical_sha256": sha256Hex(completed.confirmed.SourceSpecBytes)},
			worldPlan:      evidencePayload{"authority": "compiled-world-plan", "digest": completed.confirmed.Plan.Digest().String(), "canonical_sha256": sha256Hex(completed.confirmed.Plan.CanonicalBytes())},
			ruling:         rulingPayload,
			decisionRecord: decisionPayload,
			contractBundle: bundlePayload,
		},
		fresh: freshEvidencePayloads{
			worldInstance:  evidencePayload{"authority": "physical-world-instances", "physical_run_authority": physical, "world_digests": worlds, "nonclaim": "HOSTILE_CONTAINMENT_UNVALIDATED"},
			attempts:       evidencePayload{"authority": "physical-attempts", "physical_run_authority": physical, "attempt_digests": attempts, "phase_kinds": phaseKinds},
			measurements:   evidencePayload{"authority": "typed-instance-measurements", "physical_run_authority": physical, "measurement_digests": measurements, "nonclaim": "FULL_STUDY_RESOURCE_BOUND_UNVALIDATED"},
			captures:       evidencePayload{"authority": "typed-cli-captures", "physical_run_authority": physical, "capture_digests": captures, "projection_digests": projections},
			confirmation:   evidencePayload{"authority": "fresh-main-and-auxiliary-confirmations", "physical_run_authority": physical, "main_digest": completed.confirmed.Confirmation.Draft().Digest().String(), "auxiliary_digest": completed.auxiliaryConfirmed.Confirmation.Draft().Digest().String()},
			target:         evidencePayload{"authority": "official-default-exit-zero-targets", "physical_run_authority": physical, "target_digests": targetDigests, "canonical_sha256": targetSHA},
			finalized:      evidencePayload{"authority": "official-durable-runs", "physical_run_authority": physical, "run_digests": runDigests, "canonical_sha256": runSHA},
			classification: evidencePayload{"authority": "official-cli-classifications", "physical_run_authority": physical, "classification_digests": classificationDigests, "canonical_sha256": classificationSHA, "results": results, "nonclaim": "CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED"},
		},
	}
}
