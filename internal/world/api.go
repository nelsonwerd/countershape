package world

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

const requireDurableMarkerBeforeSpawn = true // MUTANT_U2_CREATE_MARKER_AFTER_SPAWN

const maxInstanceNonceBytes = 256

// materializer is an unexported fault-injection seam. Production Execute is
// hard-wired to gitobj.DefaultMaterializer so callers cannot substitute a
// forgeable filesystem/receipt implementation at the execution boundary.
type materializer interface {
	Materialize(context.Context, gitobj.BoundCandidate, string) (gitobj.MaterializationReceipt, error)
}

// Request contains capabilities, not copied identity strings. A real matching
// WorldPlan and opaque BoundCandidate are required for every execution.
type Request struct {
	Plan            domain.WorldPlan
	Candidate       gitobj.BoundCandidate
	Tools           ToolRegistry
	AllocationRoot  string
	StimulusDigest  domain.Digest
	Purpose         domain.AttemptPurpose
	InstanceNonce   string
	ScheduleOrdinal int
}

type Result struct {
	world             domain.WorldInstance
	finalized         domain.FinalizedAttempt
	states            []domain.AttemptState
	roots             Roots
	materialization   gitobj.MaterializationReceipt
	process           ProcessReceipt
	cliFixture        CLIFixtureOverlayReceipt
	hasCLIFixture     bool
	cliInvocation     CLIInvocationEvidenceReceipt
	hasCLIInvocation  bool
	httpSeed          HTTPSeedOverlayReceipt
	hasHTTPSeed       bool
	httpReadiness     HTTPReadinessReceipt
	hasHTTPReadiness  bool
	httpExchange      HTTPExchangeReceipt
	hasHTTPExchange   bool
	httpInvocation    HTTPInvocationEvidenceReceipt
	hasHTTPInvocation bool
}

func (r Result) World() domain.WorldInstance               { return r.world }
func (r Result) FinalizedAttempt() domain.FinalizedAttempt { return r.finalized }
func (r Result) StateHistory() []domain.AttemptState {
	return append([]domain.AttemptState(nil), r.states...)
}
func (r Result) Roots() Roots { return r.roots }
func (r Result) Materialization() gitobj.MaterializationReceipt {
	return cloneMaterializationReceipt(r.materialization)
}
func (r Result) Process() ProcessReceipt { return r.process.clone() }
func (r Result) CLIFixtureOverlay() (CLIFixtureOverlayReceipt, bool) {
	if !r.hasCLIFixture {
		return CLIFixtureOverlayReceipt{}, false
	}
	result := r.cliFixture
	result.canonicalBytes = append([]byte(nil), r.cliFixture.canonicalBytes...)
	result.entries = append([]CLIFixtureEntryReceipt(nil), r.cliFixture.entries...)
	return result, true
}
func (r Result) CLIInvocationEvidence() (CLIInvocationEvidenceReceipt, bool) {
	if !r.hasCLIInvocation {
		return CLIInvocationEvidenceReceipt{}, false
	}
	result := r.cliInvocation
	result.canonicalBytes = append([]byte(nil), r.cliInvocation.canonicalBytes...)
	result.expectedLogicalArgv = append([]string(nil), r.cliInvocation.expectedLogicalArgv...)
	return result, true
}

func (r Result) HTTPSeedOverlay() (HTTPSeedOverlayReceipt, bool) {
	if !r.hasHTTPSeed {
		return HTTPSeedOverlayReceipt{}, false
	}
	result := r.httpSeed
	result.canonicalBytes = append([]byte(nil), r.httpSeed.canonicalBytes...)
	result.entries = append([]HTTPSeedEntryReceipt(nil), r.httpSeed.entries...)
	return result, true
}

func (r Result) HTTPReadiness() (HTTPReadinessReceipt, bool) {
	if !r.hasHTTPReadiness {
		return HTTPReadinessReceipt{}, false
	}
	result := r.httpReadiness
	result.canonicalBytes = append([]byte(nil), r.httpReadiness.canonicalBytes...)
	return result, true
}

func (r Result) HTTPExchange() (HTTPExchangeReceipt, bool) {
	if !r.hasHTTPExchange {
		return HTTPExchangeReceipt{}, false
	}
	result := r.httpExchange
	result.canonicalBytes = append([]byte(nil), r.httpExchange.canonicalBytes...)
	result.requestWire = append([]byte(nil), r.httpExchange.requestWire...)
	result.responseWire = append([]byte(nil), r.httpExchange.responseWire...)
	return result, true
}

func (r Result) HTTPInvocationEvidence() (HTTPInvocationEvidenceReceipt, bool) {
	if !r.hasHTTPInvocation {
		return HTTPInvocationEvidenceReceipt{}, false
	}
	result := r.httpInvocation
	result.canonicalBytes = append([]byte(nil), r.httpInvocation.canonicalBytes...)
	return result, true
}

func Execute(ctx context.Context, request Request) (Result, error) {
	return executeWithMaterializer(ctx, request, gitobj.DefaultMaterializer{})
}

func executeWithMaterializer(ctx context.Context, request Request, source materializer) (Result, error) {
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	tool, err := request.Tools.resolvePlan(request.Plan)
	if err != nil {
		return Result{}, err
	}
	allocated, err := allocateAttempt(request)
	if err != nil {
		return Result{}, err
	}
	attempt := allocated.domainAttempt
	attempt, err = attempt.Advance(domain.AttemptMaterializing)
	if err != nil {
		return Result{}, err
	}
	materialization, materializeErr := source.Materialize(ctx, request.Candidate, allocated.roots.candidateParent)
	if materializeErr != nil {
		primary := materializationControl(ctx, materializeErr)
		return finalizeWithoutProcess(
			allocated, attempt, primary, materializationDiagnostic(ctx, materializeErr),
		)
	}
	if ctx.Err() != nil {
		return finalizeWithoutProcessWithMaterialization(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledAfterMaterialization, ctx.Err()),
		)
	}
	if err := validateMaterialization(ctx, request, allocated.roots, materialization); err != nil {
		return finalizeWithoutProcess(allocated, attempt, domain.ControlMaterializationError, err)
	}
	attempt, err = attempt.Advance(domain.AttemptStarting)
	if err != nil {
		return Result{}, err
	}

	if err := requireDurableMarker(allocated); err != nil {
		return finalizeWithoutProcessWithMaterialization(
			allocated, attempt, materialization, domain.ControlStartError, err,
		)
	}
	environment := buildEnvironment(request.Plan.Environment(), allocated.roots, allocated.attemptID)
	if ctx.Err() != nil {
		return finalizeWithoutProcessWithMaterialization(
			allocated, attempt, materialization, domain.ControlCancelled,
			withReceiptDiagnostic(diagnosticCancelledBeforeSpawn, ctx.Err()),
		)
	}
	physical := runProcess(ctx, processRequest{
		tool: tool, logicalArgv: request.Plan.StartArgv(), environment: environment,
		cwd: materialization.PublishedRoot, stdoutLimit: request.Plan.Budgets().StdoutBytes,
		stderrLimit:       request.Plan.Budgets().StderrBytes,
		executionBudgetMS: request.Plan.Budgets().ProbeMS,
		teardownBudgetMS:  request.Plan.Budgets().TeardownMS,
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
	})
	if !physical.started && physical.primary == "" {
		physical.primary = domain.ControlStartError
	}
	return finalizePhysical(
		allocated, attempt, materialization, tool, request.Plan.StartArgv(), environment, physical,
	)
}

func validateRequest(request Request) error {
	if !request.Plan.Digest().Valid() || len(request.Plan.CanonicalBytes()) == 0 || !request.Candidate.Valid() ||
		!request.StimulusDigest.Valid() || !request.Purpose.Valid() ||
		!validInstanceNonce(request.InstanceNonce) || request.ScheduleOrdinal < 0 {
		return refuse(CodeInvalidRequest, "request is missing executable authority or structural identity", nil)
	}
	if request.Plan.Adapter().Domain != domain.AdapterCLI ||
		request.Plan.ExecutionShape() != domain.OneCLIInvocation || len(request.Plan.SetupArgv()) != 0 {
		return refuse(CodePlanProfileRejected, "U2 executes only one-shot CLI plans without setup", nil)
	}
	for _, slot := range request.Plan.SecretSlots() {
		if slot.Presence != domain.SecretAbsent {
			return refuse(CodePlanProfileRejected, "U2 does not admit inherited secret values", nil)
		}
	}
	identity := request.Candidate.Binding().Identity()
	if identity.WorldPlanDigest != request.Plan.Digest() ||
		identity.MaterializationPolicyDigest != request.Plan.MaterializationPolicyDigest() {
		return refuse(CodeInvalidRequest, "bound candidate does not match the supplied plan", nil)
	}
	return nil
}

func validInstanceNonce(nonce string) bool {
	return validBoundedText(nonce, maxInstanceNonceBytes)
}

func validateMaterialization(ctx context.Context, request Request, roots Roots, receipt gitobj.MaterializationReceipt) error {
	expected := filepath.Join(roots.candidateParent, "candidate")
	if !receipt.Valid() || receipt.PolicyDigest != request.Plan.MaterializationPolicyDigest() ||
		receipt.PortableTreeDigest != request.Candidate.PortableTreeDigest() || receipt.PublishedRoot != expected {
		return withReceiptDiagnostic(
			diagnosticMaterializationReceiptInvalid,
			refuse(CodeInvalidAllocation, "materialization receipt does not bind the fresh destination and candidate", nil),
		)
	}
	info, err := os.Lstat(receipt.PublishedRoot)
	if err != nil || !info.IsDir() {
		return withReceiptDiagnostic(
			diagnosticMaterializationRootUnavailable,
			refuse(CodeInvalidAllocation, "published candidate root is unavailable", err),
		)
	}
	if _, err := os.Lstat(filepath.Join(receipt.PublishedRoot, ".git")); err == nil || !errors.Is(err, os.ErrNotExist) {
		return withReceiptDiagnostic(
			diagnosticMaterializationGitMetadataPresent,
			refuse(CodeInvalidAllocation, "published candidate unexpectedly contains .git", err),
		)
	}
	if err := gitobj.ValidatePublishedMaterialization(ctx, request.Candidate, receipt); err != nil {
		return withReceiptDiagnostic(
			diagnosticMaterializationRevalidationFailed,
			refuse(CodeInvalidAllocation, "published candidate failed bound-source revalidation", err),
		)
	}
	return nil
}

func requireDurableMarker(allocated allocatedAttempt) error {
	if !requireDurableMarkerBeforeSpawn {
		return nil
	}
	data, err := os.ReadFile(allocated.roots.marker)
	if err != nil || string(data) != string(allocated.markerBytes) {
		return withReceiptDiagnostic(
			diagnosticMarkerContentInvalid,
			refuse(CodeMarkerWriteFailed, "attempt marker is missing or changed before spawn", err),
		)
	}
	info, err := os.Lstat(allocated.roots.marker)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return withReceiptDiagnostic(
			diagnosticMarkerModeInvalid,
			refuse(CodeMarkerWriteFailed, "attempt marker mode changed before spawn", err),
		)
	}
	return nil
}

func materializationDiagnostic(ctx context.Context, err error) error {
	if code, ok := gitobj.RefusalCodeOf(err); ok {
		return withReceiptDiagnostic(string(code), err)
	}
	if contextErr := ctx.Err(); contextErr != nil && errors.Is(err, contextErr) {
		return withReceiptDiagnostic(diagnosticCancelledDuringMaterialization, err)
	}
	return withReceiptDiagnostic(diagnosticMaterializationFailed, err)
}

func materializationControl(ctx context.Context, err error) domain.ControlReason {
	if code, ok := gitobj.RefusalCodeOf(err); ok {
		switch code {
		case gitobj.CodeMissingObject:
			return domain.ControlMissingObject
		case gitobj.CodeUnsupportedMode:
			return domain.ControlUnsupportedGitMode
		case gitobj.CodeBudgetExceeded:
			return domain.ControlBudgetExhausted
		}
	}
	if contextErr := ctx.Err(); contextErr != nil && errors.Is(err, contextErr) {
		return domain.ControlCancelled
	}
	return domain.ControlMaterializationError
}

func finalizeWithoutProcess(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	primary domain.ControlReason,
	diagnostic error,
) (Result, error) {
	return finalizeWithoutProcessWithMaterialization(
		allocated, attempt, gitobj.MaterializationReceipt{}, primary, diagnostic,
	)
}

func finalizeWithoutProcessWithMaterialization(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	materialization gitobj.MaterializationReceipt,
	primary domain.ControlReason,
	diagnostic error,
) (Result, error) {
	return finalizeWithoutProcessWithLineage(
		allocated, attempt, materialization, primary, diagnostic, processCLILineage{},
	)
}

func finalizeWithoutProcessWithLineage(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	materialization gitobj.MaterializationReceipt,
	primary domain.ControlReason,
	diagnostic error,
	cli processCLILineage,
) (Result, error) {
	return finalizeWithoutProcessWithLineageAndTool(
		allocated, attempt, materialization, primary, diagnostic, resolvedTool{}, cli,
	)
}

func finalizeWithoutProcessWithLineageAndTool(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	materialization gitobj.MaterializationReceipt,
	primary domain.ControlReason,
	diagnostic error,
	tool resolvedTool,
	cli processCLILineage,
) (Result, error) {
	attempt, err := attempt.Fail(primary)
	if err != nil {
		return Result{}, err
	}
	attempt, err = attempt.Advance(domain.AttemptFinalized)
	if err != nil {
		return Result{}, err
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		return Result{}, err
	}
	process, err := buildProcessReceipt(receiptInput{
		allocated: allocated, materialization: materialization, attempt: attempt,
		tool: tool, primary: primary, diagnostic: diagnostic, cli: cli,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		world: allocated.world, finalized: finalized, states: attempt.StateHistory(), roots: allocated.roots,
		materialization: cloneMaterializationReceipt(materialization), process: process,
	}, nil
}
