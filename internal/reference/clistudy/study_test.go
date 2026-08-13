package clistudy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

func TestRunBoundedCLIWorkflowTasksUsesTheExactWorkerLimitAndJoins(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const taskCount = 6
	const workerLimit = 3
	started := make(chan int, taskCount)
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var state sync.Mutex
	active, maximumActive, completed := 0, 0, 0
	tasks := make([]cliWorkflowTask, taskCount)
	for index := range tasks {
		index := index
		tasks[index] = func(taskContext context.Context) error {
			state.Lock()
			active++
			if active > maximumActive {
				maximumActive = active
			}
			state.Unlock()
			started <- index
			select {
			case <-release:
			case <-taskContext.Done():
				return taskContext.Err()
			}
			state.Lock()
			active--
			completed++
			state.Unlock()
			return nil
		}
	}

	result := make(chan error, 1)
	go func() { result <- runBoundedCLIWorkflowTasks(ctx, workerLimit, tasks) }()
	seen := make(map[int]struct{}, workerLimit)
	for len(seen) < workerLimit {
		select {
		case index := <-started:
			seen[index] = struct{}{}
		case <-ctx.Done():
			t.Fatalf("worker pool did not reach its bound: %v", ctx.Err())
		}
	}
	state.Lock()
	if active != workerLimit || maximumActive != workerLimit {
		state.Unlock()
		t.Fatalf("worker state before release = active:%d maximum:%d, want %d", active, maximumActive, workerLimit)
	}
	state.Unlock()
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("bounded tasks returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("bounded tasks did not join: %v", ctx.Err())
	}
	state.Lock()
	defer state.Unlock()
	if active != 0 || maximumActive != workerLimit || completed != taskCount {
		t.Fatalf("terminal worker state = active:%d maximum:%d completed:%d", active, maximumActive, completed)
	}
}

func TestCLIWorkflowProductionFanoutLimitsStayExact(t *testing.T) {
	if independentCLIPhaseWorkerLimit != 7 {
		t.Fatalf("independent CLI phase worker limit = %d, want 7", independentCLIPhaseWorkerLimit)
	}
	if semanticCLIControlWorkerLimit != 16 {
		t.Fatalf("semantic CLI control worker limit = %d, want 16", semanticCLIControlWorkerLimit)
	}
	if officialCLITrialCount != 10 {
		t.Fatalf("official CLI trial count = %d, want 10", officialCLITrialCount)
	}
	if officialCLIWorkerLimit != 4 {
		t.Fatalf("official CLI worker limit = %d, want 4", officialCLIWorkerLimit)
	}
}

func TestRunBoundedCLIWorkflowTasksCancelsJoinsAndReturnsLowestNonContextError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	lowSlotErr := errors.New("low-slot-failure")
	highSlotErr := errors.New("high-slot-failure")
	lowStarted := make(chan struct{})
	highStarted := make(chan struct{})
	releaseHigh := make(chan struct{})
	var lowStartOnce, highStartOnce, releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseHigh) })
	var state sync.Mutex
	lowJoined, lateStarts := false, 0
	tasks := []cliWorkflowTask{
		func(taskContext context.Context) error {
			lowStartOnce.Do(func() { close(lowStarted) })
			<-taskContext.Done()
			state.Lock()
			lowJoined = true
			state.Unlock()
			return lowSlotErr
		},
		func(context.Context) error {
			highStartOnce.Do(func() { close(highStarted) })
			<-releaseHigh
			return highSlotErr
		},
		func(context.Context) error {
			state.Lock()
			lateStarts++
			state.Unlock()
			return nil
		},
		func(context.Context) error {
			state.Lock()
			lateStarts++
			state.Unlock()
			return nil
		},
	}

	result := make(chan error, 1)
	go func() { result <- runBoundedCLIWorkflowTasks(ctx, 2, tasks) }()
	for _, started := range []<-chan struct{}{lowStarted, highStarted} {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatalf("initial task did not start: %v", ctx.Err())
		}
	}
	releaseOnce.Do(func() { close(releaseHigh) })
	select {
	case err := <-result:
		if !errors.Is(err, lowSlotErr) {
			t.Fatalf("bounded tasks returned %v, want lowest non-context error %v", err, lowSlotErr)
		}
	case <-ctx.Done():
		t.Fatalf("bounded tasks did not join after cancellation: %v", ctx.Err())
	}
	state.Lock()
	defer state.Unlock()
	if !lowJoined || lateStarts != 0 {
		t.Fatalf("cancellation join state = low-joined:%t late-starts:%d", lowJoined, lateStarts)
	}
}

func TestPublicationReadyCLIStudyBindsIndependentDeepSnapshots(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*publicationReadyCLIStudy)
	}{
		{
			name: "deterministic-source-spec-payload",
			mutate: func(ready *publicationReadyCLIStudy) {
				ready.input.deterministic.sourceSpec["roles"].([]string)[0] = "mutated-role"
			},
		},
		{
			name: "phase-receipt",
			mutate: func(ready *publicationReadyCLIStudy) {
				ready.input.phaseTrials[0].authority = testDigest("mutated-phase-receipt")
			},
		},
		{
			name: "inspection-artifact",
			mutate: func(ready *publicationReadyCLIStudy) {
				ready.inspection.artifacts[0].canonical[0] ^= 0xff
			},
		},
		{
			name: "inspection-phase-fact",
			mutate: func(ready *publicationReadyCLIStudy) {
				ready.inspection.phaseFacts[0].kind = "mutated-kind"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ready := testPublicationReadyCLIStudy(t)
			if !ready.valid() {
				t.Fatal("synthetic publication witness is not initially valid")
			}
			test.mutate(&ready)
			if ready.valid() {
				t.Fatal("publication witness remained valid after bound content changed")
			}
		})
	}
}

func TestPublishCompletedEvidenceRefusesMutatedReadyBeforeWrites(t *testing.T) {
	node := absoluteTool(t, "node")
	git := absoluteTool(t, "git")
	working := privateDirectory(t, "working-mutated-ready")
	scratch := privateDirectory(t, "scratch-mutated-ready")
	evidence := privateDirectory(t, "evidence-mutated-ready")
	ready := testPublicationReadyCLIStudy(t)
	ready.inspection.phaseFacts[0].kind = "mutated-kind"
	values := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  evidenceDomainCLI,
		"COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence,
		"COUNTERSHAPE_NODE":          node,
		"COUNTERSHAPE_GIT":           git,
		"TMPDIR":                     scratch,
	}
	var stdout, stderr bytes.Buffer
	handlerCalled := false
	runtime := app.Runtime{
		Context: context.Background(), Stdout: &stdout, Stderr: &stderr, WorkingDirectory: working,
		LookupEnvironment: func(key string) (string, bool) {
			value, present := values[key]
			return value, present
		},
		Studies: map[string]app.StudyHandler{
			evidenceDomainCLI: func(request app.StudyRequest) error {
				handlerCalled = true
				_, err := publishCompletedEvidence(request.Context, ready, request.Workspace)
				return err
			},
		},
	}
	if exit := app.Run([]string{"study", evidenceDomainCLI, "--json"}, runtime); exit == app.ExitOK {
		t.Fatalf("mutated publication witness returned success: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	entries, err := os.ReadDir(evidence)
	if !handlerCalled || err != nil || len(entries) != 0 ||
		!strings.Contains(stdout.String(), `"status":"REFUSED"`) || stderr.Len() != 0 {
		t.Fatalf("mutated witness crossed publication boundary: called=%t entries=%v readErr=%v stdout=%q stderr=%q",
			handlerCalled, entries, err, stdout.String(), stderr.String())
	}
}

func TestRunUsesThePreparedOuterFixtureAndProducesTypedDivergence(t *testing.T) {
	root := testRepositoryRoot(t)
	node := absoluteTool(t, "node")
	git := absoluteTool(t, "git")
	fixtureParent := privateDirectory(t, "fixture-parent")
	fixtureRoot := filepath.Join(fixtureParent, "fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node,
		filepath.Join(root, "tools/run-u7-cli-study.mjs"),
		"--prepare", "--domain", evidenceDomainCLI, "--ordinal", "1", "--fixture-root", fixtureRoot,
	)
	command.Dir = root
	command.Env = []string{
		"HOME=" + fixtureParent, "TMPDIR=" + fixtureParent,
		"PATH=/usr/bin:/bin:/opt/homebrew/bin", "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
	}
	var driverStdout, driverStderr bytes.Buffer
	command.Stdout, command.Stderr = &driverStdout, &driverStderr
	if err := command.Run(); err != nil || driverStdout.Len() != 0 || driverStderr.Len() != 0 {
		t.Fatalf("prepare fixture: err=%v stdout=%q stderr=%q", err, driverStdout.String(), driverStderr.String())
	}

	result, err := Run(ctx, Config{
		RepositoryRoot: fixtureRoot,
		ScratchRoot:    privateDirectory(t, "scratch"),
		EvidenceRoot:   privateDirectory(t, "evidence"),
		GitExecutable:  git,
		NodeExecutable: node,
		Ordinal:        1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Plan.Digest().Valid() || !result.Binding.Valid() || !result.HasOutcomeMap || len(result.Trials) != 9 || len(result.CandidateBindings) != 3 {
		t.Fatalf("typed result is incomplete: plan=%t binding=%t map=%t trials=%d bindings=%d",
			result.Plan.Digest().Valid(), result.Binding.Valid(), result.HasOutcomeMap, len(result.Trials), len(result.CandidateBindings))
	}
	if _, err := compare.RequireDivergence(result.OutcomeMap); err != nil || result.OutcomeMap.DistinctProjectionCount() != 3 || len(result.OutcomeMap.Exclusions()) != 0 {
		t.Fatalf("outcome map is not the exact three-way divergence: distinct=%d exclusions=%d err=%v",
			result.OutcomeMap.DistinctProjectionCount(), len(result.OutcomeMap.Exclusions()), err)
	}
	roleProjection := make(map[reference.CLIRole]string, 3)
	worlds := make(map[string]struct{}, len(result.Trials))
	attempts := make(map[string]struct{}, len(result.Trials))
	for _, trial := range result.Trials {
		if !trial.Admitted || !trial.Projected || !trial.Result.World().Digest().Valid() ||
			!trial.Result.FinalizedAttempt().ArtifactDigest().Valid() || !trial.Projection.DerivationDigest().Valid() {
			t.Fatalf("trial is not admitted typed evidence: role=%s admitted=%t projected=%t", trial.Role, trial.Admitted, trial.Projected)
		}
		digest := sha256.Sum256(trial.Projection.ProjectionBytes())
		projection := hex.EncodeToString(digest[:])
		if prior, present := roleProjection[trial.Role]; present && prior != projection {
			t.Fatalf("role %s changed projection across repeats: %s != %s", trial.Role, prior, projection)
		}
		roleProjection[trial.Role] = projection
		worlds[trial.Result.World().Digest().String()] = struct{}{}
		attempts[trial.Result.FinalizedAttempt().ArtifactDigest().String()] = struct{}{}
	}
	if len(roleProjection) != 3 || len(worlds) != 9 || len(attempts) != 9 {
		t.Fatalf("fresh typed authority counts = roles:%d worlds:%d attempts:%d", len(roleProjection), len(worlds), len(attempts))
	}
	status := exec.CommandContext(ctx, git, "-C", fixtureRoot, "status", "--porcelain=v2", "--untracked-files=all")
	status.Env = []string{"HOME=" + fixtureParent, "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_REPLACE_OBJECTS=1"}
	if output, err := status.CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("outer fixture changed: err=%v output=%q", err, output)
	}
}

func TestEvidencePlanIsExactAndSeparatesDeterministicFromFresh(t *testing.T) {
	firstFiles, firstDeterministic, firstFresh, err := planEvidenceFiles(testEvidenceInput(1))
	if err != nil {
		t.Fatal(err)
	}
	secondFiles, secondDeterministic, secondFresh, err := planEvidenceFiles(testEvidenceInput(2))
	if err != nil {
		t.Fatal(err)
	}
	if len(firstFiles) != 111 || len(firstDeterministic) != 5 || len(firstFresh) != 8 {
		t.Fatalf("evidence counts = files:%d deterministic:%d fresh:%d", len(firstFiles), len(firstDeterministic), len(firstFresh))
	}
	for field, digest := range firstDeterministic {
		if secondDeterministic[field] != digest {
			t.Fatalf("deterministic artifact %s changed across physical runs", field)
		}
	}
	for field, digest := range firstFresh {
		if secondFresh[field] == digest {
			t.Fatalf("fresh artifact %s repeated across physical runs", field)
		}
	}
	for _, relative := range []string{
		"deterministic/source-spec.json",
		"fresh/contract-execution.json",
		"phases/search/trial-001.json",
		"phases/search/trial-080.json",
		"phases/confirm/trial-008.json",
		"phases/contract/trial-010.json",
	} {
		if len(firstFiles[relative]) == 0 {
			t.Fatalf("missing exact evidence path %s", relative)
		}
	}
	if bytes.Equal(firstFiles["fresh/world-instance.json"], secondFiles["fresh/world-instance.json"]) {
		t.Fatal("fresh wrapper bytes repeated across physical runs")
	}
}

func TestEvidencePlanRejectsForgedOrAliasedAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*evidencePublicationInput)
		code   string
	}{
		{
			name: "missing-phase-receipt",
			mutate: func(input *evidencePublicationInput) {
				input.phaseTrials = input.phaseTrials[:len(input.phaseTrials)-1]
			},
			code: "PHASE_RECEIPTS",
		},
		{
			name: "wrong-phase-order",
			mutate: func(input *evidencePublicationInput) {
				input.phaseTrials[0].phase = "confirm"
			},
			code: "PHASE_RECEIPTS",
		},
		{
			name: "duplicate-typed-authority",
			mutate: func(input *evidencePublicationInput) {
				input.phaseTrials[1].authority = input.phaseTrials[0].authority
			},
			code: "PHASE_AUTHORITY_ALIAS",
		},
		{
			name: "fresh-run-mismatch",
			mutate: func(input *evidencePublicationInput) {
				input.fresh.confirmation["physical_run_authority"] = testDigest("different-run").String()
			},
			code: "PHYSICAL_RUN_AUTHORITY",
		},
		{
			name: "deterministic-run-smuggling",
			mutate: func(input *evidencePublicationInput) {
				input.deterministic.sourceSpec["physical_run_authority"] = input.physicalRunAuthority.String()
			},
			code: "PHYSICAL_RUN_AUTHORITY",
		},
		{
			name: "payload-alias",
			mutate: func(input *evidencePublicationInput) {
				input.deterministic.worldPlan = input.deterministic.sourceSpec
			},
			code: "PAYLOAD_ALIAS",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := testEvidenceInput(1)
			test.mutate(&input)
			if _, _, _, err := planEvidenceFiles(input); err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestEvidencePublicationUsesTheAdmittedWorkspaceAndExactMachineResult(t *testing.T) {
	node := absoluteTool(t, "node")
	git := absoluteTool(t, "git")
	working := privateDirectory(t, "working")
	scratch := privateDirectory(t, "scratch")
	evidence := privateDirectory(t, "evidence")
	values := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  evidenceDomainCLI,
		"COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence,
		"COUNTERSHAPE_NODE":          node,
		"COUNTERSHAPE_GIT":           git,
		"TMPDIR":                     scratch,
	}
	var stdout, stderr bytes.Buffer
	var publication publicationProjection
	runtime := app.Runtime{
		Context:          context.Background(),
		Stdout:           &stdout,
		Stderr:           &stderr,
		WorkingDirectory: working,
		LookupEnvironment: func(key string) (string, bool) {
			value, present := values[key]
			return value, present
		},
		Studies: map[string]app.StudyHandler{
			evidenceDomainCLI: func(request app.StudyRequest) error {
				var err error
				publication, err = publishEvidenceInput(request.Context, testEvidenceInput(request.Ordinal), request.Workspace)
				return err
			},
		},
	}
	if exit := app.Run([]string{"study", evidenceDomainCLI, "--json"}, runtime); exit != app.ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	want := "{\"schema_version\":\"countershape/u7-study-domain-result/v1\",\"domain\":\"cli\",\"ordinal\":1,\"status\":\"GREEN\"}\n"
	if stdout.String() != want || stderr.Len() != 0 || !publication.valid() || len(publication.manifest) != 111 {
		t.Fatalf("stdout=%q stderr=%q publication=%+v", stdout.String(), stderr.String(), publication)
	}
	for _, entry := range publication.manifest {
		metadata, err := os.Lstat(filepath.Join(evidence, filepath.FromSlash(entry.Path)))
		if err != nil || !metadata.Mode().IsRegular() || metadata.Mode().Perm() != 0o600 || metadata.Size() != entry.Bytes {
			t.Fatalf("manifest entry %s differs: metadata=%v err=%v", entry.Path, metadata, err)
		}
	}
}

func TestEvidencePublicationRefusesCanceledContextBeforeWriting(t *testing.T) {
	node := absoluteTool(t, "node")
	git := absoluteTool(t, "git")
	working := privateDirectory(t, "working-canceled")
	scratch := privateDirectory(t, "scratch-canceled")
	evidence := privateDirectory(t, "evidence-canceled")
	values := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  evidenceDomainCLI,
		"COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence,
		"COUNTERSHAPE_NODE":          node,
		"COUNTERSHAPE_GIT":           git,
		"TMPDIR":                     scratch,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	handlerCalled := false
	runtime := app.Runtime{
		Context: ctx, Stdout: &stdout, Stderr: &stderr, WorkingDirectory: working,
		LookupEnvironment: func(key string) (string, bool) {
			value, present := values[key]
			return value, present
		},
		Studies: map[string]app.StudyHandler{
			evidenceDomainCLI: func(request app.StudyRequest) error {
				handlerCalled = true
				cancel()
				_, err := publishEvidenceInput(request.Context, testEvidenceInput(request.Ordinal), request.Workspace)
				return err
			},
		},
	}
	if exit := app.Run([]string{"study", evidenceDomainCLI, "--json"}, runtime); exit == app.ExitOK {
		t.Fatalf("canceled publication returned success: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	entries, err := os.ReadDir(evidence)
	if !handlerCalled || err != nil || len(entries) != 0 ||
		!strings.Contains(stdout.String(), `"status":"REFUSED"`) ||
		!strings.Contains(stdout.String(), `"exit_code":70`) || stderr.Len() != 0 {
		t.Fatalf("canceled publication crossed boundary: called=%t entries=%v readErr=%v stdout=%q stderr=%q",
			handlerCalled, entries, err, stdout.String(), stderr.String())
	}
}

func TestEvidenceWorkspaceRejectsAnExternalHardLinkBeforeMachineSuccess(t *testing.T) {
	node := absoluteTool(t, "node")
	git := absoluteTool(t, "git")
	working := privateDirectory(t, "working")
	scratch := privateDirectory(t, "scratch")
	evidence := privateDirectory(t, "evidence")
	values := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN": evidenceDomainCLI, "COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence, "COUNTERSHAPE_NODE": node, "COUNTERSHAPE_GIT": git, "TMPDIR": scratch,
	}
	var stdout bytes.Buffer
	payload := []byte("{\"probe\":true}\n")
	runtime := app.Runtime{
		Context: context.Background(), Stdout: &stdout, WorkingDirectory: working,
		LookupEnvironment: func(key string) (string, bool) {
			value, present := values[key]
			return value, present
		},
		Studies: map[string]app.StudyHandler{
			evidenceDomainCLI: func(request app.StudyRequest) error {
				if err := request.Workspace.CreateDirectory("deterministic"); err != nil {
					return err
				}
				if err := request.Workspace.WriteFileExclusive("deterministic/probe.json", payload); err != nil {
					return err
				}
				if err := os.Link(filepath.Join(evidence, "deterministic/probe.json"), filepath.Join(scratch, "outside-alias")); err != nil {
					return err
				}
				digest := sha256.Sum256(payload)
				return request.Workspace.VerifyExactManifest(app.EvidenceManifest{
					Directories: []string{"deterministic"},
					Files: []app.EvidenceFile{{
						Path: "deterministic/probe.json", Bytes: int64(len(payload)), SHA256: hex.EncodeToString(digest[:]),
					}},
				})
			},
		},
	}
	if exit := app.Run([]string{"study", evidenceDomainCLI, "--json"}, runtime); exit == app.ExitOK || strings.Contains(stdout.String(), `"status":"GREEN"`) {
		t.Fatalf("hard-linked evidence emitted success: exit=%d stdout=%q", exit, stdout.String())
	}
}

func testEvidenceInput(ordinal int) evidencePublicationInput {
	physical := testDigest("physical-run", fmt.Sprintf("%d", ordinal))
	deterministic := func(name string) evidencePayload {
		return evidencePayload{"authority": "deterministic-" + name, "digest": testDigest("deterministic", name).String()}
	}
	fresh := func(name string) evidencePayload {
		return evidencePayload{
			"authority":              "fresh-" + name,
			"identity":               testDigest("fresh", name, fmt.Sprintf("%d", ordinal)).String(),
			"physical_run_authority": physical.String(),
		}
	}
	return evidencePublicationInput{
		ordinal: ordinal, physicalRunAuthority: physical, phaseTrials: testPhaseReceipts(),
		deterministic: deterministicEvidencePayloads{
			sourceSpec: deterministic("source"), worldPlan: deterministic("plan"), ruling: deterministic("ruling"),
			decisionRecord: deterministic("decision"), contractBundle: deterministic("bundle"),
		},
		fresh: freshEvidencePayloads{
			worldInstance: fresh("world"), attempts: fresh("attempts"), measurements: fresh("measurements"),
			captures: fresh("captures"), confirmation: fresh("confirmation"), target: fresh("target"),
			finalized: fresh("finalized"), classification: fresh("classification"),
		},
	}
}

func testPublicationReadyCLIStudy(t *testing.T) publicationReadyCLIStudy {
	t.Helper()
	input := testEvidenceInput(1)
	input.deterministic.sourceSpec["roles"] = []string{"first", "second"}
	boundInput, err := cloneEvidencePublicationInput(input)
	if err != nil {
		t.Fatal(err)
	}
	inspection := testStudyInspection(1)
	if !inspection.Valid() || !inspection.publication.empty() {
		t.Fatal("synthetic inspection is not valid and unpublished")
	}
	return publicationReadyCLIStudy{
		ordinal: 1, physicalRunAuthority: input.physicalRunAuthority,
		inspection:      cloneStudyInspection(inspection),
		boundInspection: cloneStudyInspection(inspection),
		input:           input,
		boundInput:      boundInput,
		seal:            issuedPublicationReadyCLIStudy,
	}
}

func testStudyInspection(ordinal int) StudyInspection {
	physical := testDigest("physical-run", fmt.Sprintf("%d", ordinal))
	ruling := testDigest("inspection-artifact", "ruling")
	inspection := StudyInspection{
		ordinal: ordinal, physicalRunAuthority: physical,
		artifacts: []ArtifactFact{
			{name: "source-spec", authority: testDigest("inspection-artifact", "source-spec"), canonical: []byte("source-spec\n")},
			{name: "world-plan", authority: testDigest("inspection-artifact", "world-plan"), canonical: []byte("world-plan\n")},
			{name: "ruling", authority: ruling, canonical: []byte("ruling\n")},
			{name: "decision-record", authority: ruling, canonical: []byte("ruling\n")},
			{name: "contract-bundle", authority: testDigest("inspection-artifact", "contract-bundle"), canonical: []byte("contract-bundle\n")},
		},
		nonclaims: append([]string(nil), expectedInspectionNonclaims...),
		seal:      issuedStudyInspection,
	}
	receipts := testPhaseReceipts()
	inspection.phaseFacts = make([]PhaseFact, len(receipts))
	for index, receipt := range receipts {
		inspection.phaseFacts[index] = PhaseFact{
			phase: receipt.phase, trial: receipt.trial, kind: receipt.phase + "-trial", owner: receipt.authority,
		}
	}
	inspection.officialFacts = make([]OfficialFact, 10)
	for index := range inspection.officialFacts {
		trial := index + 1
		inspection.officialFacts[index] = OfficialFact{
			trial:                         trial,
			attempt:                       testDigest("official", "attempt", fmt.Sprintf("%d", trial)),
			target:                        testDigest("official", "target", fmt.Sprintf("%d", trial)),
			targetCanonicalSHA256:         testSHA256("official", "target-canonical", fmt.Sprintf("%d", trial)),
			bundle:                        testDigest("official", "bundle", fmt.Sprintf("%d", trial)),
			residueHead:                   testDigest("official", "residue", fmt.Sprintf("%d", trial)),
			run:                           testDigest("official", "run", fmt.Sprintf("%d", trial)),
			runCanonicalSHA256:            testSHA256("official", "run-canonical", fmt.Sprintf("%d", trial)),
			classification:                inspection.phaseFacts[88+index].owner,
			classificationCanonicalSHA256: testSHA256("official", "classification-canonical", fmt.Sprintf("%d", trial)),
			result:                        "CONFORMS", exitCode: 0, recoveryEqual: true,
		}
	}
	minimized := testDigest("reduction", "minimized")
	inspection.reductionFacts = []ReductionFact{
		{
			name: "main-weak", run: testDigest("reduction", "weak-run"),
			transcript: testDigest("reduction", "weak-transcript"), grade: testDigest("reduction", "weak-grade"),
			status: "BEST_KNOWN", reducerSet: testDigest("reduction", "weak-reducer-set"),
			minimizedStimulus: minimized, finalSweepState: "INCOMPLETE", gradeLimitations: []string{"test limitation"},
		},
		{
			name: "auxiliary-strong", run: testDigest("reduction", "strong-run"),
			transcript: testDigest("reduction", "strong-transcript"), grade: testDigest("reduction", "strong-grade"),
			status: "ONE_MINIMAL_UNDER", reducerSet: testDigest("reduction", "strong-reducer-set"),
			minimizedStimulus: minimized, finalSweepState: "COMPLETE", completedSweep: testDigest("reduction", "strong-sweep"),
		},
	}
	return inspection
}

func testSHA256(parts ...string) string {
	return strings.TrimPrefix(testDigest(parts...).String(), "sha256:")
}

func testPhaseReceipts() []phaseEvidenceReceipt {
	receipts := make([]phaseEvidenceReceipt, 0, 98)
	for _, phase := range trialPhaseCounts {
		for trial := 1; trial <= phase.count; trial++ {
			receipts = append(receipts, phaseEvidenceReceipt{
				phase: phase.phase, trial: trial,
				authority: testDigest("phase", phase.phase, fmt.Sprintf("%d", trial)),
			})
		}
	}
	return receipts
}

func testDigest(parts ...string) domain.Digest {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	digest, err := domain.ParseDigest("sha256:" + hex.EncodeToString(sum[:]))
	if err != nil {
		panic(err)
	}
	return digest
}

func privateDirectory(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("private directory %q is not canonical: %q %v", path, resolved, err)
	}
	return resolved
}

func absoluteTool(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is unavailable: %v", name, err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(working, "..", "..", ".."))
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root is invalid: %v", err)
	}
	return root
}
