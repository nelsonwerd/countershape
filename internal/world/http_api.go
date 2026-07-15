package world

import (
	"context"
	"sort"
	"strconv"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

const (
	httpStimulusDigestEnvironment   = "COUNTERSHAPE_HTTP_STIMULUS_DIGEST"
	diagnosticHTTPSeedOverlayFailed = "HTTP_SEED_OVERLAY_FAILED"
)

// HTTPRequest carries one opaque adapter-owned binding plus physical
// capabilities. It has no port, URL, proxy, request bytes, parser callback,
// shared root, redirect policy, retry policy, or independently pairable plan.
type HTTPRequest struct {
	Binding         httpmodel.HTTPExecutionBinding
	Candidate       gitobj.BoundCandidate
	Tools           ToolRegistry
	AllocationRoot  string
	Purpose         domain.AttemptPurpose
	InstanceNonce   string
	ScheduleOrdinal int
}

func ExecuteHTTP(ctx context.Context, request HTTPRequest) (Result, error) {
	return executeHTTPWithMaterializer(ctx, request, gitobj.DefaultMaterializer{})
}

func executeHTTPWithMaterializer(ctx context.Context, request HTTPRequest, source materializer) (Result, error) {
	derived, err := deriveHTTPRequest(request)
	if err != nil {
		return Result{}, err
	}
	tool, err := derived.Tools.resolvePlan(derived.Plan)
	if err != nil {
		return Result{}, err
	}
	// MUTATION_ANCHOR: http-product-execution-always-allocates-one-fresh-private-attempt-root
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
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, gitobj.MaterializationReceipt{}, materializationControl(ctx, materializeErr),
			materializationDiagnostic(ctx, materializeErr), tool, processCLILineage{},
		)
	}
	if ctx.Err() != nil {
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledAfterMaterialization, ctx.Err()), tool, processCLILineage{},
		)
	}
	if err := validateMaterialization(ctx, derived, allocated.roots, materialization); err != nil {
		return finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, gitobj.MaterializationReceipt{}, domain.ControlMaterializationError,
			err, tool, processCLILineage{},
		)
	}
	seedReceipt, err := materializeHTTPSeeds(request.Binding, allocated.roots.fixture)
	if err != nil {
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlMaterializationError,
			withReceiptDiagnostic(diagnosticHTTPSeedOverlayFailed, err), tool, processCLILineage{},
		)
		return attachHTTPArtifacts(result, HTTPSeedOverlayReceipt{}, HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, HTTPInvocationEvidenceReceipt{}), finalizeErr
	}
	// MUTATION_ANCHOR: http-seeds-and-candidate-share-world-materialization-budgets
	if err := validateCombinedHTTPMaterializationBudgets(derived.Plan.Budgets(), materialization, seedReceipt); err != nil {
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlMaterializationError,
			withReceiptDiagnostic("HTTP_COMBINED_MATERIALIZATION_BUDGET_EXCEEDED", err), tool, processCLILineage{},
		)
		return attachHTTPArtifacts(result, seedReceipt, HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, HTTPInvocationEvidenceReceipt{}), finalizeErr
	}
	environment, err := buildHTTPEnvironment(
		derived.Plan.Environment(), allocated.roots, allocated.attemptID,
		request.Binding.StimulusDigest(), request.ScheduleOrdinal, derived.Plan.Budgets().CandidateCount,
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
			allocated, attempt, materialization, domain.ControlStartError, err, tool, processCLILineage{},
		)
		return attachHTTPArtifacts(result, seedReceipt, HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, HTTPInvocationEvidenceReceipt{}), finalizeErr
	}
	if ctx.Err() != nil {
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledBeforeSpawn, ctx.Err()), tool, processCLILineage{},
		)
		return attachHTTPArtifacts(result, seedReceipt, HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, HTTPInvocationEvidenceReceipt{}), finalizeErr
	}

	policy := request.Binding.CapturePolicy()
	responseWireLimit := policy.OwnerResponseReadLimit()
	service := runLiveHTTPService(ctx, httpServiceRequest{
		tool: tool, binding: request.Binding, logicalArgv: request.Binding.LogicalArgv(), environment: environment,
		cwd: materialization.PublishedRoot, responseLimit: responseWireLimit,
		stdoutLimit: derived.Plan.Budgets().StdoutBytes, stderrLimit: derived.Plan.Budgets().StderrBytes,
		readinessBudgetMS: derived.Plan.Budgets().ReadinessMS, probeBudgetMS: derived.Plan.Budgets().ProbeMS,
		teardownBudgetMS: derived.Plan.Budgets().TeardownMS, markerBeforeSpawn: true,
		// MUTATION_ANCHOR: http-readiness-advances-only-after-inherited-pipe-byte-and-eof
		onReadinessAccepted: func() error {
			var transitionErr error
			attempt, transitionErr = attempt.Advance(domain.AttemptReady)
			if transitionErr != nil {
				return transitionErr
			}
			attempt, transitionErr = attempt.Advance(domain.AttemptProbing)
			return transitionErr
		},
		onResponseCaptured: func() error {
			var transitionErr error
			attempt, transitionErr = attempt.Advance(domain.AttemptCapturing)
			return transitionErr
		},
	})
	if !service.process.physicalExecutionEntered {
		primary := service.process.primary
		if primary == "" {
			primary = domain.ControlStartError
		}
		diagnostic := service.process.diagnosticCode
		if diagnostic == "" {
			diagnostic = "HTTP_PRE_PROCESS_START_FAILURE"
		}
		result, finalizeErr := finalizeWithoutProcessWithLineageAndTool(
			allocated, attempt, materialization, primary, withReceiptDiagnostic(diagnostic, nil), tool, processCLILineage{},
		)
		readiness, exchange, receiptErr := buildHTTPPhysicalReceipts(allocated, request.Binding, service)
		if receiptErr != nil {
			return Result{}, receiptErr
		}
		return attachHTTPArtifacts(result, seedReceipt, readiness, exchange, HTTPInvocationEvidenceReceipt{}), finalizeErr
	}
	if !service.process.spawnAttempted {
		return Result{}, refuse(CodeHTTPExecutionRejected, "physical HTTP entry lacks a spawn attempt", nil)
	}

	readinessReceipt, exchangeReceipt, err := buildHTTPPhysicalReceipts(allocated, request.Binding, service)
	if err != nil {
		return Result{}, err
	}
	invocationReceipt, err := inspectHTTPInvocationEvidence(
		allocated.world.Digest(), allocated.markerDigest, request.Binding.Digest(), request.Binding.StimulusDigest(), allocated.roots.evidence,
		allocated.attemptID, service.exchange.requestWire,
	)
	if err != nil {
		return Result{}, err
	}
	if service.exchange.responseParsed && service.process.primary == "" && !invocationReceipt.Validated() {
		// A complete HTTP response is not enough for the reference fixture's
		// physical-freshness claim. The control remains outside the projection.
		service.process.primary = domain.ControlProbeTransportError
		service.process.diagnosticCode = "HTTP_INVOCATION_EVIDENCE_INVALID"
	}
	// MUTATION_ANCHOR: successful-http-response-still-routes-through-clean-teardown-eligibility
	result, err := finalizePhysical(
		allocated, attempt, materialization, tool, request.Binding.LogicalArgv(), service.environment, service.process,
	)
	if err != nil {
		return Result{}, err
	}
	return attachHTTPArtifacts(result, seedReceipt, readinessReceipt, exchangeReceipt, invocationReceipt), nil
}

func deriveHTTPRequest(request HTTPRequest) (Request, error) {
	if !request.Binding.Valid() || !request.Candidate.Valid() || !request.Purpose.Valid() ||
		!validInstanceNonce(request.InstanceNonce) || request.ScheduleOrdinal < 0 {
		return Request{}, refuse(CodeHTTPExecutionRejected, "opaque HTTP execution authority is incomplete", nil)
	}
	plan := request.Binding.Plan()
	if plan.Adapter().Domain != domain.AdapterHTTP || plan.ExecutionShape() != domain.OneLoopbackHTTPRequest || len(plan.SetupArgv()) != 0 ||
		!request.Binding.StartSpec().Valid() || !request.Binding.CapturePolicy().Valid() || !request.Binding.Readiness().Valid() ||
		request.Binding.Readiness().Protocol() != httpmodel.ReadinessProtocolV1 ||
		request.Binding.Readiness().SuccessByte() != httpmodel.ReadinessSuccessByte {
		return Request{}, refuse(CodeHTTPExecutionRejected, "binding is outside the closed U4 HTTP profile", nil)
	}
	for _, slot := range plan.SecretSlots() {
		if slot.Presence != domain.SecretAbsent {
			return Request{}, refuse(CodeHTTPExecutionRejected, "HTTP v1 does not admit inherited secret values", nil)
		}
	}
	identity := request.Candidate.Binding().Identity()
	if identity.WorldPlanDigest != plan.Digest() || identity.MaterializationPolicyDigest != plan.MaterializationPolicyDigest() {
		return Request{}, refuse(CodeHTTPExecutionRejected, "bound candidate does not match the HTTP plan", nil)
	}
	candidateCount := plan.Budgets().CandidateCount
	repeatCount := plan.RepeatSchedule().DiscoveryRepeats
	if request.Purpose == domain.AttemptFinalSweep || request.Purpose == domain.AttemptConfirmation || request.Purpose == domain.AttemptConformance {
		repeatCount = plan.RepeatSchedule().ConfirmationRepeats
	}
	if candidateCount < 2 || request.ScheduleOrdinal >= candidateCount*repeatCount {
		return Request{}, refuse(CodeHTTPExecutionRejected, "schedule ordinal is outside the plan-bound matrix", nil)
	}
	return Request{
		Plan: plan, Candidate: request.Candidate, Tools: request.Tools, AllocationRoot: request.AllocationRoot,
		StimulusDigest: request.Binding.StimulusDigest(), Purpose: request.Purpose,
		InstanceNonce: request.InstanceNonce, ScheduleOrdinal: request.ScheduleOrdinal,
	}, nil
}

func buildHTTPEnvironment(
	planEntries []domain.EnvironmentEntry,
	roots Roots,
	attemptID string,
	stimulusDigest domain.Digest,
	scheduleOrdinal, candidateCount int,
) ([]string, error) {
	if !stimulusDigest.Valid() || candidateCount < 2 || scheduleOrdinal < 0 {
		return nil, refuse(CodeHTTPEnvironmentRejected, "HTTP environment inputs are invalid", nil)
	}
	values := make(map[string]string, len(planEntries)+13)
	for _, entry := range planEntries {
		if isHTTPDescriptorEnvironmentName(entry.Name) {
			return nil, refuse(CodeHTTPEnvironmentRejected, "plan collides with runner-owned HTTP descriptor environment", nil)
		}
		values[entry.Name] = entry.Value
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
	values[httpStimulusDigestEnvironment] = stimulusDigest.String()
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

func buildHTTPPhysicalReceipts(
	allocated allocatedAttempt,
	binding httpmodel.HTTPExecutionBinding,
	service liveHTTPServiceResult,
) (HTTPReadinessReceipt, HTTPExchangeReceipt, error) {
	if service.readiness.endpoint == "" {
		if service.process.physicalExecutionEntered {
			return HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, refuse(CodeHTTPExecutionRejected, "physical HTTP entry has no owned endpoint receipt", nil)
		}
		return HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, nil
	}
	if !service.readiness.accepted && service.readiness.diagnosticCode == "" {
		service.readiness.diagnosticCode = firstDiagnostic(service.process.diagnosticCode, "HTTP_READINESS_NOT_REACHED")
	}
	readiness, readinessErr := newHTTPReadinessReceipt(
		allocated.world.Digest(), allocated.markerDigest, binding.Digest(), service.readiness,
	)
	if readinessErr != nil {
		return HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, readinessErr
	}
	if len(service.exchange.requestWire) == 0 {
		if service.process.physicalExecutionEntered {
			return HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, refuse(CodeHTTPExecutionRejected, "physical HTTP entry has no encoded request receipt", nil)
		}
		return readiness, HTTPExchangeReceipt{}, nil
	}
	if !service.exchange.responseParsed && service.exchange.diagnosticCode == "" && service.process.primary != "" {
		service.exchange.diagnosticCode = service.process.diagnosticCode
	}
	exchange, exchangeErr := newHTTPExchangeReceipt(
		allocated.world.Digest(), allocated.markerDigest, binding.Digest(), service.readiness.endpoint, service.exchange,
	)
	if exchangeErr != nil {
		return HTTPReadinessReceipt{}, HTTPExchangeReceipt{}, exchangeErr
	}
	return readiness, exchange, nil
}

func attachHTTPArtifacts(
	result Result,
	seed HTTPSeedOverlayReceipt,
	readiness HTTPReadinessReceipt,
	exchange HTTPExchangeReceipt,
	invocation HTTPInvocationEvidenceReceipt,
) Result {
	if seed.Valid() {
		result.httpSeed, result.hasHTTPSeed = seed, true
	}
	if readiness.Valid() {
		result.httpReadiness, result.hasHTTPReadiness = readiness, true
	}
	if exchange.Valid() {
		result.httpExchange, result.hasHTTPExchange = exchange, true
	}
	if invocation.Valid() {
		result.httpInvocation, result.hasHTTPInvocation = invocation, true
	}
	return result
}

func validateCombinedHTTPMaterializationBudgets(
	budgets domain.Budgets,
	materialization gitobj.MaterializationReceipt,
	seeds HTTPSeedOverlayReceipt,
) error {
	if !materialization.Valid() || !seeds.Valid() {
		return refuse(CodeHTTPSeedRejected, "candidate or seed receipt is invalid", nil)
	}
	entryCount := len(materialization.Entries) + len(seeds.entries)
	var totalBytes int64
	for _, entry := range materialization.Entries {
		if entry.Bytes > budgets.SingleBlobBytes {
			return refuse(CodeHTTPSeedRejected, "candidate entry exceeds single-blob budget", nil)
		}
		totalBytes += entry.Bytes
	}
	for _, entry := range seeds.entries {
		if entry.bytes > budgets.SingleBlobBytes {
			return refuse(CodeHTTPSeedRejected, "seed entry exceeds single-blob budget", nil)
		}
		totalBytes += entry.bytes
	}
	if entryCount > budgets.MaterializedEntryCount || totalBytes > budgets.MaterializedBytesPerWorld {
		return refuse(CodeHTTPSeedRejected, "candidate plus seed overlay exceeds world materialization budget", nil)
	}
	return nil
}
