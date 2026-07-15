package world

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

const (
	// CLIExecutionAuthorityV1 is present only on receipts produced from the
	// opaque U3 CLIExecutionBinding path. Legacy U2 Execute receipts leave it empty.
	CLIExecutionAuthorityV1 = "U3_OPAQUE_CLI_EXECUTION_BINDING_V1"
	fixtureRootEnvironment  = "COUNTERSHAPE_FIXTURE_ROOT"
	scheduleEnvironment     = "COUNTERSHAPE_SCHEDULE_ORDINAL"
	repetitionEnvironment   = "COUNTERSHAPE_SCHEDULE_REPETITION"

	diagnosticCLIFixtureOverlayFailed = "CLI_FIXTURE_OVERLAY_FAILED"
)

// CLIRequest carries one opaque execution binding plus physical capabilities.
// There is intentionally no separately caller-paired plan, stimulus digest,
// argv, environment, stdin, fixture list, or cwd value.
type CLIRequest struct {
	Binding         climodel.CLIExecutionBinding
	Candidate       gitobj.BoundCandidate
	Tools           ToolRegistry
	AllocationRoot  string
	Purpose         domain.AttemptPurpose
	InstanceNonce   string
	ScheduleOrdinal int
}

func ExecuteCLI(ctx context.Context, request CLIRequest) (Result, error) {
	return executeCLIWithMaterializer(ctx, request, gitobj.DefaultMaterializer{})
}

func executeCLIWithMaterializer(ctx context.Context, request CLIRequest, source materializer) (Result, error) {
	derived, lineage, err := deriveCLIRequest(request)
	if err != nil {
		return Result{}, err
	}
	tool, err := derived.Tools.resolvePlan(derived.Plan)
	if err != nil {
		return Result{}, err
	}
	allocated, err := allocateAttempt(derived)
	if err != nil {
		return Result{}, err
	}
	attempt := allocated.domainAttempt
	attempt, err = attempt.Advance(domain.AttemptMaterializing)
	if err != nil {
		return Result{}, err
	}

	materialization, materializeErr := source.Materialize(ctx, derived.Candidate, allocated.roots.candidateParent)
	if materializeErr != nil {
		primary := materializationControl(ctx, materializeErr)
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, gitobj.MaterializationReceipt{}, primary,
			materializationDiagnostic(ctx, materializeErr), tool, lineage,
		)
	}
	if ctx.Err() != nil {
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledAfterMaterialization, ctx.Err()), tool, lineage,
		)
	}
	if err := validateMaterialization(ctx, derived, allocated.roots, materialization); err != nil {
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, gitobj.MaterializationReceipt{}, domain.ControlMaterializationError, err, tool, lineage,
		)
	}

	fixtureReceipt, err := materializeCLIFixtures(request.Binding, allocated.roots.fixture)
	if err != nil {
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlMaterializationError,
			withReceiptDiagnostic(diagnosticCLIFixtureOverlayFailed, err), tool, lineage,
		)
	}
	lineage.fixtureOverlayDigest = fixtureReceipt.Digest()
	lineage.workingDirectory = materialization.PublishedRoot
	environment, err := buildCLIEnvironment(
		derived.Plan.Environment(), request.Binding.Environment(), allocated.roots,
		allocated.attemptID, request.ScheduleOrdinal, derived.Plan.Budgets().CandidateCount,
	)
	if err != nil {
		return Result{}, err
	}
	attempt, err = attempt.Advance(domain.AttemptStarting)
	if err != nil {
		return Result{}, err
	}
	if err := requireDurableMarker(allocated); err != nil {
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlStartError, err, tool, lineage,
		)
		return attachCLIArtifacts(result, fixtureReceipt, CLIInvocationEvidenceReceipt{}), finalizeErr
	}
	if ctx.Err() != nil {
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledBeforeSpawn, ctx.Err()), tool, lineage,
		)
		return attachCLIArtifacts(result, fixtureReceipt, CLIInvocationEvidenceReceipt{}), finalizeErr
	}

	logicalArgv := request.Binding.LogicalArgv()
	physicalRequest := processRequest{
		tool: tool, logicalArgv: logicalArgv, environment: environment,
		stdin: cliProcessStdin(request.Binding.Stdin()), cwd: lineage.workingDirectory,
		stdoutLimit:       request.Binding.CapturePolicy().StdoutBytes(),
		stderrLimit:       request.Binding.CapturePolicy().StderrBytes(),
		executionBudgetMS: derived.Plan.Budgets().ProbeMS,
		teardownBudgetMS:  derived.Plan.Budgets().TeardownMS,
		markerBeforeSpawn: true,
		onGroupOwned: func() error {
			for _, next := range []domain.AttemptState{
				domain.AttemptReady, domain.AttemptProbing, domain.AttemptCapturing,
			} {
				attempt, err = attempt.Advance(next)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	physical := runProcess(ctx, physicalRequest)
	if !physical.physicalExecutionEntered {
		primary := physical.primary
		if primary == "" {
			primary = domain.ControlStartError
		}
		diagnosticCode := physical.diagnosticCode
		if diagnosticCode == "" {
			diagnosticCode = diagnosticGenericPreProcessStartFailure
		}
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, primary,
			withReceiptDiagnostic(diagnosticCode, nil), tool, lineage,
		)
		return attachCLIArtifacts(result, fixtureReceipt, CLIInvocationEvidenceReceipt{}), finalizeErr
	}
	if !physical.spawnAttempted {
		return Result{}, refuse(CodeCLIExecutionRejected, "physical entry lacks an exact spawn attempt", nil)
	}
	// Receipt the limits actually handed through the physical runner boundary,
	// not a second copy of the requested policy. A miswired runner input must
	// therefore fail even when the observed output stays below both caps.
	lineage.stdoutCaptureLimit = physical.stdoutCaptureLimit
	lineage.stderrCaptureLimit = physical.stderrCaptureLimit
	if !physical.started && physical.primary == "" {
		physical.primary = domain.ControlStartError
	}
	if err := bindPhysicalCLIStdin(&lineage, physical); err != nil {
		return Result{}, err
	}
	invocation, err := inspectCLIInvocationEvidence(
		request.Binding, allocated.roots.evidence, allocated.attemptID, logicalArgv,
	)
	if err != nil {
		return Result{}, err
	}
	lineage.invocationEvidenceDigest = invocation.Digest()
	lineage.invocationEvidenceStatus = invocation.Status()
	lineage.invocationEvidencePresence = invocation.Presence()
	lineage.invocationEvidenceValid = invocation.Status() == CLIInvocationValidated
	result, err := finalizePhysicalWithLineage(
		allocated, attempt, materialization, tool, logicalArgv, environment, physical, lineage,
	)
	if err != nil {
		return Result{}, err
	}
	return attachCLIArtifacts(result, fixtureReceipt, invocation), nil
}

func deriveCLIRequest(request CLIRequest) (Request, processCLILineage, error) {
	if !request.Binding.Valid() {
		return Request{}, processCLILineage{}, refuse(CodeCLIExecutionRejected, "opaque CLI execution binding is invalid", nil)
	}
	if err := validateCLIEnvironmentBindings(request.Binding.Plan().Environment(), request.Binding.Environment()); err != nil {
		return Request{}, processCLILineage{}, err
	}
	derived := Request{
		Plan: request.Binding.Plan(), Candidate: request.Candidate, Tools: request.Tools,
		AllocationRoot: request.AllocationRoot, StimulusDigest: request.Binding.StimulusDigest(),
		Purpose: request.Purpose, InstanceNonce: request.InstanceNonce, ScheduleOrdinal: request.ScheduleOrdinal,
	}
	if err := validateRequest(derived); err != nil {
		return Request{}, processCLILineage{}, err
	}
	candidateCount := derived.Plan.Budgets().CandidateCount
	repeatCount := derived.Plan.RepeatSchedule().DiscoveryRepeats
	if request.Purpose == domain.AttemptFinalSweep || request.Purpose == domain.AttemptConfirmation ||
		request.Purpose == domain.AttemptConformance {
		repeatCount = derived.Plan.RepeatSchedule().ConfirmationRepeats
	}
	if candidateCount < 2 || request.ScheduleOrdinal >= candidateCount*repeatCount {
		return Request{}, processCLILineage{}, refuse(CodeCLIExecutionRejected, "schedule ordinal is outside the plan-bound finite matrix", nil)
	}
	stdin := request.Binding.Stdin()
	stdinDigest, err := canon.DigestBytes("CLIStdinBytes", stdin.Bytes())
	if err != nil {
		return Request{}, processCLILineage{}, err
	}
	parsedStdinDigest, err := domain.ParseDigest(stdinDigest.String())
	if err != nil {
		return Request{}, processCLILineage{}, err
	}
	lineage := processCLILineage{
		authority: CLIExecutionAuthorityV1, declaredLogicalArgv: request.Binding.LogicalArgv(),
		executionBindingDigest: request.Binding.Digest(),
		stimulusDigest:         request.Binding.StimulusDigest(), executionPayloadDigest: request.Binding.ExecutionPayloadDigest(),
		fixtureRecipeDigest: request.Binding.FixtureRecipeDigest(),
		cwdPolicy:           string(request.Binding.CWDPolicy()),
		stdinPresence:       string(stdin.Presence()), stdinBytes: int64(len(stdin.Bytes())),
		stdinDigest: parsedStdinDigest, stdinDelivery: "NOT_APPLIED", scheduleOrdinal: request.ScheduleOrdinal,
		candidateCount: candidateCount, scheduleRepetition: request.ScheduleOrdinal / candidateCount,
		invocationEvidenceStatus:   CLIInvocationNotInspected,
		invocationEvidencePresence: CLIInvocationPresenceUnknown,
	}
	if !lineage.valid() {
		return Request{}, processCLILineage{}, refuse(CodeCLIExecutionRejected, "derived CLI receipt lineage is invalid", nil)
	}
	return derived, lineage, nil
}

func validateCLIEnvironmentBindings(
	planEntries []domain.EnvironmentEntry,
	bindings []climodel.CLIEnvironmentBinding,
) error {
	planNames := make(map[string]struct{}, len(planEntries))
	for _, entry := range planEntries {
		planNames[entry.Name] = struct{}{}
	}
	for _, binding := range bindings {
		if reservedCLIEnvironmentName(binding.Name()) {
			return refuse(CodeCLIEnvironmentRejected, "stimulus attempts to control runner-owned environment name", nil)
		}
		if _, collision := planNames[binding.Name()]; collision {
			return refuse(CodeCLIEnvironmentRejected, "stimulus environment name collides with immutable plan environment", nil)
		}
	}
	return nil
}

func reservedCLIEnvironmentName(name string) bool {
	switch name {
	case "HOME", "PATH", "TMPDIR", "TMP", "TEMP", "PWD", "OLDPWD", "SHELL",
		"NODE_OPTIONS", "BASH_ENV", "ENV", "RUBYOPT", "PERL5OPT", "PYTHONPATH", "PYTHONHOME", "GODEBUG":
		return true
	}
	// MUTATION_ANCHOR: cli-stimulus-must-not-control-runner-environment
	return strings.HasPrefix(name, "XDG_") || strings.HasPrefix(name, "COUNTERSHAPE_") ||
		strings.HasPrefix(name, "DYLD_")
}

func buildCLIEnvironment(
	planEntries []domain.EnvironmentEntry,
	bindings []climodel.CLIEnvironmentBinding,
	roots Roots,
	attemptID string,
	scheduleOrdinal int,
	candidateCount int,
) ([]string, error) {
	if candidateCount < 2 || scheduleOrdinal < 0 {
		return nil, refuse(CodeCLIExecutionRejected, "schedule environment inputs are invalid", nil)
	}
	if err := validateCLIEnvironmentBindings(planEntries, bindings); err != nil {
		return nil, err
	}
	values := make(map[string]string, len(planEntries)+len(bindings)+12)
	for _, entry := range planEntries {
		values[entry.Name] = entry.Value
	}
	for _, binding := range bindings {
		if binding.Present() {
			values[binding.Name()] = binding.Value()
		}
	}
	values["HOME"] = roots.home
	values["TMPDIR"] = roots.temporary
	values["XDG_CONFIG_HOME"] = roots.xdgConfig
	values["XDG_CACHE_HOME"] = roots.xdgCache
	values["XDG_DATA_HOME"] = roots.xdgData
	values["XDG_STATE_HOME"] = roots.xdgState
	values[stateRootEnvironment] = roots.state
	values[evidenceRootEnvironment] = roots.evidence
	values[attemptIDEnvironment] = attemptID
	values[fixtureRootEnvironment] = roots.fixture
	values[scheduleEnvironment] = strconv.Itoa(scheduleOrdinal)
	values[repetitionEnvironment] = strconv.Itoa(scheduleOrdinal / candidateCount)
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, len(names))
	for index, name := range names {
		environment[index] = name + "=" + values[name]
	}
	return environment, nil
}

func cliProcessStdin(stdin climodel.CLIStdin) processStdin {
	if !stdin.Present() {
		return processStdin{presence: processStdinAbsent}
	}
	return processStdin{presence: processStdinPresent, bytes: stdin.Bytes()}
}

// bindPhysicalCLIStdin accepts stdin lineage only from the physical exec.Cmd
// edge. The binding-derived declaration is used solely as the expected value;
// it cannot backfill missing or substituted execution evidence.
func bindPhysicalCLIStdin(lineage *processCLILineage, physical physicalProcessResult) error {
	if lineage == nil || lineage.physicalExecutionEntered || lineage.stdinDelivery != "NOT_APPLIED" ||
		!physical.physicalExecutionEntered || !physical.stdinDigest.Valid() ||
		string(physical.stdinPresence) != lineage.stdinPresence ||
		physical.stdinDeclared != lineage.stdinBytes || physical.stdinDigest != lineage.stdinDigest {
		return refuse(CodeCLIExecutionRejected, "physical stdin identity differs from the opaque binding", nil)
	}
	delivery := "NOT_APPLIED"
	if physical.started {
		delivery = "ABSENT_NULL_DEVICE"
		if physical.stdinPresence == processStdinPresent {
			delivery = "PRESENT_EXPLICIT_PIPE_WRITER"
		}
	}
	if physical.stdinWritten < 0 || physical.stdinWritten > physical.stdinDeclared {
		return refuse(CodeCLIExecutionRejected, "physical stdin delivery is invalid", nil)
	}
	switch physical.stdinPresence {
	case processStdinAbsent:
		if physical.stdinPipeAllocated || physical.stdinWriterStarted || physical.stdinHandoffAttempted ||
			physical.stdinWritten != 0 || !physical.stdinComplete || physical.stdinErrorCode != "" {
			return refuse(CodeCLIExecutionRejected, "absent stdin acquired physical delivery state", nil)
		}
	case processStdinPresent:
		if physical.stdinWriterStarted != physical.stdinHandoffAttempted ||
			(physical.stdinWriterStarted && !physical.stdinPipeAllocated) ||
			(physical.started && (!physical.stdinPipeAllocated || !physical.stdinWriterStarted || !physical.stdinHandoffAttempted)) ||
			(!physical.started && (physical.stdinWriterStarted || physical.stdinHandoffAttempted ||
				physical.stdinWritten != 0 || physical.stdinComplete || physical.stdinErrorCode != "")) ||
			(physical.stdinComplete && (physical.stdinWritten != physical.stdinDeclared || physical.stdinErrorCode != "")) ||
			(physical.started && !physical.stdinComplete && physical.stdinErrorCode == "") {
			return refuse(CodeCLIExecutionRejected, "present stdin lacks exact physical pipe delivery evidence", nil)
		}
	default:
		return refuse(CodeCLIExecutionRejected, "physical stdin presence is not a U3 state", nil)
	}
	lineage.stdinPresence = string(physical.stdinPresence)
	lineage.physicalExecutionEntered = true
	lineage.stdinBytes = physical.stdinDeclared
	lineage.stdinDigest = physical.stdinDigest
	lineage.stdinDelivery = delivery
	lineage.stdinPipeAllocated = physical.stdinPipeAllocated
	lineage.stdinWriterStarted = physical.stdinWriterStarted
	lineage.stdinHandoffAttempted = physical.stdinHandoffAttempted
	lineage.stdinWritten = physical.stdinWritten
	lineage.stdinDeliveryComplete = physical.stdinPresence == processStdinPresent && physical.stdinComplete
	lineage.stdinDeliveryError = physical.stdinErrorCode
	return nil
}

func attachCLIArtifacts(
	result Result,
	fixture CLIFixtureOverlayReceipt,
	invocation CLIInvocationEvidenceReceipt,
) Result {
	if fixture.Valid() {
		result.cliFixture = fixture
		result.hasCLIFixture = true
	}
	if invocation.Valid() {
		result.cliInvocation = invocation
		result.hasCLIInvocation = true
	}
	return result
}

// retained for error-chain assertions without allowing arbitrary diagnostic
// text into a receipt identity.
func isCLIRefusal(err error, code RefusalCode) bool {
	actual, ok := RefusalCodeOf(err)
	return ok && actual == code && !errors.Is(err, context.Canceled)
}
