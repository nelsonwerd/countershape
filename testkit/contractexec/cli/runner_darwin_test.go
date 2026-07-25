//go:build darwin && arm64 && cgo

package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/contractexec/runner"
	"github.com/nelsonwerd/countershape/internal/store"
	clitest "github.com/nelsonwerd/countershape/testkit/contractexec/cli"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

func TestMain(m *testing.M) { os.Exit(clitest.RunMain(m)) }

const c4PhysicalTestParallelism = 2

var c4PhysicalTestSlots = make(chan struct{}, c4PhysicalTestParallelism)

func runC4PhysicalTest(t *testing.T, exercise func()) {
	t.Helper()
	c4PhysicalTestSlots <- struct{}{}
	defer func() { <-c4PhysicalTestSlots }()
	exercise()
}

func TestCLIContractExecutionClosesStandaloneScope(t *testing.T) {
	t.Parallel()
	t.Run("reference-contradicts", func(t *testing.T) {
		t.Parallel()
		runC4PhysicalTest(t, func() { assertReferenceContractExecution(t) })
	})
	t.Run("physical-conforms", func(t *testing.T) {
		t.Parallel()
		runC4PhysicalTest(t, func() { assertConformingContractExecution(t) })
	})
}

func assertReferenceContractExecution(t *testing.T) {
	t.Helper()
	fixture := clitest.NewReferenceTarget(t)
	record, err := runner.ExecuteCLI(context.Background(), fixture.Target)
	if err != nil || !record.Valid() {
		t.Fatalf("reference execution did not publish an exact classification: valid=%t err=%v", record.Valid(), err)
	}
	if result := record.Model().Result(); result != contractmodel.ResultContradicts {
		t.Fatalf("clean deterministic reference classification = %s, want CONTRADICTS", result)
	}
	run := openRun(t, fixture)
	witness := run.Model().Witness()
	var drainRef contractmodel.EvidenceRef
	for _, reference := range witness.ProcessClosure().Evidence() {
		if reference.Kind() == contractmodel.EvidenceDrainResult {
			drainRef = reference
		}
	}
	capturedRef, captured := witness.Observation().CapturedRef()
	if !drainRef.Valid() || !captured || !capturedRef.Valid() || drainRef.Digest() != capturedRef.Digest() {
		t.Fatalf("drain and captured evidence did not share one persisted body: drain=%s captured=%s",
			drainRef.Digest(), capturedRef.Digest())
	}
	if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected ||
		witness.StandaloneScope().State() != contractmodel.ScopeComplete ||
		witness.Disposition() != contractmodel.DispositionEligibleClean {
		t.Fatalf(
			"reference closure states process=%s capture=%s scope=%s disposition=%s",
			witness.ProcessClosure().State(), witness.Observation().State(),
			witness.StandaloneScope().State(), witness.Disposition(),
		)
	}
	checks := witness.StandaloneScope().Checks()
	if len(checks) != 5 {
		t.Fatalf("standalone scope has %d checks, want 5", len(checks))
	}
	for _, check := range checks {
		if check.State() != contractmodel.ScopeCheckClean {
			t.Fatalf("reference domain %s = %s, want independently present clean", check.Domain(), check.State())
		}
		if evidence, ok := check.Evidence(); !ok || !evidence.Valid() {
			t.Fatalf("reference domain %s lacks typed retained evidence", check.Domain())
		}
	}
	recovered, err := runner.ResumeCLIClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("classification-only recovery changed the terminal result: valid=%t err=%v", recovered.Valid(), err)
	}
}

func assertConformingContractExecution(t *testing.T) {
	t.Helper()
	conforming := clitest.NewConformingTarget(t)
	conformingRecord, err := runner.ExecuteCLI(context.Background(), conforming.Target)
	if err != nil || !conformingRecord.Valid() || conformingRecord.Model().Result() != contractmodel.ResultConforms {
		t.Fatalf("physical conforming execution valid=%t result=%s err=%v",
			conformingRecord.Valid(), conformingRecord.Model().Result(), err)
	}
	conformingWitness := openRun(t, conforming).Model().Witness()
	if conformingWitness.ProcessClosure().State() != contractmodel.ProcessClean ||
		conformingWitness.Observation().State() != contractmodel.CaptureProjected ||
		conformingWitness.StandaloneScope().State() != contractmodel.ScopeComplete ||
		conformingWitness.Disposition() != contractmodel.DispositionEligibleClean {
		t.Fatalf("physical conforming execution lacks clean/projected/complete facts: disposition=%s",
			conformingWitness.Disposition())
	}
	conformingTuple, ok := conformingWitness.Observation().Tuple()
	if !ok {
		t.Fatal("physical conforming execution lacks its exact projected tuple")
	}
	conformingFields := conformingTuple.Fields()
	if len(conformingFields) != 2 {
		t.Fatalf("physical conforming tuple has %d fields, want exactly 2", len(conformingFields))
	}
	wantFieldIDs := []countercli.CLIFieldID{
		countercli.CLIFieldStdoutJSONMode,
		countercli.CLIFieldStdoutJSONSource,
	}
	for index, field := range conformingFields {
		text, isString := field.Value().PortableValue().StringText()
		if field.FieldID() != string(wantFieldIDs[index]) || !isString || text != "argv" {
			t.Fatalf("physical conforming field %d id=%q string=%t value=%q, want %q/argv",
				index, field.FieldID(), isString, text, wantFieldIDs[index])
		}
	}
}

func TestCLIContractExecutionForbiddenPositiveControls(t *testing.T) {
	t.Parallel()
	runC4PhysicalTest(t, func() { assertForbiddenPositiveControls(t) })
}

func assertForbiddenPositiveControls(t *testing.T) {
	t.Helper()
	fixture := clitest.NewForbiddenTarget(t)
	record, err := runner.ExecuteCLI(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("positive control classification valid=%t result=%s err=%v", record.Valid(), record.Model().Result(), err)
	}
	run := openRun(t, fixture)
	witness := run.Model().Witness()
	scope := witness.StandaloneScope()
	if scope.State() != contractmodel.ScopeViolated || run.Model().Disposition() != contractmodel.DispositionIneligibleStandalone {
		t.Fatalf("positive control scope=%s disposition=%s", scope.State(), run.Model().Disposition())
	}
	assertPhysicalScope(t, witness, contractmodel.ScopeViolated, map[contractmodel.ScopeDomain]scopeExpectation{
		contractmodel.ScopeTargetInventory:     {state: contractmodel.ScopeCheckMissing},
		contractmodel.ScopeChildBindings:       {state: contractmodel.ScopeCheckMissing},
		contractmodel.ScopeImportResolution:    {state: contractmodel.ScopeCheckViolated, violation: contractmodel.ViolationImportResolution},
		contractmodel.ScopeServiceBindings:     {state: contractmodel.ScopeCheckViolated, violation: contractmodel.ViolationServiceBinding},
		contractmodel.ScopeSentinelInheritance: {state: contractmodel.ScopeCheckMissing},
	})
}

func TestCLIContractExecutionChildBindingEvidenceStates(t *testing.T) {
	cases := []struct {
		name  string
		files func(testing.TB) []gitrepo.File
		state contractmodel.ScopeCheckState
	}{
		{name: "attempt-mismatch", files: clitest.ChildBindingMismatchFiles, state: contractmodel.ScopeCheckViolated},
		{name: "argv-mismatch", files: clitest.ChildArgvMismatchFiles, state: contractmodel.ScopeCheckViolated},
		{name: "malformed", files: clitest.MalformedInvocationFiles, state: contractmodel.ScopeCheckMissing},
		{name: "absent", files: clitest.MissingInvocationFiles, state: contractmodel.ScopeCheckMissing},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := clitest.NewTarget(t, testCase.files(t), "C4 child binding "+testCase.name)
			record, err := runner.ExecuteCLI(context.Background(), fixture.Target)
			if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
				t.Fatalf("child binding control valid=%t result=%s err=%v", record.Valid(), record.Model().Result(), err)
			}
			witness := openRun(t, fixture).Model().Witness()
			scopeState := contractmodel.ScopePartial
			childExpectation := scopeExpectation{state: testCase.state}
			if testCase.state == contractmodel.ScopeCheckViolated {
				scopeState = contractmodel.ScopeViolated
				childExpectation.violation = contractmodel.ViolationChildBinding
			}
			assertPhysicalScope(t, witness, scopeState, map[contractmodel.ScopeDomain]scopeExpectation{
				contractmodel.ScopeTargetInventory:     {state: contractmodel.ScopeCheckMissing},
				contractmodel.ScopeChildBindings:       childExpectation,
				contractmodel.ScopeImportResolution:    {state: contractmodel.ScopeCheckMissing},
				contractmodel.ScopeServiceBindings:     {state: contractmodel.ScopeCheckMissing},
				contractmodel.ScopeSentinelInheritance: {state: contractmodel.ScopeCheckMissing},
			})
			entries, err := os.ReadDir(fixture.Target.Roots().EvidenceRoot())
			if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(fixture.Target.Roots().MarkerPath()) {
				t.Fatalf("child receipt was not retired to marker-only evidence root: entries=%v err=%v", entries, err)
			}
		})
	}
}

type scopeExpectation struct {
	state     contractmodel.ScopeCheckState
	violation contractmodel.ScopeViolation
}

func assertPhysicalScope(
	t testing.TB,
	witness contractmodel.ClosedRunWitness,
	wantScope contractmodel.ScopeState,
	want map[contractmodel.ScopeDomain]scopeExpectation,
) {
	t.Helper()
	scope := witness.StandaloneScope()
	if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected || scope.State() != wantScope {
		t.Fatalf("isolated physical states process=%s capture=%s scope=%s, want PROCESS_CLEAN/PROJECTED/%s",
			witness.ProcessClosure().State(), witness.Observation().State(), scope.State(), wantScope)
	}
	checks := scope.Checks()
	if len(checks) != len(want) {
		t.Fatalf("standalone scope has %d checks, want exactly %d", len(checks), len(want))
	}
	remaining := make(map[contractmodel.ScopeDomain]scopeExpectation, len(want))
	for domain, expectation := range want {
		remaining[domain] = expectation
	}
	for _, check := range checks {
		expectation, ok := remaining[check.Domain()]
		if !ok {
			t.Fatalf("standalone scope contains unexpected or duplicate domain %s", check.Domain())
		}
		if check.State() != expectation.state {
			t.Fatalf("scope domain %s state=%s, want %s", check.Domain(), check.State(), expectation.state)
		}
		violation, violated := check.Violation()
		if expectation.violation == "" {
			if violated {
				t.Fatalf("scope domain %s carried unexpected violation %s", check.Domain(), violation)
			}
		} else if !violated || violation != expectation.violation {
			t.Fatalf("scope domain %s violation=%s present=%t, want %s", check.Domain(), violation, violated, expectation.violation)
		}
		delete(remaining, check.Domain())
	}
	if len(remaining) != 0 {
		t.Fatalf("standalone scope omitted expected domains: %v", remaining)
	}
}

func TestCLIContractExecutionTargetMutationBlocksFinalization(t *testing.T) {
	fixture := clitest.NewMutationTarget(t, clitest.TargetMutationFiles(t), "C4 target mutation positive control")
	record, err := runner.ExecuteCLI(context.Background(), fixture.Target)
	if err == nil || !runner.IsCode(err, runner.CodeTargetChanged) || record.Valid() {
		t.Fatalf("candidate mutation escaped terminal target revalidation: valid=%t err=%v", record.Valid(), err)
	}
	if _, openErr := store.OpenFinalizedRunRecord(
		context.Background(), fixture.Target.TargetRecord(), fixture.Target.Model(),
	); openErr == nil {
		t.Fatal("candidate mutation published a finalized run despite failed terminal revalidation")
	}
}

func openRun(t testing.TB, fixture clitest.Fixture) store.FinalizedRunRecord {
	t.Helper()
	run, err := store.OpenFinalizedRunRecord(
		context.Background(), fixture.Target.TargetRecord(), fixture.Target.Model(),
	)
	if err != nil || !run.Valid() {
		t.Fatalf("open exact finalized run: valid=%t err=%v", run.Valid(), err)
	}
	return run
}
