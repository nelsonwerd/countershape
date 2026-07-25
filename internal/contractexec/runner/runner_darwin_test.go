//go:build darwin && arm64 && cgo

package runner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
	"github.com/nelsonwerd/countershape/internal/store"
	clitest "github.com/nelsonwerd/countershape/testkit/contractexec/cli"
)

func TestMain(m *testing.M) { os.Exit(clitest.RunMain(m)) }

func TestConcurrentAdmissionProducesExactlyOneStart(t *testing.T) {
	fixture := clitest.NewAdmissionFixture(t)
	type outcome struct {
		started bool
		clean   bool
		err     error
	}
	ready := make(chan struct{})
	results := make(chan outcome, 2)
	for range 2 {
		prepared, binding := preparedAdmissionProcess(t)
		go func() {
			<-ready
			ctx := context.Background()
			owner, err := store.AcquireContractRunOwner(ctx, fixture.Target, fixture.Epoch)
			if err != nil {
				results <- outcome{err: err}
				return
			}
			observation, running, err := consumeAndStart(ctx, ctx, owner, prepared, binding)
			if err != nil || running == nil {
				results <- outcome{err: err}
				return
			}
			closed := running.Close()
			results <- outcome{
				started: observation.PID > 0 && observation.Binding == prepared.BindingDigest(),
				clean: closed.Primary == "" && closed.ChildWaited && closed.FinalProbeClean &&
					!closed.TeardownError && !closed.OrphanRisk,
				err: err,
			}
		}()
	}
	close(ready)

	successes := 0
	refusals := 0
	for range 2 {
		result := <-results
		switch {
		case result.err == nil && result.started && result.clean:
			successes++
		case result.err != nil && !result.started:
			refusals++
		default:
			t.Fatalf("concurrent result escaped the exact winner/refusal split: %+v", result)
		}
	}
	if successes != 1 || refusals != 1 {
		t.Fatalf("concurrent admission produced successes=%d refusals=%d, want exactly one each", successes, refusals)
	}
}

func TestRunPermitConsumptionIsSingleUseAndAdjacentToStart(t *testing.T) {
	assertConsumeImmediatelyPrecedesStart(t)
	assertTargetReopenImmediatelyPrecedesPermitPath(t)
	fixture := clitest.NewAdmissionFixture(t)
	ctx := context.Background()
	owner, err := store.AcquireContractRunOwner(ctx, fixture.Target, fixture.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	copied := owner
	prepared, binding := preparedAdmissionProcess(t)
	observation, running, err := consumeAndStart(ctx, ctx, owner, prepared, binding)
	if err != nil || running == nil || observation.PID < 1 {
		t.Fatalf("first exact one-shot operation did not start: observation=%+v err=%v", observation, err)
	}
	closed := running.Close()
	if closed.Primary != "" || !closed.ChildWaited || !closed.FinalProbeClean {
		t.Fatalf("first exact one-shot operation did not close cleanly: %+v", closed)
	}
	secondPrepared, secondBinding := preparedAdmissionProcess(t)
	secondObservation, secondRunning, err := consumeAndStart(ctx, ctx, copied, secondPrepared, secondBinding)
	if err == nil || !IsCode(err, CodeAdmissionRefused) || secondRunning != nil ||
		secondObservation != (processmechanics.SpawnObservation{}) {
		t.Fatalf("copied owner multiplied start authority: observation=%+v running=%v err=%v", secondObservation, secondRunning, err)
	}
}

func TestStartErrorClosesDurableRunAndClassificationWithoutChild(t *testing.T) {
	fixture := clitest.NewReferenceTarget(t)
	preparation, err := prepareCLIExecution(fixture.Target)
	if err != nil {
		t.Fatalf("prepare start-error execution: %v (cause: %v)", err, errors.Unwrap(err))
	}
	input := cliExecutionInput{target: fixture.Target, cliExecutionPreparation: preparation}
	t.Cleanup(func() { _ = input.closeProbe() })
	ctx := context.Background()
	epoch, err := hostepoch.Measure(ctx)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.AcquireContractRunOwner(ctx, fixture.Target.TargetRecord(), epoch)
	if err != nil {
		t.Fatal(err)
	}
	candidateRoot := input.target.CandidateRoot()
	displacedRoot := candidateRoot + ".start-error-test"
	if err := os.Rename(candidateRoot, displacedRoot); err != nil {
		t.Fatal(err)
	}
	restored := false
	t.Cleanup(func() {
		if !restored {
			_ = os.Rename(displacedRoot, candidateRoot)
		}
	})
	observation, running, startErr := consumeAndStart(ctx, ctx, owner, input.prepared, input.binding)
	if running != nil {
		t.Cleanup(func() {
			running.AbortSpawnObservationPersistence()
			_ = running.Close()
		})
	}
	restoreErr := os.Rename(displacedRoot, candidateRoot)
	restored = restoreErr == nil
	if restoreErr != nil {
		t.Fatalf("restore exact candidate root after start refusal: %v", restoreErr)
	}
	var failure *processmechanics.StartError
	if startErr == nil || !errors.As(startErr, &failure) || failure.Code() != "SPAWN_FAILED" ||
		running != nil || observation != (processmechanics.SpawnObservation{}) {
		t.Fatalf("missing cwd did not produce the exact OS start refusal: observation=%+v running=%v err=%v", observation, running, startErr)
	}
	startResult := failure.Result()
	if !startResult.PhysicalExecutionEntered || !startResult.SpawnAttempted || startResult.Started ||
		startResult.PID != 0 || startResult.ProcessGroupID != 0 || startResult.ProcessGroupOwned ||
		startResult.Binding != input.prepared.BindingDigest() {
		t.Fatalf("OS start refusal facts are incomplete: %+v", startResult)
	}
	closureContext, cancelClosure := terminalClosureContext(ctx, input)
	defer cancelClosure()
	record, err := closeStartError(closureContext, owner, input, startErr)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("start refusal did not close a durable ineligible classification: valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	run, err := store.OpenFinalizedRunRecord(ctx, fixture.Target.TargetRecord(), fixture.Target.Model())
	if err != nil || !run.Valid() {
		t.Fatalf("start-refusal finalized run did not reopen: valid=%t err=%v", run.Valid(), err)
	}
	witness := run.Model().Witness()
	errorCode, hasErrorCode := witness.SpawnObservation().ErrorCode()
	primary, hasPrimary := witness.ProcessClosure().Primary()
	if witness.SpawnObservation().State() != contractmodel.SpawnStartError || !hasErrorCode || errorCode != "OS_START_ERROR" ||
		witness.ProcessClosure().State() != contractmodel.ProcessControlled || !hasPrimary || primary != domain.ControlStartError ||
		witness.Observation().State() != contractmodel.CaptureNone || witness.StandaloneScope().State() != contractmodel.ScopePartial ||
		witness.Disposition() != contractmodel.DispositionIneligibleControlStandalone {
		t.Fatalf("start-refusal witness axes differ: spawn=%s code=%q process=%s primary=%s capture=%s scope=%s disposition=%s",
			witness.SpawnObservation().State(), errorCode, witness.ProcessClosure().State(), primary,
			witness.Observation().State(), witness.StandaloneScope().State(), witness.Disposition())
	}
	checks := witness.StandaloneScope().Checks()
	if len(checks) != 5 {
		t.Fatalf("start-refusal scope has %d checks, want five missing domains", len(checks))
	}
	for _, check := range checks {
		if check.State() != contractmodel.ScopeCheckMissing {
			t.Fatalf("start-refusal scope domain %s = %s, want missing", check.Domain(), check.State())
		}
	}
	recovered, err := ResumeCLIClassification(ctx, fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("start-refusal classification recovery diverged: valid=%t err=%v", recovered.Valid(), err)
	}
}

func assertTargetReopenImmediatelyPrecedesPermitPath(t *testing.T) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runner test source location unavailable")
	}
	sourcePath := filepath.Join(filepath.Dir(filename), "runner_darwin.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body *ast.BlockStmt
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "executeCLI" {
			body = function.Body
			break
		}
	}
	if body == nil {
		t.Fatal("executeCLI declaration is absent")
	}
	exactCall := func(expression ast.Expr, callee string, arguments ...string) bool {
		call, ok := expression.(*ast.CallExpr)
		if !ok || expressionName(call.Fun) != callee || len(call.Args) != len(arguments) {
			return false
		}
		for index, argument := range call.Args {
			if expressionName(argument) != arguments[index] {
				return false
			}
		}
		return true
	}
	exactCallAssignment := func(statement ast.Stmt, assignmentToken token.Token, left []string, callee string, arguments ...string) bool {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != assignmentToken || len(assignment.Lhs) != len(left) || len(assignment.Rhs) != 1 {
			return false
		}
		for index, expression := range assignment.Lhs {
			if expressionName(expression) != left[index] {
				return false
			}
		}
		return exactCall(assignment.Rhs[0], callee, arguments...)
	}
	refusalReturn := func(statement ast.Stmt, code string) bool {
		returned, ok := statement.(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 2 {
			return false
		}
		emptyRecord, ok := returned.Results[0].(*ast.CompositeLit)
		refusal, refusalOK := returned.Results[1].(*ast.CallExpr)
		return ok && expressionName(emptyRecord.Type) == "store.ContractExecutionRecord" && len(emptyRecord.Elts) == 0 &&
			refusalOK && expressionName(refusal.Fun) == "refuse" && len(refusal.Args) == 3 &&
			expressionName(refusal.Args[0]) == code
	}
	errGuard := func(statement ast.Stmt) bool {
		guard, ok := statement.(*ast.IfStmt)
		if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
			return false
		}
		condition, ok := guard.Cond.(*ast.BinaryExpr)
		return ok && condition.Op == token.NEQ && expressionName(condition.X) == "err" &&
			expressionName(condition.Y) == "nil" && refusalReturn(guard.Body.List[0], "CodeTargetChanged")
	}
	freshnessGuard := func(statement ast.Stmt) bool {
		guard, ok := statement.(*ast.IfStmt)
		if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 2 {
			return false
		}
		negated, ok := guard.Cond.(*ast.UnaryExpr)
		return ok && negated.Op == token.NOT && exactCall(negated.X, "samePreparedTarget", "input.cliExecutionPreparation", "spawnTarget") &&
			exactCallAssignment(guard.Body.List[0], token.ASSIGN, []string{"_"}, "spawnTarget.Close") &&
			refusalReturn(guard.Body.List[1], "CodeTargetChanged")
	}
	targetReplacement := func(statement ast.Stmt) bool {
		assignment, ok := statement.(*ast.AssignStmt)
		return ok && assignment.Tok == token.ASSIGN && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 &&
			expressionName(assignment.Lhs[0]) == "input.target" && expressionName(assignment.Rhs[0]) == "spawnTarget"
	}
	targetCleanup := func(statement ast.Stmt) bool {
		deferred, ok := statement.(*ast.DeferStmt)
		if !ok || len(deferred.Call.Args) != 0 {
			return false
		}
		closure, ok := deferred.Call.Fun.(*ast.FuncLit)
		return ok && closure.Type.Params.NumFields() == 0 && closure.Type.Results == nil && len(closure.Body.List) == 1 &&
			exactCallAssignment(closure.Body.List[0], token.ASSIGN, []string{"_"}, "spawnTarget.Close")
	}

	acquiredIndex := -1
	for index, statement := range body.List {
		if isSelectorCall(statement, "store", "AcquireContractRunOwner") {
			acquiredIndex = index
			break
		}
	}
	if acquiredIndex < 0 {
		t.Fatal("executeCLI lacks contract-run owner acquisition")
	}
	found := false
	for index := acquiredIndex + 1; index+5 < len(body.List); index++ {
		window := body.List[index : index+6]
		if !exactCallAssignment(window[0], token.DEFINE, []string{"spawnTarget", "err"},
			"contractexec.ReopenOfficialTarget", "closureContext", "input.target") ||
			!errGuard(window[1]) || !freshnessGuard(window[2]) || !targetReplacement(window[3]) ||
			!targetCleanup(window[4]) ||
			!exactCallAssignment(window[5], token.DEFINE, []string{"physicalObservation", "running", "startErr"},
				"consumeAndStart", "ctx", "closureContext", "owner", "input.prepared", "input.binding") {
			continue
		}
		if found {
			t.Fatal("executeCLI contains more than one exact spawn-adjacent target-reopen window")
		}
		found = true
	}
	if !found {
		t.Fatal("executeCLI lacks the exact acquired/reopened/guarded/replaced/cleanup/consume spawn window")
	}
}

func TestPreparedCLIEnvironmentUsesFreshAttemptEvidenceRootAndShortCanaries(t *testing.T) {
	t.Setenv(parentSecretSentinel, "must-not-cross")
	t.Setenv(parentCredentialSentinel, "must-not-cross")
	fixture := clitest.NewReferenceTarget(t)
	preparation, err := prepareCLIExecution(fixture.Target)
	if err != nil {
		t.Fatal(err)
	}
	input := cliExecutionInput{target: fixture.Target, cliExecutionPreparation: preparation}
	probeRoot := input.scope.root
	shortRoot := input.scope.socketRoot
	t.Cleanup(func() { _ = input.closeProbe() })
	assertPrivateEvidenceCapacityAndFrame(t, input)

	environment := make(map[string]string, len(input.environment))
	for _, entry := range input.environment {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			t.Fatalf("prepared environment contains malformed entry %q", entry)
		}
		environment[name] = value
	}
	roots := fixture.Target.Roots()
	forbidden := "attempt:" + strings.TrimPrefix(fixture.Target.Model().Input().Attempt.ArtifactDigest.String(), "sha256:")
	if !validCLIRuntimeAttemptID(input.attemptID) || environment[attemptIDEnvironment] != input.attemptID ||
		input.attemptID == forbidden || environment[evidenceRootEnvironment] != roots.EvidenceRoot() ||
		input.evidenceRoot != roots.EvidenceRoot() || input.sentinelInherited ||
		environment[parentSecretSentinel] != "" || environment[parentCredentialSentinel] != "" {
		t.Fatalf("prepared environment did not preserve fresh/private/empty-parent authority: attempt=%q evidence=%q env=%v",
			input.attemptID, input.evidenceRoot, environment)
	}
	if len([]byte(filepath.Join(probeRoot, "import.sock"))) <= maxDarwinUnixSocketPathBytes {
		t.Fatalf("fixture does not exercise the long-root regression: root=%q", probeRoot)
	}
	for _, socketPath := range []string{input.scope.importCanary.path, input.scope.serviceCanary.path} {
		if len([]byte(socketPath)) > maxDarwinUnixSocketPathBytes {
			t.Fatalf("short canary path exceeds Darwin sockaddr_un capacity: %q", socketPath)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Dir(socketPath))
		if err != nil || resolved != probeRoot {
			t.Fatalf("short canary alias escaped the attempt-private probe root: path=%q resolved=%q err=%v", socketPath, resolved, err)
		}
	}
	if err := input.closeProbe(); err != nil {
		t.Fatal(err)
	}
	for _, retired := range []string{probeRoot, shortRoot} {
		if _, err := os.Lstat(retired); !os.IsNotExist(err) {
			t.Fatalf("scope probe root was not retired: path=%q err=%v", retired, err)
		}
	}
	type invocationWire struct {
		AttemptID   string   `json:"attempt_id"`
		LogicalArgv []string `json:"logical_argv"`
	}
	exactReceipt, err := json.Marshal(invocationWire{AttemptID: input.attemptID, LogicalArgv: input.logicalArgv})
	if err != nil {
		t.Fatal(err)
	}
	wrongAttempt := "attempt:" + strings.Repeat("0", 64)
	if wrongAttempt == input.attemptID {
		wrongAttempt = "attempt:" + strings.Repeat("1", 64)
	}
	attemptMismatch, _ := json.Marshal(invocationWire{AttemptID: wrongAttempt, LogicalArgv: input.logicalArgv})
	wrongArgv := append([]string(nil), input.logicalArgv...)
	wrongArgv[len(wrongArgv)-1] += ".mismatch"
	argvMismatch, _ := json.Marshal(invocationWire{AttemptID: input.attemptID, LogicalArgv: wrongArgv})
	invocationPath := filepath.Join(input.evidenceRoot, cliInvocationFilename)
	for _, parserCase := range []struct {
		name, status string
		body         []byte
		mode         os.FileMode
		symlink      bool
		absent       bool
	}{
		{name: "exact", status: invocationEvidenceValidated, body: exactReceipt, mode: 0o600},
		{name: "noncanonical", status: invocationEvidenceSchemaMismatch, body: append(append([]byte(nil), exactReceipt...), '\n'), mode: 0o600},
		{name: "schema", status: invocationEvidenceSchemaMismatch, body: []byte(`{"attempt_id":"missing-argv"}`), mode: 0o600},
		{name: "attempt", status: invocationEvidenceAttemptMismatch, body: attemptMismatch, mode: 0o600},
		{name: "argv", status: invocationEvidenceArgvMismatch, body: argvMismatch, mode: 0o600},
		{name: "mode", status: invocationEvidenceFileFacts, body: exactReceipt, mode: 0o644},
		{name: "oversized", status: invocationEvidenceTooLarge, body: make([]byte, maxCLIInvocationBytes+1), mode: 0o600},
		{name: "symlink", status: invocationEvidenceFileFacts, symlink: true},
		{name: "absent", status: invocationEvidenceAbsent, absent: true},
	} {
		if !parserCase.absent {
			if parserCase.symlink {
				if err := os.Symlink(input.target.Roots().MarkerPath(), invocationPath); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(invocationPath, parserCase.body, parserCase.mode); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(invocationPath, parserCase.mode); err != nil {
					t.Fatal(err)
				}
			}
		}
		observed, err := inspectAndRetireCLIInvocationEvidence(
			context.Background(), input.evidenceRoot, input.target.Roots().MarkerPath(), input.attemptID, input.logicalArgv,
			input.evidenceAuthority,
		)
		if err != nil || observed.Status != parserCase.status {
			t.Fatalf("invocation parser case %s status=%s err=%v, want %s", parserCase.name, observed.Status, err, parserCase.status)
		}
	}
	failedProbe, err := newScopeProbe(roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := failedProbe.importCanary.listener.Close(); err != nil {
		t.Fatal(err)
	}
	measurements, finishErr := failedProbe.finish(context.Background())
	if finishErr == nil || !measurements.Ambiguous {
		t.Fatalf("unexpected canary accept failure became a clean zero observation: measurements=%+v err=%v", measurements, finishErr)
	}
	tamperedProbe, err := newScopeProbe(roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(tamperedProbe.serviceCanary.path); err != nil {
		t.Fatal(err)
	}
	measurements, finishErr = tamperedProbe.finish(context.Background())
	if finishErr == nil || !measurements.Ambiguous {
		t.Fatalf("changed canary socket identity became a clean zero observation: measurements=%+v err=%v", measurements, finishErr)
	}
	assertBoundedForeignEvidenceAndScopeResidue(t, input)
	fixturePath := filepath.Join(roots.FixtureRoot(), "config.json")
	for _, hostile := range []struct {
		name   string
		create func() error
	}{
		{name: "symlink", create: func() error { return os.Symlink("/dev/zero", fixturePath) }},
		{name: "fifo", create: func() error { return syscall.Mkfifo(fixturePath, 0o600) }},
	} {
		if err := os.Remove(fixturePath); err != nil {
			t.Fatal(err)
		}
		if err := hostile.create(); err != nil {
			t.Fatal(err)
		}
		finished := make(chan error, 1)
		go func() { finished <- materializeCLIFixtures(roots.FixtureRoot(), input.view.Stimulus().Fixtures()) }()
		select {
		case err := <-finished:
			if err == nil {
				t.Fatalf("existing %s fixture leaf was accepted", hostile.name)
			}
		case <-time.After(time.Second):
			t.Fatalf("existing %s fixture leaf blocked bounded convergence", hostile.name)
		}
	}
	if err := os.Remove(fixturePath); err != nil {
		t.Fatal(err)
	}
	if err := materializeCLIFixtures(roots.FixtureRoot(), input.view.Stimulus().Fixtures()); err != nil {
		t.Fatalf("exact fixture could not be restored after hostile leaf controls: %v", err)
	}
	assertCandidateInventoryLimits(t)
}

func assertPrivateEvidenceCapacityAndFrame(t testing.TB, input cliExecutionInput) {
	t.Helper()
	capacity, err := preflightPrivateEvidenceCapacity(
		input.target.Model(), input.pre, privateEvidenceMaximumChannelBytes, privateEvidenceMaximumChannelBytes,
	)
	if err != nil || capacity.maximumUniqueBytes != 48<<20 {
		t.Fatalf("maximum legal C4 private-evidence capacity = %d, err=%v; want %d", capacity.maximumUniqueBytes, err, 48<<20)
	}
	for _, limits := range [][2]int64{
		{privateEvidenceMaximumChannelBytes + 1, privateEvidenceMaximumChannelBytes},
		{privateEvidenceMaximumChannelBytes, privateEvidenceMaximumChannelBytes + 1},
		{math.MaxInt64, 1},
	} {
		if _, err := preflightPrivateEvidenceCapacity(input.target.Model(), input.pre, limits[0], limits[1]); err == nil || !IsCode(err, CodeUnsupportedProfile) {
			t.Fatalf("out-of-envelope channel limits were admitted: limits=%v err=%v", limits, err)
		}
	}
	stdout := []byte{0x00, 0xff, 'o'}
	stderr := []byte("err\n")
	frame, err := framePrivateEvidence(
		"CAPTURE_AND_DRAIN",
		map[string]any{"exact_boundary_probe": true},
		privateEvidenceCaptureOverheadMaxBytes+int64(len(stdout)+len(stderr)),
		privateEvidenceSegment{name: "stdout", body: stdout},
		privateEvidenceSegment{name: "stderr", body: stderr},
	)
	if err != nil || !bytes.HasPrefix(frame, []byte(privateEvidenceFrameMagic)) {
		t.Fatalf("private capture frame did not encode: bytes=%d err=%v", len(frame), err)
	}
	offset := len(privateEvidenceFrameMagic)
	if len(frame) < offset+8 {
		t.Fatal("private capture frame omitted metadata length")
	}
	metadataBytes := int(binary.BigEndian.Uint64(frame[offset : offset+8]))
	offset += 8 + metadataBytes
	for index, want := range [][]byte{stdout, stderr} {
		if len(frame) < offset+8 {
			t.Fatalf("private capture frame omitted segment %d length", index)
		}
		count := int(binary.BigEndian.Uint64(frame[offset : offset+8]))
		offset += 8
		if count < 0 || len(frame) < offset+count || !bytes.Equal(frame[offset:offset+count], want) {
			t.Fatalf("private capture frame segment %d differs", index)
		}
		offset += count
	}
	if offset != len(frame) {
		t.Fatalf("private capture frame has trailing bytes: offset=%d bytes=%d", offset, len(frame))
	}
	if err := capacity.validate(map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidenceDrainResult: frame, contractmodel.EvidenceCapturedObservation: frame,
	}); err != nil {
		t.Fatalf("shared capture/drain frame escaped the capacity envelope: %v", err)
	}
	if err := capacity.validate(map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidenceDrainResult:         frame,
		contractmodel.EvidenceCapturedObservation: append(append([]byte(nil), frame...), 0),
	}); err == nil {
		t.Fatal("distinct raw output bodies were admitted for drain and captured observation")
	}
	fullRoster := map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidenceDrainResult:         frame,
		contractmodel.EvidenceCapturedObservation: frame,
	}
	for _, kind := range []contractmodel.EvidenceKind{
		contractmodel.EvidenceMaterializationRevalidation,
		contractmodel.EvidenceRuntimeRevalidation,
		contractmodel.EvidenceProcessResult,
		contractmodel.EvidenceWaitResult,
		contractmodel.EvidenceTeardownResult,
		contractmodel.EvidenceOrphanCheck,
		contractmodel.EvidenceFinalizationMarker,
		contractmodel.EvidenceProjectionResult,
		contractmodel.EvidenceTargetInventory,
		contractmodel.EvidenceChildBindings,
		contractmodel.EvidenceImportResolution,
		contractmodel.EvidenceServiceBindings,
		contractmodel.EvidenceSentinelInheritance,
	} {
		fullRoster[kind] = []byte("unique-private-evidence:" + string(kind))
	}
	aggregate := int64(len(frame))
	for kind, body := range fullRoster {
		if kind != contractmodel.EvidenceDrainResult && kind != contractmodel.EvidenceCapturedObservation {
			aggregate += int64(len(body))
		}
	}
	exactCapacity := cliEvidenceCapacity{maximumUniqueBytes: aggregate}
	exactErr := exactCapacity.validate(fullRoster)
	if len(fullRoster) != 15 || exactErr != nil {
		t.Fatalf("complete 15-kind/14-unique evidence roster escaped its exact envelope: kinds=%d bytes=%d err=%v", len(fullRoster), aggregate, exactErr)
	}
	if err := (cliEvidenceCapacity{maximumUniqueBytes: aggregate - 1}).validate(fullRoster); err == nil {
		t.Fatal("complete evidence roster crossed an envelope one byte below its exact aggregate")
	}
}

func assertBoundedForeignEvidenceAndScopeResidue(t testing.TB, primary cliExecutionInput) {
	t.Helper()
	evidenceFixture := clitest.NewReferenceTarget(t)
	evidencePreparation, err := prepareCLIExecution(evidenceFixture.Target)
	if err != nil {
		t.Fatal(err)
	}
	evidenceInput := cliExecutionInput{
		target: evidenceFixture.Target, cliExecutionPreparation: evidencePreparation,
	}
	t.Cleanup(func() { _ = evidenceInput.closeProbe() })
	assertPreparedTargetIdentityGuards(t, primary, evidenceInput)
	foreignEvidence := filepath.Join(evidenceInput.evidenceRoot, "foreign", strings.Repeat("d", 32), "sentinel")
	if err := os.MkdirAll(filepath.Dir(foreignEvidence), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreignEvidence, []byte("retain"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(evidenceInput.evidenceRoot, "foreign")) })
	if _, err := inspectAndRetireCLIInvocationEvidence(
		context.Background(), evidenceInput.evidenceRoot, evidenceInput.target.Roots().MarkerPath(),
		evidenceInput.attemptID, evidenceInput.logicalArgv, evidenceInput.evidenceAuthority,
	); err == nil {
		t.Fatal("foreign evidence residue did not fail closed")
	}
	if body, err := os.ReadFile(foreignEvidence); err != nil || string(body) != "retain" {
		t.Fatalf("foreign evidence residue was traversed or removed: body=%q err=%v", body, err)
	}

	scopeFixture := clitest.NewReferenceTarget(t)
	probe, err := newScopeProbe(scopeFixture.Target.Roots())
	if err != nil {
		t.Fatal(err)
	}
	foreignScope := filepath.Join(probe.root, "foreign", strings.Repeat("d", 32), "sentinel")
	if err := os.MkdirAll(filepath.Dir(foreignScope), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreignScope, []byte("retain"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(probe.root)
		_ = os.RemoveAll(probe.socketRoot)
	})
	measurements, finishErr := probe.finish(context.Background())
	if finishErr == nil || !measurements.Ambiguous {
		t.Fatalf("foreign scope residue became a clean observation: measurements=%+v err=%v", measurements, finishErr)
	}
	if body, err := os.ReadFile(foreignScope); err != nil || string(body) != "retain" {
		t.Fatalf("foreign scope residue was traversed or removed: body=%q err=%v", body, err)
	}

	moduleProbe, err := newScopeProbe(scopeFixture.Target.Roots())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(moduleProbe.root)
		_ = os.RemoveAll(moduleProbe.socketRoot)
	})
	changed := []byte(importCanaryModuleSource)
	changed[0] ^= 1
	if err := os.WriteFile(moduleProbe.importModule, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	measurements, finishErr = moduleProbe.finish(context.Background())
	if finishErr == nil || !measurements.Ambiguous {
		t.Fatalf("same-size scope module rewrite became clean: measurements=%+v err=%v", measurements, finishErr)
	}
}

func assertPreparedTargetIdentityGuards(t testing.TB, primary, independent cliExecutionInput) {
	t.Helper()
	if samePreparedTarget(cliExecutionPreparation{}, primary.target) {
		t.Fatal("zero preparation matched a live official target")
	}
	reopened, err := contractexec.ReopenOfficialTarget(context.Background(), primary.target)
	if err != nil {
		t.Fatalf("reopen primary target for identity guard: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if !samePreparedTarget(primary.cliExecutionPreparation, reopened) {
		t.Fatal("freshly reopened target did not match its prepared identity")
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close identity-guard target: %v", err)
	}
	if samePreparedTarget(primary.cliExecutionPreparation, reopened) {
		t.Fatal("closed target matched its prepared identity")
	}

	primaryModel := primary.targetIdentity.model
	independentModel := independent.targetIdentity.model
	if primaryModel.Equal(independentModel) {
		t.Fatal("independent target unexpectedly reused the prepared target model")
	}
	if primaryModel.Input().Tree != independentModel.Input().Tree {
		t.Fatal("identical reference files did not reuse one deterministic cached tree authority")
	}
	if primaryModel.Input().Attempt == independentModel.Input().Attempt ||
		primary.targetIdentity.roots.AttemptRoot() == independent.targetIdentity.roots.AttemptRoot() {
		t.Fatal("independent targets reused attempt or physical-root authority")
	}
	if samePreparedTarget(primary.cliExecutionPreparation, independent.target) {
		t.Fatal("different fresh target model matched the primary prepared identity")
	}

	crossRootIdentity, err := newPreparedTargetIdentity(
		primaryModel, primary.targetIdentity.bundle, independent.targetIdentity.roots,
	)
	if err != nil {
		t.Fatalf("construct same-model/different-roots identity: %v", err)
	}
	crossRootPreparation := primary.cliExecutionPreparation
	crossRootPreparation.targetIdentity = crossRootIdentity
	if samePreparedTarget(crossRootPreparation, primary.target) {
		t.Fatal("same model with different physical roots matched a live target")
	}
}

func assertCandidateInventoryLimits(t *testing.T) {
	t.Helper()
	newRoot := func(t *testing.T) string {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
		return root
	}
	write := func(t *testing.T, root, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	wantError := func(t *testing.T, err error, fragment string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), fragment) {
			t.Fatalf("candidate inventory error=%v, want fragment %q", err, fragment)
		}
	}
	t.Run("exact-boundaries", func(t *testing.T) {
		root := newRoot(t)
		write(t, root, "a", "1234")
		write(t, root, "b", "5678")
		limits := inventoryLimits{entries: 2, pathBytes: 2, depth: 1, fileBytes: 4, aggregateBytes: 8, readBatch: 2}
		snapshot, err := snapshotCandidateWithLimits(root, limits)
		if err != nil || !snapshot.valid() || len(snapshot.entries) != 2 {
			t.Fatalf("exact candidate inventory boundaries failed: entries=%d err=%v", len(snapshot.entries), err)
		}
	})
	t.Run("entry-limit", func(t *testing.T) {
		root := newRoot(t)
		for _, name := range []string{"a", "b", "c"} {
			write(t, root, name, "x")
		}
		_, err := snapshotCandidateWithLimits(root, inventoryLimits{
			entries: 2, pathBytes: 16, depth: 1, fileBytes: 1, aggregateBytes: 3, readBatch: 2,
		})
		wantError(t, err, "entry limit")
	})
	t.Run("single-file-limit", func(t *testing.T) {
		root := newRoot(t)
		file, err := os.Create(filepath.Join(root, "sparse"))
		if err != nil {
			t.Fatal(err)
		}
		truncateErr := file.Truncate(5)
		closeErr := file.Close()
		if truncateErr != nil || closeErr != nil {
			t.Fatal(errors.Join(truncateErr, closeErr))
		}
		_, err = snapshotCandidateWithLimits(root, inventoryLimits{
			entries: 1, pathBytes: 16, depth: 1, fileBytes: 4, aggregateBytes: 8, readBatch: 2,
		})
		wantError(t, err, "single-file byte limit")
	})
	t.Run("aggregate-limit", func(t *testing.T) {
		root := newRoot(t)
		write(t, root, "a", "1234")
		write(t, root, "b", "5678")
		_, err := snapshotCandidateWithLimits(root, inventoryLimits{
			entries: 2, pathBytes: 16, depth: 1, fileBytes: 4, aggregateBytes: 7, readBatch: 2,
		})
		wantError(t, err, "aggregate byte limit")
	})
	t.Run("depth-limit", func(t *testing.T) {
		root := newRoot(t)
		if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o700); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(root, "a", "b"), "c", "x")
		_, err := snapshotCandidateWithLimits(root, inventoryLimits{
			entries: 3, pathBytes: 32, depth: 2, fileBytes: 1, aggregateBytes: 1, readBatch: 2,
		})
		wantError(t, err, "depth limit")
	})
	t.Run("path-byte-limit", func(t *testing.T) {
		root := newRoot(t)
		write(t, root, "long-name", "x")
		_, err := snapshotCandidateWithLimits(root, inventoryLimits{
			entries: 1, pathBytes: 4, depth: 1, fileBytes: 1, aggregateBytes: 1, readBatch: 2,
		})
		wantError(t, err, "path-byte limit")
	})
	for _, special := range []struct {
		name   string
		create func(string) error
	}{
		{name: "symlink", create: func(path string) error { return os.Symlink("missing", path) }},
		{name: "fifo", create: func(path string) error { return syscall.Mkfifo(path, 0o600) }},
	} {
		t.Run(special.name, func(t *testing.T) {
			root := newRoot(t)
			if err := special.create(filepath.Join(root, "special")); err != nil {
				t.Fatal(err)
			}
			_, err := snapshotCandidateWithLimits(root, inventoryLimits{
				entries: 1, pathBytes: 16, depth: 1, fileBytes: 1, aggregateBytes: 1, readBatch: 2,
			})
			wantError(t, err, "candidate inventory")
		})
	}
}

func TestParentSentinelEnvironmentIsOmittedFromRealChild(t *testing.T) {
	t.Setenv(parentSecretSentinel, "ambient-parent-secret")
	t.Setenv(parentCredentialSentinel, "ambient-parent-credential")
	fixture := clitest.NewReferenceTarget(t)
	record, err := ExecuteCLI(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultContradicts {
		t.Fatalf("ambient-sentinel reference valid=%t result=%s err=%v", record.Valid(), record.Model().Result(), err)
	}
	run, err := store.OpenFinalizedRunRecord(context.Background(), fixture.Target.TargetRecord(), fixture.Target.Model())
	if err != nil || !run.Valid() {
		t.Fatalf("open ambient-sentinel reference run: valid=%t err=%v", run.Valid(), err)
	}
	for _, check := range run.Model().Witness().StandaloneScope().Checks() {
		if check.State() != contractmodel.ScopeCheckClean {
			t.Fatalf("ambient parent value contaminated scope domain %s: %s", check.Domain(), check.State())
		}
	}
	witness := run.Model().Witness()
	if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected ||
		witness.StandaloneScope().State() != contractmodel.ScopeComplete {
		t.Fatalf("ambient sentinel run did not stay clean/projected/complete: process=%s observation=%s scope=%s",
			witness.ProcessClosure().State(), witness.Observation().State(), witness.StandaloneScope().State())
	}
}

func TestSameTargetRetryRefusesAfterTerminalClosure(t *testing.T) {
	fixture := clitest.NewReferenceTarget(t)
	first, err := ExecuteCLI(context.Background(), fixture.Target)
	if err != nil || !first.Valid() {
		t.Fatalf("first exact owner did not complete: valid=%t err=%v", first.Valid(), err)
	}
	second, err := ExecuteCLI(context.Background(), fixture.Target)
	if err == nil || !IsCode(err, CodeAdmissionRefused) || second.Valid() {
		t.Fatalf("durably consumed target started twice: valid=%t err=%v", second.Valid(), err)
	}
	recovered, err := ResumeCLIClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != first.Digest() ||
		!recovered.Model().Equal(first.Model()) {
		t.Fatalf("classification-only recovery diverged after refused retry: valid=%t err=%v", recovered.Valid(), err)
	}
}

func preparedAdmissionProcess(t testing.TB) (*processmechanics.Prepared, domain.Digest) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	invocation, err := processmechanics.NewInvocation(
		"/bin/echo", []string{"echo", "countershape-c4-admission"}, []string{"LANG=C", "LC_ALL=C"},
		processmechanics.AbsentStdin(), root,
		processmechanics.Limits{
			StdoutBytes: 128, StderrBytes: 128,
			Execution: time.Second, Teardown: 500 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := processmechanics.Prepare(invocation)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := domain.ParseDigest(prepared.BindingDigest().String())
	if err != nil {
		t.Fatal(err)
	}
	return prepared, binding
}

func TestCallerCancellationAfterAdmissionStillClosesTerminalFacts(t *testing.T) {
	fixture := clitest.NewTarget(t, clitest.CancellationFiles(t), "C4 cancellation closure")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		record store.ContractExecutionRecord
		err    error
	}
	finished := make(chan outcome, 1)
	go func() {
		record, err := ExecuteCLI(ctx, fixture.Target)
		finished <- outcome{record: record, err: err}
	}()
	claimPath := filepath.Join(
		fixture.StoreRoot, "contract-execution", "operations", "start-claims",
		strings.TrimPrefix(fixture.Target.Digest().String(), "sha256:"),
	)
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case early := <-finished:
			t.Fatalf("execution ended before durable admission was observed: valid=%t err=%v", early.record.Valid(), early.err)
		default:
		}
		if _, err := os.Lstat(claimPath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("durable admission was not observable before cancellation deadline")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	result := <-finished
	if result.err != nil || !result.record.Valid() || result.record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("caller cancellation stranded terminal facts: valid=%t result=%s err=%v", result.record.Valid(), result.record.Model().Result(), result.err)
	}
	recovered, err := ResumeCLIClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != result.record.Digest() {
		t.Fatalf("canceled run did not recover without rerunning the contract subject: valid=%t err=%v", recovered.Valid(), err)
	}
}

func assertConsumeImmediatelyPrecedesStart(t *testing.T) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runner test source location unavailable")
	}
	sourcePath := filepath.Join(filepath.Dir(filename), "runner_darwin.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body *ast.BlockStmt
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "consumeAndStart" {
			body = function.Body
			break
		}
	}
	if body == nil {
		t.Fatal("consumeAndStart declaration is absent")
	}
	for index, statement := range body.List {
		ifStatement, ok := statement.(*ast.IfStmt)
		if !ok || !isSelectorCall(ifStatement.Init, "owner", "ConsumeForStart") {
			continue
		}
		if index+1 >= len(body.List) || !isSelectorCall(body.List[index+1], "prepared", "Start") {
			t.Fatal("ConsumeForStart is not the immediately preceding top-level operation before Prepared.Start")
		}
		return
	}
	t.Fatal("consumeAndStart does not contain the private consume operation")
}

func isSelectorCall(node ast.Node, receiver, method string) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(next ast.Node) bool {
		call, ok := next.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != method {
			return true
		}
		found = expressionName(selector.X) == receiver
		return !found
	})
	return found
}

func isSelectorCallWithFirstIdent(node ast.Node, receiver, method, argument string) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(next ast.Node) bool {
		call, ok := next.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		first, firstOK := call.Args[0].(*ast.Ident)
		if !ok || !firstOK || selector.Sel.Name != method || expressionName(selector.X) != receiver || first.Name != argument {
			return true
		}
		found = true
		return false
	})
	return found
}

func isIdentCall(node ast.Node, name string) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(next ast.Node) bool {
		call, ok := next.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		found = ok && identifier.Name == name
		return !found
	})
	return found
}

func expressionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		prefix := expressionName(value.X)
		if prefix == "" {
			return ""
		}
		return prefix + "." + value.Sel.Name
	default:
		return ""
	}
}
