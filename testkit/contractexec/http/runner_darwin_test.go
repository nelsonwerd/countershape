//go:build darwin && arm64 && cgo

package http_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	contracthttp "github.com/nelsonwerd/countershape/internal/contractexec/http"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	contractscope "github.com/nelsonwerd/countershape/internal/contractexec/scope"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/store"
	httpfixture "github.com/nelsonwerd/countershape/testkit/contractexec/http"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
	studyfixture "github.com/nelsonwerd/countershape/testkit/httpfixture"
)

func TestMain(m *testing.M) { os.Exit(httpfixture.RunMain(m)) }

func diagnosticErrorChain(err error) string {
	if err == nil {
		return "<nil>"
	}
	parts := make([]string, 0, 8)
	var visit func(error, int)
	visit = func(current error, depth int) {
		if current == nil || depth >= 16 || len(parts) >= 64 {
			return
		}
		parts = append(parts, current.Error())
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range wrapped.Unwrap() {
				visit(child, depth+1)
			}
		case interface{ Unwrap() error }:
			visit(wrapped.Unwrap(), depth+1)
		}
	}
	visit(err, 0)
	return strings.Join(parts, " <- ")
}

func TestHTTPContractExecutionConformingDifferingCustom401AndAllowMany(t *testing.T) {
	cases := []struct {
		name   string
		target func(testing.TB) httpfixture.Fixture
		want   contractmodel.ExecutionResult
		status string
	}{
		{
			name: "observed-404-conforms",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewReferenceTarget(t)
			},
			want: contractmodel.ResultConforms, status: "404",
		},
		{
			name: "observed-403-differs",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewDifferingTarget(t)
			},
			want: contractmodel.ResultContradicts, status: "403",
		},
		{
			name: "custom-401-differs-from-403",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewCustom401Target(t, studyfixture.Forbidden)
			},
			want: contractmodel.ResultContradicts, status: "403",
		},
		{
			name: "allow-many-includes-403",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewAllowManyTarget(t, studyfixture.Forbidden)
			},
			want: contractmodel.ResultConforms, status: "403",
		},
		{
			name: "allow-many-includes-404",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewAllowManyTarget(t, studyfixture.ConcealNotFound)
			},
			want: contractmodel.ResultConforms, status: "404",
		},
		{
			name: "allow-many-excludes-200",
			target: func(t testing.TB) httpfixture.Fixture {
				return httpfixture.NewAllowManyTarget(t, studyfixture.MetadataDisclosure)
			},
			want: contractmodel.ResultContradicts, status: "200",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := testCase.target(t)
			record, err := contracthttp.Execute(context.Background(), fixture.Target)
			if err != nil || !record.Valid() || record.Model().Result() != testCase.want {
				t.Fatalf("HTTP classification valid=%t result=%s want=%s err=%v",
					record.Valid(), record.Model().Result(), testCase.want, err)
			}
			witness := openRun(t, fixture).Model().Witness()
			assertCleanHTTPWitness(t, witness, testCase.status)
		})
	}
}

func TestHTTPChildReportedReadinessBindsExactService(t *testing.T) {
	fixture := httpfixture.NewReferenceTarget(t)
	inventory, inventoryErr := contractscope.Snapshot(fixture.Target.CandidateRoot())
	if inventoryErr != nil || !inventory.ReferenceHTTPFixture() {
		t.Fatalf(
			"reference target is not enrolled: valid=%t entries=%#v err=%v",
			inventory.Valid(), inventory.Entries(), inventoryErr,
		)
	}
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() {
		t.Fatalf("reference readiness execution valid=%t err=%v cause=%v",
			record.Valid(), err, errors.Unwrap(err))
	}
	run := openRun(t, fixture)
	witness := run.Model().Witness()
	if record.Model().Result() != contractmodel.ResultConforms {
		for index, check := range witness.StandaloneScope().Checks() {
			evidence, present := check.Evidence()
			violation, violated := check.Violation()
			t.Logf(
				"scope[%d] domain=%s state=%s evidence=%s present=%t violation=%s violated=%t",
				index, check.Domain(), check.State(), evidence.Kind(), present, violation, violated,
			)
		}
		t.Fatalf(
			"reference readiness result=%s process=%s capture=%s scope=%s disposition=%s",
			record.Model().Result(), witness.ProcessClosure().State(),
			witness.Observation().State(), witness.StandaloneScope().State(),
			witness.Disposition(),
		)
	}
	assertCleanHTTPWitness(t, witness, "404")
	checks := witness.StandaloneScope().Checks()
	var service, child contractmodel.ScopeCheck
	for _, check := range checks {
		switch check.Domain() {
		case contractmodel.ScopeServiceBindings:
			service = check
		case contractmodel.ScopeChildBindings:
			child = check
		}
	}
	serviceEvidence, servicePresent := service.Evidence()
	childEvidence, childPresent := child.Evidence()
	if service.State() != contractmodel.ScopeCheckClean || !servicePresent ||
		serviceEvidence.Kind() != contractmodel.EvidenceServiceBindings ||
		child.State() != contractmodel.ScopeCheckClean || !childPresent ||
		childEvidence.Kind() != contractmodel.EvidenceChildBindings ||
		serviceEvidence.Digest() == childEvidence.Digest() {
		t.Fatalf("child/service bindings did not retain distinct exact evidence: service=%#v child=%#v",
			serviceEvidence, childEvidence)
	}
	recovered, err := contracthttp.ResumeClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("classification-only recovery changed readiness result: valid=%t err=%v", recovered.Valid(), err)
	}
}

func TestHTTPEarlyExitAndTeardownRetainCausalFacts(t *testing.T) {
	files := httpfixture.EarlyExitFiles(t, studyfixture.ConcealNotFound)
	fixture := httpfixture.NewControlTarget(t, files, "C5 HTTP early-exit control")
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("early-exit classification valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	witness := openRun(t, fixture).Model().Witness()
	process := witness.ProcessClosure()
	primary, controlled := process.Primary()
	if process.State() != contractmodel.ProcessControlled || !controlled ||
		primary != domain.ControlReadinessError || process.TeardownError() ||
		process.OrphanRisk() || witness.Observation().State() != contractmodel.CaptureNone ||
		witness.StandaloneScope().State() != contractmodel.ScopePartial ||
		witness.Disposition() != contractmodel.DispositionIneligibleControlStandalone {
		t.Fatalf(
			"early-exit closure process=%s primary=%s teardown=%t orphan=%t capture=%s scope=%s disposition=%s",
			process.State(), primary, process.TeardownError(), process.OrphanRisk(),
			witness.Observation().State(), witness.StandaloneScope().State(), witness.Disposition(),
		)
	}
	wantKinds := []contractmodel.EvidenceKind{
		contractmodel.EvidenceMaterializationRevalidation,
		contractmodel.EvidenceRuntimeRevalidation,
		contractmodel.EvidenceProcessResult,
		contractmodel.EvidenceWaitResult,
		contractmodel.EvidenceDrainResult,
		contractmodel.EvidenceTeardownResult,
		contractmodel.EvidenceOrphanCheck,
		contractmodel.EvidenceFinalizationMarker,
	}
	evidence := process.Evidence()
	if len(evidence) != len(wantKinds) {
		t.Fatalf("early-exit process evidence count=%d want=%d", len(evidence), len(wantKinds))
	}
	for index, reference := range evidence {
		if !reference.Valid() || reference.Kind() != wantKinds[index] {
			t.Fatalf("early-exit process evidence %d = %s, want %s",
				index, reference.Kind(), wantKinds[index])
		}
	}
}

func TestHTTPDecoyReadinessCannotBecomeEligible(t *testing.T) {
	fixture := httpfixture.NewControlTarget(
		t,
		httpfixture.DecoyReadinessFiles(t, studyfixture.ConcealNotFound),
		"C5 decoy readiness control",
	)
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("decoy classification valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	witness := openRun(t, fixture).Model().Witness()
	if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected ||
		witness.StandaloneScope().State() != contractmodel.ScopePartial ||
		witness.Disposition() != contractmodel.DispositionIneligibleStandalone {
		t.Fatalf("decoy closure process=%s capture=%s scope=%s disposition=%s",
			witness.ProcessClosure().State(), witness.Observation().State(),
			witness.StandaloneScope().State(), witness.Disposition())
	}
	assertProjectedHTTPStatus(t, witness, "418")
	assertEveryScopeCheckState(t, witness, contractmodel.ScopeCheckMissing)
	assertMarkerOnlyEvidenceRoot(t, fixture)
}

func TestHTTPReadinessControlMatrixClosesExactly(t *testing.T) {
	cases := []struct {
		name    string
		files   func(testing.TB, studyfixture.CandidateRole) []gitrepo.File
		primary domain.ControlReason
	}{
		{name: "partial-eof", files: httpfixture.PartialReadinessFiles, primary: domain.ControlReadinessError},
		{name: "malformed-eof", files: httpfixture.MalformedReadinessFiles, primary: domain.ControlReadinessError},
		{name: "withheld-timeout", files: httpfixture.WithheldReadinessFiles, primary: domain.ControlReadinessError},
		{name: "wrong-endpoint-reset", files: httpfixture.WrongEndpointFiles, primary: domain.ControlProbeTransportError},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := httpfixture.NewControlTarget(
				t,
				testCase.files(t, studyfixture.ConcealNotFound),
				"C5 readiness "+testCase.name,
			)
			assertControlledHTTPWitness(t, fixture, testCase.primary, contractmodel.CaptureNone)
		})
	}
}

func TestHTTPResponseControlMatrixRetainsBoundedCapture(t *testing.T) {
	cases := []struct {
		name    string
		files   func(testing.TB, studyfixture.CandidateRole) []gitrepo.File
		primary domain.ControlReason
		capture contractmodel.CaptureState
	}{
		{
			name: "malformed-status", files: httpfixture.MalformedResponseFiles,
			primary: domain.ControlProbeTransportError, capture: contractmodel.CaptureUnprojected,
		},
		{
			name: "partial-body", files: httpfixture.PartialResponseFiles,
			primary: domain.ControlProbeTransportError, capture: contractmodel.CaptureUnprojected,
		},
		{
			name: "physical-response-limit", files: httpfixture.OverflowResponseFiles,
			primary: domain.ControlOutputLimit, capture: contractmodel.CaptureUnprojected,
		},
		{
			name: "no-response-timeout", files: httpfixture.ResponseTimeoutFiles,
			primary: domain.ControlTimeout, capture: contractmodel.CaptureNone,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := httpfixture.NewControlTarget(
				t,
				testCase.files(t, studyfixture.ConcealNotFound),
				"C5 response "+testCase.name,
			)
			assertControlledHTTPWitness(t, fixture, testCase.primary, testCase.capture)
		})
	}
}

func TestHTTPContractExecutionScopePositiveControls(t *testing.T) {
	fixture := httpfixture.NewControlTarget(
		t,
		httpfixture.ForbiddenScopeFiles(t, studyfixture.ConcealNotFound),
		"C5 four-domain scope positive control",
	)
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("scope-positive classification valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	witness := openRun(t, fixture).Model().Witness()
	if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected ||
		witness.StandaloneScope().State() != contractmodel.ScopeViolated ||
		witness.Disposition() != contractmodel.DispositionIneligibleStandalone {
		t.Fatalf("scope-positive closure process=%s capture=%s scope=%s disposition=%s",
			witness.ProcessClosure().State(), witness.Observation().State(),
			witness.StandaloneScope().State(), witness.Disposition())
	}
	assertProjectedHTTPStatus(t, witness, "404")
	want := map[contractmodel.ScopeDomain]contractmodel.ScopeViolation{
		contractmodel.ScopeTargetInventory:  contractmodel.ViolationTargetSourceAndDependency,
		contractmodel.ScopeChildBindings:    contractmodel.ViolationChildBinding,
		contractmodel.ScopeImportResolution: contractmodel.ViolationImportResolution,
		contractmodel.ScopeServiceBindings:  contractmodel.ViolationServiceBinding,
	}
	for _, check := range witness.StandaloneScope().Checks() {
		if check.Domain() == contractmodel.ScopeSentinelInheritance {
			if check.State() != contractmodel.ScopeCheckMissing {
				t.Fatalf("scope-positive sentinel state=%s, want missing", check.State())
			}
			continue
		}
		violation, present := check.Violation()
		evidence, retained := check.Evidence()
		if check.State() != contractmodel.ScopeCheckViolated || !present ||
			violation != want[check.Domain()] || !retained || !evidence.Valid() {
			t.Fatalf("scope-positive domain=%s state=%s violation=%s present=%t evidence=%t",
				check.Domain(), check.State(), violation, present, retained)
		}
		delete(want, check.Domain())
	}
	if len(want) != 0 {
		t.Fatalf("scope-positive controls omitted domains: %v", want)
	}
	assertMarkerOnlyEvidenceRoot(t, fixture)
}

func TestHTTPContractExecutionReceiptEvidenceStates(t *testing.T) {
	cases := []struct {
		name      string
		files     func(testing.TB, studyfixture.CandidateRole) []gitrepo.File
		child     contractmodel.ScopeCheckState
		scope     contractmodel.ScopeState
		violation bool
	}{
		{name: "absent", files: httpfixture.MissingInvocationFiles, child: contractmodel.ScopeCheckMissing, scope: contractmodel.ScopePartial},
		{name: "malformed", files: httpfixture.MalformedInvocationFiles, child: contractmodel.ScopeCheckMissing, scope: contractmodel.ScopePartial},
		{name: "attempt-mismatch", files: httpfixture.ChildBindingMismatchFiles, child: contractmodel.ScopeCheckViolated, scope: contractmodel.ScopeViolated, violation: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := httpfixture.NewControlTarget(
				t,
				testCase.files(t, studyfixture.ConcealNotFound),
				"C5 receipt "+testCase.name,
			)
			record, err := contracthttp.Execute(context.Background(), fixture.Target)
			if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
				t.Fatalf("receipt %s classification valid=%t result=%s err=%v",
					testCase.name, record.Valid(), record.Model().Result(), err)
			}
			witness := openRun(t, fixture).Model().Witness()
			if witness.ProcessClosure().State() != contractmodel.ProcessClean ||
				witness.Observation().State() != contractmodel.CaptureProjected ||
				witness.StandaloneScope().State() != testCase.scope ||
				witness.Disposition() != contractmodel.DispositionIneligibleStandalone {
				t.Fatalf("receipt %s process=%s capture=%s scope=%s disposition=%s",
					testCase.name, witness.ProcessClosure().State(), witness.Observation().State(),
					witness.StandaloneScope().State(), witness.Disposition())
			}
			assertProjectedHTTPStatus(t, witness, "404")
			for _, check := range witness.StandaloneScope().Checks() {
				want := contractmodel.ScopeCheckMissing
				if check.Domain() == contractmodel.ScopeChildBindings {
					want = testCase.child
				}
				if check.State() != want {
					t.Fatalf("receipt %s domain=%s state=%s want=%s",
						testCase.name, check.Domain(), check.State(), want)
				}
				violation, present := check.Violation()
				if check.Domain() == contractmodel.ScopeChildBindings && testCase.violation {
					if !present || violation != contractmodel.ViolationChildBinding {
						t.Fatalf("receipt %s child violation=%s present=%t", testCase.name, violation, present)
					}
				} else if present {
					t.Fatalf("receipt %s domain=%s carried violation=%s", testCase.name, check.Domain(), violation)
				}
			}
			assertMarkerOnlyEvidenceRoot(t, fixture)
		})
	}
}

func TestHTTPParentSentinelsAreOmitted(t *testing.T) {
	for _, name := range contractscope.ParentSentinels() {
		t.Setenv(name, "must-not-cross-the-child-boundary")
	}
	fixture := httpfixture.NewReferenceTarget(t)
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultConforms {
		t.Fatalf("sentinel-omission classification valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	witness := openRun(t, fixture).Model().Witness()
	assertCleanHTTPWitness(t, witness, "404")
	for _, check := range witness.StandaloneScope().Checks() {
		if check.Domain() != contractmodel.ScopeSentinelInheritance {
			continue
		}
		evidence, present := check.Evidence()
		if check.State() != contractmodel.ScopeCheckClean || !present || !evidence.Valid() ||
			evidence.Kind() != contractmodel.EvidenceSentinelInheritance {
			t.Fatalf("sentinel omission state=%s evidence=%t kind=%s",
				check.State(), present, evidence.Kind())
		}
		return
	}
	t.Fatal("sentinel omission witness lacks the named domain")
}

func TestHTTPExactTargetRunJoinsRefuseCrossTargetConfusion(t *testing.T) {
	firstFixture := httpfixture.NewReferenceTarget(t)
	firstRecord, err := contracthttp.Execute(context.Background(), firstFixture.Target)
	if err != nil || !firstRecord.Valid() {
		t.Fatalf("first exact target execution valid=%t err=%v", firstRecord.Valid(), err)
	}
	firstRun := openRun(t, firstFixture)

	secondFixture := httpfixture.NewReferenceTarget(t)
	secondRecord, err := contracthttp.Execute(context.Background(), secondFixture.Target)
	if err != nil || !secondRecord.Valid() {
		t.Fatalf("second exact target execution valid=%t err=%v", secondRecord.Valid(), err)
	}
	secondRun := openRun(t, secondFixture)

	if firstFixture.Target.Digest() == secondFixture.Target.Digest() ||
		firstRun.Digest() == secondRun.Digest() ||
		firstRecord.Digest() == secondRecord.Digest() {
		t.Fatal("fresh target, run, and execution identities did not remain distinct")
	}
	for _, item := range []struct {
		fixture httpfixture.Fixture
		run     store.FinalizedRunRecord
		record  store.ContractExecutionRecord
	}{
		{fixture: firstFixture, run: firstRun, record: firstRecord},
		{fixture: secondFixture, run: secondRun, record: secondRecord},
	} {
		if item.run.Model().TargetDigest() != item.fixture.Target.Digest() ||
			item.record.Model().TargetDigest() != item.fixture.Target.Digest() ||
			item.record.Model().FinalizedRunDigest() != item.run.Digest() {
			t.Fatal("target/run/execution relationship did not join exactly")
		}
	}
	if _, openErr := store.OpenFinalizedRunRecord(
		context.Background(),
		secondFixture.Target.TargetRecord(),
		firstFixture.Target.Model(),
	); openErr == nil {
		t.Fatal("second target record opened against the first target model")
	}
	if _, deriveErr := contractmodel.DeriveContractExecution(
		firstFixture.Target.ContractBundle(),
		firstFixture.Target.Model(),
		secondRun.Model(),
	); deriveErr == nil {
		t.Fatal("first target derived a classification from the second run")
	}
	if _, persistErr := store.PersistContractExecutionRecord(
		context.Background(),
		firstRun,
		secondRecord.Model(),
	); persistErr == nil {
		t.Fatal("second execution persisted against the first released run")
	}
	for _, item := range []struct {
		fixture httpfixture.Fixture
		record  store.ContractExecutionRecord
	}{
		{fixture: firstFixture, record: firstRecord},
		{fixture: secondFixture, record: secondRecord},
	} {
		recovered, recoverErr := contracthttp.ResumeClassification(context.Background(), item.fixture.Target)
		if recoverErr != nil || !recovered.Valid() || recovered.Digest() != item.record.Digest() {
			t.Fatalf("cross-target recovery changed exact record: valid=%t err=%v",
				recovered.Valid(), recoverErr)
		}
	}
}

func TestHTTPClassificationRecoveryConvergesTerminalClosureWithoutRespawn(t *testing.T) {
	fixture := httpfixture.NewReferenceTarget(t)
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() {
		t.Fatalf(
			"initial recovery subject valid=%t err=%v chain=%s",
			record.Valid(),
			err,
			diagnosticErrorChain(err),
		)
	}
	run := openRun(t, fixture)
	recovered, err := contracthttp.ResumeClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("released-run recovery valid=%t err=%v", recovered.Valid(), err)
	}

	runHex := strings.TrimPrefix(run.Digest().String(), "sha256:")
	if len(runHex) != 64 {
		t.Fatalf("finalized run digest has unexpected spelling %q", run.Digest())
	}
	releaseDirectory := filepath.Join(
		fixture.StoreRoot,
		"contract-execution",
		"operations",
		"finalized-run-releases",
	)
	releasePath := filepath.Join(releaseDirectory, runHex)
	if err := os.Remove(releasePath); err != nil {
		t.Fatalf("remove test-owned finalized release link: %v", err)
	}
	directory, err := os.Open(releaseDirectory)
	if err != nil {
		t.Fatalf("open finalized release directory: %v", err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil || closeErr != nil {
		t.Fatalf("sync removed finalized release link: sync=%v close=%v", syncErr, closeErr)
	}

	recovered, err = contracthttp.ResumeClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != record.Digest() {
		t.Fatalf("terminal-closure recovery valid=%t err=%v", recovered.Valid(), err)
	}
	if info, statErr := os.Lstat(releasePath); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("terminal recovery did not restore exact release link: mode=%v err=%v", info, statErr)
	}
	if second, secondErr := contracthttp.Execute(context.Background(), fixture.Target); secondErr == nil ||
		!contracthttp.IsCode(secondErr, contracthttp.CodeAdmissionRefused) || second.Valid() {
		t.Fatalf("classification recovery reopened spawn authority: valid=%t err=%v", second.Valid(), secondErr)
	}
	assertMarkerOnlyEvidenceRoot(t, fixture)
}

func TestHTTPConcurrentExecuteAdmitsExactlyOneSubject(t *testing.T) {
	fixture := httpfixture.NewReferenceTarget(t)
	type outcome struct {
		record store.ContractExecutionRecord
		err    error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			record, err := contracthttp.Execute(context.Background(), fixture.Target)
			results <- outcome{record: record, err: err}
		}()
	}
	ready.Wait()
	close(start)

	outcomes := make([]outcome, 0, 2)
	for range 2 {
		outcomes = append(outcomes, <-results)
	}
	var winner store.ContractExecutionRecord
	losers := 0
	loserErrors := make([]error, 0, 2)
	unexpected := false
	for _, result := range outcomes {
		switch {
		case result.err == nil && result.record.Valid():
			if winner.Valid() {
				t.Fatal("concurrent execution produced more than one valid classification")
			}
			winner = result.record
		case result.err != nil && contracthttp.IsCode(result.err, contracthttp.CodeAdmissionRefused) &&
			!result.record.Valid():
			losers++
			loserErrors = append(loserErrors, result.err)
		default:
			unexpected = true
		}
	}
	if unexpected {
		t.Fatalf(
			"concurrent execution returned unexpected outcomes: first_valid=%t first_chain=%s second_valid=%t second_chain=%s",
			outcomes[0].record.Valid(),
			diagnosticErrorChain(outcomes[0].err),
			outcomes[1].record.Valid(),
			diagnosticErrorChain(outcomes[1].err),
		)
	}
	if !winner.Valid() || winner.Model().Result() != contractmodel.ResultConforms || losers != 1 {
		chains := make([]string, len(loserErrors))
		for index, loserErr := range loserErrors {
			chains[index] = diagnosticErrorChain(loserErr)
		}
		t.Fatalf(
			"concurrent admission winner=%t result=%s losers=%d loser_chains=%q",
			winner.Valid(),
			winner.Model().Result(),
			losers,
			chains,
		)
	}
	recovered, err := contracthttp.ResumeClassification(context.Background(), fixture.Target)
	if err != nil || !recovered.Valid() || recovered.Digest() != winner.Digest() {
		t.Fatalf("concurrent winner did not recover exactly: valid=%t err=%v", recovered.Valid(), err)
	}
	assertMarkerOnlyEvidenceRoot(t, fixture)
}

func assertControlledHTTPWitness(
	t testing.TB,
	fixture httpfixture.Fixture,
	wantPrimary domain.ControlReason,
	wantCapture contractmodel.CaptureState,
) {
	t.Helper()
	record, err := contracthttp.Execute(context.Background(), fixture.Target)
	if err != nil || !record.Valid() || record.Model().Result() != contractmodel.ResultIneligible {
		t.Fatalf("controlled HTTP classification valid=%t result=%s err=%v",
			record.Valid(), record.Model().Result(), err)
	}
	witness := openRun(t, fixture).Model().Witness()
	process := witness.ProcessClosure()
	primary, controlled := process.Primary()
	if process.State() != contractmodel.ProcessControlled || !controlled || primary != wantPrimary ||
		process.TeardownError() || process.OrphanRisk() ||
		witness.Observation().State() != wantCapture ||
		witness.StandaloneScope().State() != contractmodel.ScopePartial ||
		witness.Disposition() != contractmodel.DispositionIneligibleControlStandalone {
		t.Fatalf("controlled HTTP process=%s primary=%s present=%t teardown=%t orphan=%t capture=%s scope=%s disposition=%s",
			process.State(), primary, controlled, process.TeardownError(), process.OrphanRisk(),
			witness.Observation().State(), witness.StandaloneScope().State(), witness.Disposition())
	}
	if _, present := witness.Observation().Tuple(); present {
		t.Fatal("controlled HTTP observation unexpectedly retained a projected tuple")
	}
	if wantCapture == contractmodel.CaptureUnprojected {
		if evidence, present := witness.Observation().CapturedRef(); !present || !evidence.Valid() {
			t.Fatal("captured-unprojected HTTP observation lacks bounded captured evidence")
		}
	}
	assertEveryScopeCheckState(t, witness, contractmodel.ScopeCheckMissing)
	assertMarkerOnlyEvidenceRoot(t, fixture)
}

func assertEveryScopeCheckState(
	t testing.TB,
	witness contractmodel.ClosedRunWitness,
	want contractmodel.ScopeCheckState,
) {
	t.Helper()
	checks := witness.StandaloneScope().Checks()
	if len(checks) != 5 {
		t.Fatalf("standalone scope has %d checks, want five", len(checks))
	}
	for _, check := range checks {
		if check.State() != want {
			t.Fatalf("scope domain %s state=%s want=%s", check.Domain(), check.State(), want)
		}
	}
}

func assertProjectedHTTPStatus(
	t testing.TB,
	witness contractmodel.ClosedRunWitness,
	wantStatus string,
) {
	t.Helper()
	tuple, present := witness.Observation().Tuple()
	if !present {
		t.Fatal("projected HTTP witness has no exact tuple")
	}
	fields := tuple.Fields()
	if len(fields) != 1 || fields[0].FieldID() != string(counterhttp.HTTPFieldStatus) {
		t.Fatalf("HTTP tuple fields = %#v", fields)
	}
	status, integer := fields[0].Value().PortableValue().IntegerText()
	if !integer || status != wantStatus {
		t.Fatalf("HTTP status integer=%t value=%q, want %q", integer, status, wantStatus)
	}
}

func assertMarkerOnlyEvidenceRoot(t testing.TB, fixture httpfixture.Fixture) {
	t.Helper()
	entries, err := os.ReadDir(fixture.Target.Roots().EvidenceRoot())
	want := filepath.Base(fixture.Target.Roots().MarkerPath())
	if err != nil || len(entries) != 1 || entries[0].Name() != want {
		t.Fatalf("evidence root did not retire to marker-only: entries=%v want=%q err=%v", entries, want, err)
	}
}

func assertCleanHTTPWitness(
	t testing.TB,
	witness contractmodel.ClosedRunWitness,
	wantStatus string,
) {
	t.Helper()
	if !witness.Valid() || witness.ProcessClosure().State() != contractmodel.ProcessClean ||
		witness.Observation().State() != contractmodel.CaptureProjected ||
		witness.StandaloneScope().State() != contractmodel.ScopeComplete ||
		witness.Disposition() != contractmodel.DispositionEligibleClean {
		t.Fatalf("clean HTTP witness process=%s capture=%s scope=%s disposition=%s",
			witness.ProcessClosure().State(), witness.Observation().State(),
			witness.StandaloneScope().State(), witness.Disposition())
	}
	assertProjectedHTTPStatus(t, witness, wantStatus)
	checks := witness.StandaloneScope().Checks()
	wantDomains := []contractmodel.ScopeDomain{
		contractmodel.ScopeTargetInventory,
		contractmodel.ScopeChildBindings,
		contractmodel.ScopeImportResolution,
		contractmodel.ScopeServiceBindings,
		contractmodel.ScopeSentinelInheritance,
	}
	if len(checks) != len(wantDomains) {
		t.Fatalf("HTTP standalone scope has %d checks", len(checks))
	}
	for index, check := range checks {
		evidence, ok := check.Evidence()
		if check.Domain() != wantDomains[index] || check.State() != contractmodel.ScopeCheckClean ||
			!ok || !evidence.Valid() {
			t.Fatalf("HTTP scope check %d = domain=%s state=%s evidence=%t",
				index, check.Domain(), check.State(), ok)
		}
	}
}

func openRun(t testing.TB, fixture httpfixture.Fixture) store.FinalizedRunRecord {
	t.Helper()
	run, err := store.OpenFinalizedRunRecord(
		context.Background(), fixture.Target.TargetRecord(), fixture.Target.Model(),
	)
	if err != nil || !run.Valid() {
		t.Fatalf("open exact HTTP finalized run: valid=%t err=%v", run.Valid(), err)
	}
	return run
}
