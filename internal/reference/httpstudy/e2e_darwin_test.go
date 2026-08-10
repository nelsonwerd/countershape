package httpstudy

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/reference/app"
)

const (
	e2eNode                 = "/opt/homebrew/bin/node"
	e2eGo                   = "/opt/homebrew/bin/go"
	e2eGit                  = "/usr/bin/git"
	e2eBuildTimeout         = 2 * time.Minute
	e2eStudyTimeout         = 6 * time.Minute
	e2ePhysicalSliceTimeout = 3 * time.Minute
)

func TestHTTPAppMachineRouteRunsDriverFixtureAndPublishesClosedEvidence(t *testing.T) {
	repositoryRoot := e2eRepositoryRoot(t)
	fixtureRoot := prepareE2EHTTPFixture(t, repositoryRoot, 1)
	scratchRoot := privateE2EDirectory(t, "scratch")
	evidenceRoot := privateE2EDirectory(t, "evidence")
	headBefore := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD")
	treeBefore := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD^{tree}")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	binary := buildE2ECountershape(t, repositoryRoot)
	exitCode := runE2ECompiledStudy(t, binary, fixtureRoot, scratchRoot, evidenceRoot, 1, &stdout, &stderr)
	if exitCode != app.ExitOK || stderr.Len() != 0 {
		t.Fatalf("machine route exit=%d stderr=%q stdout=%q", exitCode, stderr.Bytes(), stdout.Bytes())
	}
	wantStdout := "{\"schema_version\":\"countershape/u7-study-domain-result/v1\",\"domain\":\"http\",\"ordinal\":1,\"status\":\"GREEN\"}\n"
	if stdout.String() != wantStdout {
		t.Fatalf("machine stdout = %q, want %q", stdout.String(), wantStdout)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result) != 4 {
		t.Fatalf("four-field machine result: keys=%d err=%v", len(result), err)
	}

	assertE2EEvidenceClosure(t, evidenceRoot, 1)
	if status := e2eGitBytes(t, fixtureRoot, "status", "--porcelain=v2", "-z", "--untracked-files=all"); len(status) != 0 {
		t.Fatalf("fixture became dirty: %q", status)
	}
	if headAfter, treeAfter := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD"), e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD^{tree}"); headAfter != headBefore || treeAfter != treeBefore {
		t.Fatalf("fixture authority moved: head=%s/%s tree=%s/%s", headBefore, headAfter, treeBefore, treeAfter)
	}
}

// TestHTTPPhysicalDiscoveryRacePath keeps one real HTTP world/observe/compare
// slice inside the outer test process. Under event 5 that process is built
// with -race; the complete 80/8/10 workflow above intentionally runs in the
// separately verified non-race product child so it stays below the frozen
// fifteen-minute package alarm.
func TestHTTPPhysicalDiscoveryRacePath(t *testing.T) {
	repositoryRoot := e2eRepositoryRoot(t)
	fixtureRoot := prepareE2EHTTPFixture(t, repositoryRoot, 2)
	scratchRoot := privateE2EDirectory(t, "race-scratch")
	evidenceRoot := privateE2EDirectory(t, "race-evidence")
	headBefore := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD")
	treeBefore := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD^{tree}")

	ctx, cancel := context.WithTimeout(context.Background(), e2ePhysicalSliceTimeout)
	defer cancel()
	result, err := runPhase(ctx, Config{
		RepositoryRoot: fixtureRoot,
		ScratchRoot:    scratchRoot,
		EvidenceRoot:   evidenceRoot,
		GitExecutable:  resolvedE2EExecutable(t, e2eGit),
		NodeExecutable: resolvedE2EExecutable(t, e2eNode),
		Ordinal:        2,
	}, phaseRunSpec{
		label: "race-physical-slice", purpose: domain.AttemptDiscovery, repetitions: 2,
		planDiscoveryRepeats: 2, planConfirmationRepeats: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Observation.Status() != observe.ObservationComplete || result.Observation.CompletedMatrices() != 2 ||
		!result.HasOutcomeMap || len(result.Trials) != 8 || len(result.OutcomeMap.Entries()) != 3 ||
		len(result.OutcomeMap.Exclusions()) != 1 || !result.OutcomeMap.Divergence() ||
		result.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatalf("race physical slice is incomplete: trials=%d matrices=%d entries=%d exclusions=%d",
			len(result.Trials), result.Observation.CompletedMatrices(), len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()))
	}
	worlds := make(map[domain.Digest]struct{}, len(result.Trials))
	attempts := make(map[domain.Digest]struct{}, len(result.Trials))
	for _, trial := range result.Trials {
		world := trial.Result.World().Digest()
		attempt := trial.Result.FinalizedAttempt().ArtifactDigest()
		if !trial.Admitted || !trial.Projected || !world.Valid() || !attempt.Valid() {
			t.Fatal("race physical slice contains an inadmissible trial")
		}
		if _, duplicate := worlds[world]; duplicate {
			t.Fatal("race physical slice reused a world")
		}
		if _, duplicate := attempts[attempt]; duplicate {
			t.Fatal("race physical slice reused an attempt")
		}
		worlds[world] = struct{}{}
		attempts[attempt] = struct{}{}
	}
	if status := e2eGitBytes(t, fixtureRoot, "status", "--porcelain=v2", "-z", "--untracked-files=all"); len(status) != 0 {
		t.Fatalf("race fixture became dirty: %q", status)
	}
	if headAfter, treeAfter := e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD"), e2eGitLine(t, fixtureRoot, "rev-parse", "HEAD^{tree}"); headAfter != headBefore || treeAfter != treeBefore {
		t.Fatalf("race fixture authority moved: head=%s/%s tree=%s/%s", headBefore, headAfter, treeBefore, treeAfter)
	}
}

func TestHTTPAppConcurrentMachineRoutesShareNoMutableState(t *testing.T) {
	type route struct {
		working  string
		scratch  string
		evidence string
		node     string
		git      string
	}
	routes := make([]route, 2)
	for index := range routes {
		routes[index] = route{
			working:  privateE2EDirectory(t, fmt.Sprintf("concurrent-working-%d", index)),
			scratch:  privateE2EDirectory(t, fmt.Sprintf("concurrent-scratch-%d", index)),
			evidence: privateE2EDirectory(t, fmt.Sprintf("concurrent-evidence-%d", index)),
			node:     privateE2EExecutable(t, fmt.Sprintf("concurrent-node-%d", index)),
			git:      privateE2EExecutable(t, fmt.Sprintf("concurrent-git-%d", index)),
		}
	}
	type outcome struct {
		stdout string
		exit   int
	}
	outcomes := make([]outcome, len(routes))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range routes {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			route := routes[index]
			stdout, exitCode := runE2EProbeStudy(route.working, route.scratch, route.evidence, route.node, route.git, func(request app.StudyRequest) error {
				return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
			})
			outcomes[index] = outcome{stdout: stdout, exit: exitCode}
		}()
	}
	close(start)
	wait.Wait()

	want := "{\"schema_version\":\"countershape/u7-study-domain-result/v1\",\"domain\":\"http\",\"ordinal\":1,\"status\":\"GREEN\"}\n"
	for index, outcome := range outcomes {
		if outcome.exit != app.ExitOK || outcome.stdout != want {
			t.Fatalf("concurrent route %d exit=%d stdout=%q", index, outcome.exit, outcome.stdout)
		}
		exact, err := os.ReadFile(filepath.Join(routes[index].evidence, "probe.json"))
		if err != nil || string(exact) != "{}\n" {
			t.Fatalf("concurrent route %d evidence=%q err=%v", index, exact, err)
		}
	}
}

func TestHTTPAppDirectNodeBoundaryRevokesEscapedRequest(t *testing.T) {
	scratch := privateE2EDirectory(t, "direct-node-scratch")
	working := privateE2ESubdirectory(t, scratch, "working")
	home := privateE2ESubdirectory(t, scratch, "home")
	temporary := privateE2ESubdirectory(t, scratch, "tmp")
	bundle := privateE2ESubdirectory(t, scratch, "bundle")
	testFile := filepath.Join(bundle, "contract.test.mjs")
	if err := os.WriteFile(testFile, []byte("// exact fake contract entrypoint\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(testFile, 0o644); err != nil {
		t.Fatal(err)
	}
	evidence := privateE2EDirectory(t, "direct-node-evidence")
	node := privateE2EScript(t, "direct-node", "#!/bin/sh\nprintf 'TAP version 13\\n# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\\n1..1\\n# pass 1\\n# fail 0\\n'\n")
	git := privateE2EExecutable(t, "direct-git")
	input := app.StudyNodeTestInput{
		WorkingDirectory: working, TestFile: testFile, HomeDirectory: home,
		TemporaryDirectory: temporary, Timeout: 5 * time.Second,
	}
	var escaped app.StudyRequest
	stdout, exitCode := runE2EProbeStudy(working, scratch, evidence, node, git, func(request app.StudyRequest) error {
		escaped = request
		observation, err := request.RunNodeTest(request.Context, input)
		if err != nil {
			return err
		}
		if observation.ExitCode != 0 || len(observation.Stderr) != 0 ||
			!bytes.Contains(observation.Stdout, []byte("# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\n")) ||
			!validE2ERawDigest(observation.InvocationSHA256) || !validE2ERawDigest(observation.TestFileSHA256) {
			return fmt.Errorf("direct Node observation is incomplete")
		}
		return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
	})
	want := "{\"schema_version\":\"countershape/u7-study-domain-result/v1\",\"domain\":\"http\",\"ordinal\":1,\"status\":\"GREEN\"}\n"
	if exitCode != app.ExitOK || stdout != want {
		t.Fatalf("direct Node route exit=%d stdout=%q", exitCode, stdout)
	}
	if _, err := escaped.RunNodeTest(context.Background(), input); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("escaped request retained Node authority: %v", err)
	}
	for _, root := range []string{home, temporary} {
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("direct Node private root %s has residue: entries=%d err=%v", root, len(entries), err)
		}
	}
}

func TestHTTPAppDirectNodeCancellationPoisonsSwallowedFailure(t *testing.T) {
	scratch := privateE2EDirectory(t, "cancel-node-scratch")
	working := privateE2ESubdirectory(t, scratch, "working")
	home := privateE2ESubdirectory(t, scratch, "home")
	temporary := privateE2ESubdirectory(t, scratch, "tmp")
	bundle := privateE2ESubdirectory(t, scratch, "bundle")
	testFile := filepath.Join(bundle, "contract.test.mjs")
	if err := os.WriteFile(testFile, []byte("// cancellation probe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(testFile, 0o644); err != nil {
		t.Fatal(err)
	}
	evidence := privateE2EDirectory(t, "cancel-node-evidence")
	node := privateE2EScript(t, "cancel-node", "#!/bin/sh\nsleep 5\nprintf 'TAP version 13\\n# COUNTERSHAPE_RESULT_V1|CONFORMS|NONE\\n1..1\\n# pass 1\\n# fail 0\\n'\n")
	git := privateE2EExecutable(t, "cancel-git")
	stdout, exitCode := runE2EProbeStudy(working, scratch, evidence, node, git, func(request app.StudyRequest) error {
		runContext, cancel := context.WithCancel(request.Context)
		timer := time.AfterFunc(100*time.Millisecond, cancel)
		_, runErr := request.RunNodeTest(runContext, app.StudyNodeTestInput{
			WorkingDirectory: working, TestFile: testFile, HomeDirectory: home,
			TemporaryDirectory: temporary, Timeout: 5 * time.Second,
		})
		timer.Stop()
		cancel()
		if runErr == nil {
			return fmt.Errorf("canceled direct Node test returned success")
		}
		if err := request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n")); err != nil {
			return err
		}
		return nil // Deliberately swallow runErr; runner poison must still fail the route.
	})
	if exitCode == app.ExitOK || strings.Contains(stdout, `"status":"GREEN"`) {
		t.Fatalf("canceled direct Node route false-greened: exit=%d stdout=%q", exitCode, stdout)
	}
}

func TestHTTPAppHumanAndInvalidEnvironmentRoutesNeverInvokeStudy(t *testing.T) {
	workingRoot := privateE2EDirectory(t, "working")
	scratchRoot := privateE2EDirectory(t, "scratch")
	evidenceRoot := privateE2EDirectory(t, "evidence")
	invocations := 0
	probe := func(app.StudyRequest) error {
		invocations++
		return fmt.Errorf("probe handler must not execute")
	}
	baseEnvironment := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  "http",
		"COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidenceRoot,
		"COUNTERSHAPE_NODE":          e2eNode,
		"COUNTERSHAPE_GIT":           e2eGit,
		"TMPDIR":                     scratchRoot,
	}

	t.Run("human route", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := app.Run([]string{"study", "http"}, app.Runtime{
			Stdout: &stdout, Stderr: &stderr, WorkingDirectory: workingRoot,
			Studies: map[string]app.StudyHandler{"http": probe},
			LookupEnvironment: func(key string) (string, bool) {
				value, present := baseEnvironment[key]
				return value, present
			},
		})
		if exitCode != app.ExitRefused || stdout.Len() != 0 || !strings.Contains(stderr.String(), "STUDY_HARNESS_ONLY") {
			t.Fatalf("human route exit=%d stdout=%q stderr=%q", exitCode, stdout.Bytes(), stderr.Bytes())
		}
	})

	t.Run("invalid environment", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := app.Run([]string{"study", "http", "--json"}, app.Runtime{
			Stdout: &stdout, Stderr: &stderr, WorkingDirectory: workingRoot,
			Studies: map[string]app.StudyHandler{"http": probe},
			LookupEnvironment: func(key string) (string, bool) {
				if key == "COUNTERSHAPE_STUDY_DOMAIN" {
					return "other", true
				}
				value, present := baseEnvironment[key]
				return value, present
			},
		})
		if exitCode != app.ExitRefused || stderr.Len() != 0 || !strings.Contains(stdout.String(), "STUDY_ENVIRONMENT_INVALID") {
			t.Fatalf("invalid environment exit=%d stdout=%q stderr=%q", exitCode, stdout.Bytes(), stderr.Bytes())
		}
	})

	if invocations != 0 {
		t.Fatalf("refused routes invoked the study %d times", invocations)
	}
	entries, err := os.ReadDir(evidenceRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("refused routes changed evidence root: entries=%d err=%v", len(entries), err)
	}
}

func TestHTTPAppRejectsSpecialModesAndTerminalModeDrift(t *testing.T) {
	for _, test := range []struct {
		name       string
		mutate     func(working, scratch, evidence, node, git string) error
		wantCode   string
		invokeWant int
	}{
		{
			name: "working root special bits",
			mutate: func(working, _, _, _, _ string) error {
				return os.Chmod(working, 0o700|os.ModeSticky)
			},
			wantCode: "STUDY_WORKSPACE_INVALID",
		},
		{
			name: "tool target special bits",
			mutate: func(_, _, _, node, _ string) error {
				return os.Chmod(node, 0o700|os.ModeSetuid)
			},
			wantCode: "STUDY_RUNTIME_INVALID",
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			working := privateE2EDirectory(t, "working")
			scratch := privateE2EDirectory(t, "scratch")
			evidence := privateE2EDirectory(t, "evidence")
			node := privateE2EExecutable(t, "node")
			git := privateE2EExecutable(t, "git")
			if err := test.mutate(working, scratch, evidence, node, git); err != nil {
				t.Fatal(err)
			}
			invocations := 0
			stdout, exitCode := runE2EProbeStudy(working, scratch, evidence, node, git, func(app.StudyRequest) error {
				invocations++
				return nil
			})
			if exitCode != app.ExitRefused || invocations != test.invokeWant || !strings.Contains(stdout, test.wantCode) {
				t.Fatalf("special-mode admission exit=%d invocations=%d stdout=%q", exitCode, invocations, stdout)
			}
		})
	}

	t.Run("working root terminal drift", func(t *testing.T) {
		working := privateE2EDirectory(t, "working")
		scratch := privateE2EDirectory(t, "scratch")
		evidence := privateE2EDirectory(t, "evidence")
		node := privateE2EExecutable(t, "node")
		git := privateE2EExecutable(t, "git")
		stdout, exitCode := runE2EProbeStudy(working, scratch, evidence, node, git, func(request app.StudyRequest) error {
			if err := request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n")); err != nil {
				return err
			}
			return os.Chmod(working, 0o700|os.ModeSticky)
		})
		if exitCode != app.ExitRefused || !strings.Contains(stdout, "STUDY_WORKSPACE_CHANGED") {
			t.Fatalf("terminal mode drift exit=%d stdout=%q", exitCode, stdout)
		}
	})
}

func runE2EProbeStudy(working, scratch, evidence, node, git string, handler app.StudyHandler) (string, int) {
	environment := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN": "http", "COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence, "COUNTERSHAPE_NODE": node,
		"COUNTERSHAPE_GIT": git, "TMPDIR": scratch,
	}
	var stdout bytes.Buffer
	exitCode := app.Run([]string{"study", "http", "--json"}, app.Runtime{
		Stdout: &stdout, Stderr: io.Discard, WorkingDirectory: working,
		Studies: map[string]app.StudyHandler{"http": handler},
		LookupEnvironment: func(key string) (string, bool) {
			value, present := environment[key]
			return value, present
		},
	})
	return stdout.String(), exitCode
}

func privateE2EExecutable(t *testing.T, name string) string {
	t.Helper()
	return privateE2EScript(t, name, "#!/bin/sh\nexit 0\n")
}

func privateE2EScript(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(canonicalE2ETemp(t), name)
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func privateE2ESubdirectory(t *testing.T, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if filepath.Dir(path) != parent {
		t.Fatal("private E2E subdirectory escaped parent")
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

type e2eEvidenceExpectation struct {
	domain        string
	ordinal       int
	artifact      string
	deterministic bool
	phase         string
	trial         int
}

func assertE2EEvidenceClosure(t *testing.T, root string, ordinal int) {
	t.Helper()
	expectedFiles := expectedE2EEvidenceFiles(ordinal)
	expectedDirectories := map[string]bool{
		".": true, "deterministic": true, "fresh": true, "phases": true,
		"phases/search": true, "phases/confirm": true, "phases/contract": true,
	}
	seenFiles := make(map[string]bool, len(expectedFiles))
	seenDirectories := make(map[string]bool, len(expectedDirectories))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link %s", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if !expectedDirectories[relative] || info.Mode().Perm() != 0o700 {
				return fmt.Errorf("unexpected or non-private directory %s mode=%v", relative, info.Mode())
			}
			seenDirectories[relative] = true
			return nil
		}
		expectation, present := expectedFiles[relative]
		if !present || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() < 1 || info.Size() > maximumEvidenceBytes {
			return fmt.Errorf("unexpected or non-private evidence file %s mode=%v bytes=%d", relative, info.Mode(), info.Size())
		}
		if metadata, ok := info.Sys().(*syscall.Stat_t); !ok || metadata.Nlink != 1 {
			return fmt.Errorf("evidence file %s is not singly linked", relative)
		}
		exact, err := os.ReadFile(path)
		if err != nil || exact[len(exact)-1] != '\n' || !json.Valid(exact) {
			return fmt.Errorf("evidence file %s is not one JSON line: %v", relative, err)
		}
		if err := validateE2EEvidenceLine(exact, expectation); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		seenFiles[relative] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seenFiles) != 111 || len(seenFiles) != len(expectedFiles) || len(seenDirectories) != len(expectedDirectories) {
		t.Fatalf("evidence closure files=%d/%d directories=%d/%d", len(seenFiles), len(expectedFiles), len(seenDirectories), len(expectedDirectories))
	}
	assertE2ESemanticPayloadClosure(t, root)
}

func expectedE2EEvidenceFiles(ordinal int) map[string]e2eEvidenceExpectation {
	result := make(map[string]e2eEvidenceExpectation, 111)
	for _, artifact := range []string{"source_spec_sha256", "world_plan_sha256", "ruling_sha256", "decision_record_sha256", "contract_bundle_sha256"} {
		filename := strings.ReplaceAll(strings.TrimSuffix(artifact, "_sha256"), "_", "-") + ".json"
		result["deterministic/"+filename] = e2eEvidenceExpectation{
			domain: "http", artifact: artifact, deterministic: true,
		}
	}
	for _, artifact := range []string{"world_instance_sha256", "attempts_sha256", "measurements_sha256", "captures_sha256", "confirmation_sha256", "contract_execution_target_sha256", "finalized_contract_run_sha256", "contract_execution_sha256"} {
		filename := strings.ReplaceAll(strings.TrimSuffix(artifact, "_sha256"), "_", "-") + ".json"
		result["fresh/"+filename] = e2eEvidenceExpectation{
			domain: "http", ordinal: ordinal, artifact: artifact,
		}
	}
	for _, phase := range []struct {
		name  string
		count int
	}{{"search", 80}, {"confirm", 8}, {"contract", 10}} {
		for trial := 1; trial <= phase.count; trial++ {
			path := fmt.Sprintf("phases/%s/trial-%03d.json", phase.name, trial)
			result[path] = e2eEvidenceExpectation{domain: "http", ordinal: ordinal, phase: phase.name, trial: trial}
		}
	}
	return result
}

func validateE2EEvidenceLine(exact []byte, expected e2eEvidenceExpectation) error {
	if expected.phase != "" {
		var line struct {
			SchemaVersion string `json:"schema_version"`
			Domain        string `json:"domain"`
			Ordinal       int    `json:"ordinal"`
			Phase         string `json:"phase"`
			Trial         int    `json:"trial"`
			Status        string `json:"status"`
		}
		if err := json.Unmarshal(exact, &line); err != nil || line.SchemaVersion != evidenceTrialSchema || line.Domain != expected.domain ||
			line.Ordinal != expected.ordinal || line.Phase != expected.phase || line.Trial != expected.trial || line.Status != evidenceStatusGreen {
			return fmt.Errorf("trial wrapper mismatch: %+v err=%v", line, err)
		}
		return nil
	}
	if expected.deterministic {
		var line deterministicArtifactLine
		if err := json.Unmarshal(exact, &line); err != nil || line.SchemaVersion != evidenceArtifactSchema || line.Domain != expected.domain ||
			line.Artifact != expected.artifact || len(line.Payload) == 0 {
			return fmt.Errorf("deterministic wrapper mismatch: %+v err=%v", line, err)
		}
		return nil
	}
	var line freshArtifactLine
	if err := json.Unmarshal(exact, &line); err != nil || line.SchemaVersion != evidenceArtifactSchema || line.Domain != expected.domain ||
		line.Ordinal != expected.ordinal || line.Artifact != expected.artifact || len(line.Payload) == 0 {
		return fmt.Errorf("fresh wrapper mismatch: %+v err=%v", line, err)
	}
	return nil
}

func assertE2ESemanticPayloadClosure(t *testing.T, root string) {
	t.Helper()
	payloads := make(map[string]map[string]any, 13)
	for relative := range expectedE2EEvidenceFiles(1) {
		if !strings.HasPrefix(relative, "deterministic/") && !strings.HasPrefix(relative, "fresh/") {
			continue
		}
		exact, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		var wrapper struct {
			Payload map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(exact, &wrapper); err != nil || len(wrapper.Payload) == 0 {
			t.Fatalf("semantic payload %s: keys=%d err=%v", relative, len(wrapper.Payload), err)
		}
		payloads[relative] = wrapper.Payload
	}

	requireKeys := func(relative string, keys ...string) map[string]any {
		t.Helper()
		payload := payloads[relative]
		observed := make([]string, 0, len(payload))
		for key := range payload {
			observed = append(observed, key)
		}
		sort.Strings(observed)
		sort.Strings(keys)
		if !slicesEqual(observed, keys) {
			t.Fatalf("semantic payload keys %s = %v, want %v", relative, observed, keys)
		}
		return payload
	}
	requireString := func(relative string, payload map[string]any, key string) string {
		t.Helper()
		value, ok := payload[key].(string)
		if !ok || value == "" {
			t.Fatalf("semantic payload %s field %s = %#v", relative, key, payload[key])
		}
		return value
	}
	requireStrings := func(relative string, payload map[string]any, key string, count int, unique bool) []string {
		t.Helper()
		values, ok := payload[key].([]any)
		if !ok || len(values) != count {
			t.Fatalf("semantic payload %s field %s count = %d/%d", relative, key, len(values), count)
		}
		result := make([]string, len(values))
		seen := make(map[string]struct{}, len(values))
		for index, item := range values {
			value, ok := item.(string)
			if !ok || value == "" {
				t.Fatalf("semantic payload %s field %s[%d] = %#v", relative, key, index, item)
			}
			if unique {
				if _, duplicate := seen[value]; duplicate {
					t.Fatalf("semantic payload %s field %s repeats %s", relative, key, value)
				}
				seen[value] = struct{}{}
			}
			result[index] = value
		}
		return result
	}
	requireTypedDigests := func(relative string, payload map[string]any, key string, count int, unique bool) []string {
		t.Helper()
		values := requireStrings(relative, payload, key, count, unique)
		for _, value := range values {
			if !validE2ETypedDigest(value) {
				t.Fatalf("semantic payload %s field %s has invalid digest %q", relative, key, value)
			}
		}
		return values
	}
	requireRawDigests := func(relative string, payload map[string]any, key string, count int, unique bool) []string {
		t.Helper()
		values := requireStrings(relative, payload, key, count, unique)
		for _, value := range values {
			if !validE2ERawDigest(value) {
				t.Fatalf("semantic payload %s field %s has invalid digest %q", relative, key, value)
			}
		}
		return values
	}
	requireNumbers := func(relative string, payload map[string]any, key string, count int, want float64) {
		t.Helper()
		values, ok := payload[key].([]any)
		if !ok || len(values) != count {
			t.Fatalf("semantic payload %s field %s count = %d/%d", relative, key, len(values), count)
		}
		for index, value := range values {
			if value != want {
				t.Fatalf("semantic payload %s field %s[%d] = %#v, want %v", relative, key, index, value, want)
			}
		}
	}
	requirePositiveNumbers := func(relative string, payload map[string]any, key string, count int) {
		t.Helper()
		values, ok := payload[key].([]any)
		if !ok || len(values) != count {
			t.Fatalf("semantic payload %s field %s count = %d/%d", relative, key, len(values), count)
		}
		for index, value := range values {
			number, ok := value.(float64)
			if !ok || number <= 0 || math.Trunc(number) != number {
				t.Fatalf("semantic payload %s field %s[%d] = %#v", relative, key, index, value)
			}
		}
	}

	source := requireKeys("deterministic/source-spec.json", "authority", "canonical_sha256", "digest")
	if requireString("deterministic/source-spec.json", source, "authority") != "STRICT_SOURCE_SPEC_COMPILED_BY_SEALED_CORE" ||
		!validE2ERawDigest(requireString("deterministic/source-spec.json", source, "canonical_sha256")) ||
		!validE2ETypedDigest(requireString("deterministic/source-spec.json", source, "digest")) {
		t.Fatal("source-spec semantic authority is malformed")
	}
	plan := requireKeys("deterministic/world-plan.json", "authority", "canonical_sha256", "digest")
	if requireString("deterministic/world-plan.json", plan, "authority") != "WORLD_PLAN_COMPILED_FROM_STRICT_SOURCE" ||
		!validE2ERawDigest(requireString("deterministic/world-plan.json", plan, "canonical_sha256")) ||
		!validE2ETypedDigest(requireString("deterministic/world-plan.json", plan, "digest")) {
		t.Fatal("world-plan semantic authority is malformed")
	}
	ruling := requireKeys("deterministic/ruling.json", "action", "allowed_status", "authority", "predicate_sha256", "projection_kind", "selected_fields")
	decision := requireKeys("deterministic/decision-record.json", "authority", "canonical_record_valid", "early_reveal", "projection_kind", "semantic_sha256", "selected_fields")
	bundle := requireKeys("deterministic/contract-bundle.json", "authority", "canonical_bundle_valid", "member_count", "predicate_sha256", "projection_kind", "selected_fields", "semantic_sha256")
	if ruling["action"] != "ALLOW_OBSERVED" || ruling["allowed_status"] != float64(404) ||
		ruling["authority"] != "DETERMINISTIC_STATUS_ONLY_RULING_PROJECTION_FROM_SEALED_DECISION" || !validE2ERawDigest(requireString("deterministic/ruling.json", ruling, "predicate_sha256")) ||
		decision["canonical_record_valid"] != true || decision["early_reveal"] != false || decision["authority"] != "DETERMINISTIC_DECISION_SEMANTIC_PROJECTION_FROM_STRICT_RECORD" ||
		!validE2ERawDigest(requireString("deterministic/decision-record.json", decision, "semantic_sha256")) ||
		bundle["canonical_bundle_valid"] != true || bundle["member_count"] != float64(6) || bundle["authority"] != "DETERMINISTIC_CONTRACT_SEMANTIC_PROJECTION_FROM_STRICT_SIX_FILE_BUNDLE" ||
		!validE2ERawDigest(requireString("deterministic/contract-bundle.json", bundle, "semantic_sha256")) ||
		ruling["projection_kind"] != "SEMANTIC_REGRESSION_PROJECTION" || decision["projection_kind"] != "SEMANTIC_REGRESSION_PROJECTION" ||
		bundle["projection_kind"] != "SEMANTIC_REGRESSION_PROJECTION" ||
		requireString("deterministic/ruling.json", ruling, "predicate_sha256") != requireString("deterministic/contract-bundle.json", bundle, "predicate_sha256") ||
		!oneString(ruling["selected_fields"], "http.status") || !oneString(decision["selected_fields"], "http.status") || !oneString(bundle["selected_fields"], "http.status") {
		t.Fatal("ruling/decision/bundle semantic projection is malformed or unlinked")
	}

	world := requireKeys("fresh/world-instance.json", "authority", "digests", "physical_run_authority")
	attempts := requireKeys("fresh/attempts.json", "authority", "confirmation_attempt_digests", "digests", "physical_run_authority", "search_attempt_groups", "search_phase_receipts")
	measurements := requireKeys("fresh/measurements.json", "authority", "digests", "envelope_digest", "physical_run_authority")
	captures := requireKeys("fresh/captures.json", "authority", "observation_digests", "physical_run_authority", "projection_digests")
	confirmation := requireKeys("fresh/confirmation.json", "authority", "canonical_sha256", "digest", "phase_receipts", "physical_fact_digests", "physical_run_authority")
	targets := requireKeys("fresh/contract-execution-target.json", "attempt_digests", "authority", "canonical_sha256", "contract_bundle_digest", "digests", "physical_run_authority", "residue_head_digests")
	processes := requireKeys("fresh/finalized-contract-run.json", "authority", "availability", "contract_bundle_digest", "contract_test_sha256", "exit_codes", "invocation_sha256", "physical_run_authority", "process_fact_canonical_sha256", "process_fact_digests", "stderr_bytes", "stderr_sha256", "stdout_bytes", "stdout_sha256", "target_digests")
	results := requireKeys("fresh/contract-execution.json", "authority", "availability", "observed_diagnostic", "outcome", "phase_receipts", "physical_run_authority", "process_fact_digests")
	if world["authority"] != "FRESH_WORLD_INSTANCE_DIGESTS" || attempts["authority"] != "FRESH_ATTEMPT_ARTIFACT_DIGESTS" ||
		measurements["authority"] != "FRESH_INSTANCE_MEASUREMENT_DIGESTS" || captures["authority"] != "FRESH_CAPTURE_AND_PROJECTION_DIGESTS" ||
		confirmation["authority"] != "PHYSICAL_ROTATED_FRESH_CONFIRMATION" || targets["authority"] != "TEN_FRESH_PRESPAWN_OFFICIAL_TARGETS" ||
		processes["authority"] != "FINALIZED_RUN_AUTHORITY_ABSENT_DIRECT_PROCESS_FACTS_ONLY" || processes["availability"] != "ABSENT" ||
		results["authority"] != "CLASSIFICATION_AUTHORITY_ABSENT_DIRECT_BUNDLE_RESULTS_ONLY" || results["availability"] != "ABSENT" ||
		results["observed_diagnostic"] != "CONFORMS|NONE" || results["outcome"] != "STANDALONE_BUNDLE_TEST_PASSED" {
		t.Fatal("fresh semantic authority labels are malformed")
	}
	physicalRunAuthority := requireString("fresh/world-instance.json", world, "physical_run_authority")
	if !validE2ETypedDigest(physicalRunAuthority) {
		t.Fatal("physical run authority is malformed")
	}
	for relative, payload := range map[string]map[string]any{
		"fresh/attempts.json": attempts, "fresh/measurements.json": measurements, "fresh/captures.json": captures,
		"fresh/confirmation.json": confirmation, "fresh/contract-execution-target.json": targets,
		"fresh/finalized-contract-run.json": processes, "fresh/contract-execution.json": results,
	} {
		if requireString(relative, payload, "physical_run_authority") != physicalRunAuthority {
			t.Fatalf("semantic payload %s is not bound to the completed physical run", relative)
		}
	}
	worldDigests := requireTypedDigests("fresh/world-instance.json", world, "digests", 88, true)
	attemptDigests := requireTypedDigests("fresh/attempts.json", attempts, "digests", 88, true)
	_ = requireTypedDigests("fresh/measurements.json", measurements, "digests", 88, true)
	_ = requireTypedDigests("fresh/captures.json", captures, "observation_digests", 88, true)
	_ = requireTypedDigests("fresh/captures.json", captures, "projection_digests", 88, false)
	if !validE2ETypedDigest(requireString("fresh/measurements.json", measurements, "envelope_digest")) || len(worldDigests) != len(attemptDigests) {
		t.Fatal("fresh world/attempt/measurement authority is malformed")
	}
	confirmationFacts := requireTypedDigests("fresh/confirmation.json", confirmation, "physical_fact_digests", 8, true)
	if !validE2ERawDigest(requireString("fresh/confirmation.json", confirmation, "canonical_sha256")) ||
		!validE2ETypedDigest(requireString("fresh/confirmation.json", confirmation, "digest")) {
		t.Fatal("confirmation authority is malformed")
	}
	targetDigests := requireTypedDigests("fresh/contract-execution-target.json", targets, "digests", 10, true)
	preSpawnAttemptDigests := requireTypedDigests("fresh/contract-execution-target.json", targets, "attempt_digests", 10, true)
	processDigests := requireTypedDigests("fresh/finalized-contract-run.json", processes, "process_fact_digests", 10, true)
	_ = requireRawDigests("fresh/contract-execution-target.json", targets, "canonical_sha256", 10, true)
	_ = requireTypedDigests("fresh/contract-execution-target.json", targets, "residue_head_digests", 10, false)
	_ = requireRawDigests("fresh/finalized-contract-run.json", processes, "process_fact_canonical_sha256", 10, true)
	_ = requireRawDigests("fresh/finalized-contract-run.json", processes, "invocation_sha256", 10, true)
	contractTestSHA256 := requireRawDigests("fresh/finalized-contract-run.json", processes, "contract_test_sha256", 10, false)
	_ = requireRawDigests("fresh/finalized-contract-run.json", processes, "stdout_sha256", 10, false)
	_ = requireRawDigests("fresh/finalized-contract-run.json", processes, "stderr_sha256", 10, false)
	requireNumbers("fresh/finalized-contract-run.json", processes, "exit_codes", 10, 0)
	requireNumbers("fresh/finalized-contract-run.json", processes, "stderr_bytes", 10, 0)
	requirePositiveNumbers("fresh/finalized-contract-run.json", processes, "stdout_bytes", 10)
	if !slicesEqual(targetDigests, requireStrings("fresh/finalized-contract-run.json", processes, "target_digests", 10, true)) ||
		!slicesEqual(processDigests, requireStrings("fresh/contract-execution.json", results, "process_fact_digests", 10, true)) ||
		!validE2ETypedDigest(requireString("fresh/contract-execution-target.json", targets, "contract_bundle_digest")) ||
		requireString("fresh/finalized-contract-run.json", processes, "contract_bundle_digest") != requireString("fresh/contract-execution-target.json", targets, "contract_bundle_digest") ||
		len(preSpawnAttemptDigests) != len(targetDigests) || len(stringSet(contractTestSHA256)) != 1 {
		t.Fatal("target/process/result lineage is malformed")
	}
	searchAuthorities := requirePhaseReceipts(t, "fresh/attempts.json", attempts["search_phase_receipts"], "search", 80)
	confirmAuthorities := requirePhaseReceipts(t, "fresh/confirmation.json", confirmation["phase_receipts"], "confirm", 8)
	contractAuthorities := requirePhaseReceipts(t, "fresh/contract-execution.json", results["phase_receipts"], "contract", 10)
	searchAttemptDigests := requireSearchAttemptGroups(t, attempts["search_attempt_groups"])
	confirmationAttemptDigests := requireTypedDigests("fresh/attempts.json", attempts, "confirmation_attempt_digests", 8, true)
	if !slicesEqual(prefixRaw(searchAttemptDigests), searchAuthorities) {
		t.Fatal("search phase receipts do not match the exact ordered search-attempt partition")
	}
	partitionedAttempts := append(append([]string(nil), searchAttemptDigests...), confirmationAttemptDigests...)
	sort.Strings(partitionedAttempts)
	if !slicesEqual(partitionedAttempts, attemptDigests) {
		t.Fatal("search and confirmation attempt partitions do not close the 88-attempt roster")
	}
	if !slicesEqual(prefixRaw(confirmationFacts), confirmAuthorities) || !slicesEqual(prefixRaw(processDigests), contractAuthorities) {
		t.Fatal("confirmation or contract phase receipts do not match typed authorities")
	}
}

func requireSearchAttemptGroups(t *testing.T, value any) []string {
	t.Helper()
	want := []struct {
		name  string
		count int
	}{
		{name: "decisive", count: 12},
		{name: "recovery_one", count: 12},
		{name: "recovery_two", count: 12},
		{name: "shape_reference", count: 4},
		{name: "shape_changed", count: 4},
		{name: "baseline_checkpoint", count: 12},
		{name: "divergent_baseline", count: 12},
		{name: "accepted_reducer_evaluation", count: 12},
	}
	items, ok := value.([]any)
	if !ok || len(items) != len(want) {
		t.Fatalf("search attempt groups = %#v", value)
	}
	result := make([]string, 0, 80)
	for index, expected := range want {
		group, ok := items[index].(map[string]any)
		if !ok || len(group) != 2 || group["name"] != expected.name {
			t.Fatalf("search attempt group %d = %#v", index, items[index])
		}
		digests, ok := group["digests"].([]any)
		if !ok || len(digests) != expected.count {
			t.Fatalf("search attempt group %s count = %#v", expected.name, group["digests"])
		}
		for digestIndex, raw := range digests {
			digest, ok := raw.(string)
			if !ok || !validE2ETypedDigest(digest) {
				t.Fatalf("search attempt group %s digest %d = %#v", expected.name, digestIndex, raw)
			}
			result = append(result, digest)
		}
	}
	if len(result) != 80 || len(stringSet(result)) != 80 {
		t.Fatal("search attempt groups do not contain 80 unique authorities")
	}
	return result
}

func requirePhaseReceipts(t *testing.T, relative string, value any, phase string, count int) []string {
	t.Helper()
	items, ok := value.([]any)
	if !ok || len(items) != count {
		t.Fatalf("phase receipts %s = %d/%d", relative, len(items), count)
	}
	result := make([]string, len(items))
	for index, item := range items {
		receipt, ok := item.(map[string]any)
		if !ok || len(receipt) != 3 || receipt["phase"] != phase || receipt["trial"] != float64(index+1) {
			t.Fatalf("phase receipt %s[%d] = %#v", relative, index, item)
		}
		authority, ok := receipt["authority_sha256"].(string)
		if !ok || !validE2ERawDigest(authority) {
			t.Fatalf("phase receipt %s[%d] authority = %#v", relative, index, receipt["authority_sha256"])
		}
		result[index] = authority
	}
	if len(stringSet(result)) != len(result) {
		t.Fatalf("phase receipts %s contain duplicate authority", relative)
	}
	return result
}

func validE2ETypedDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validE2ERawDigest(strings.TrimPrefix(value, "sha256:"))
}

func validE2ERawDigest(value string) bool {
	if len(value) != 64 || value == strings.Repeat("0", 64) {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func oneString(value any, want string) bool {
	items, ok := value.([]any)
	return ok && len(items) == 1 && items[0] == want
}

func prefixRaw(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.TrimPrefix(value, "sha256:")
	}
	return result
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func slicesEqual(left, right []string) bool {
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

func buildE2ECountershape(t *testing.T, repositoryRoot string) string {
	t.Helper()
	buildRoot := privateE2EDirectory(t, "compiled-product")
	binary := filepath.Join(buildRoot, "countershape")
	ctx, cancel := context.WithTimeout(context.Background(), e2eBuildTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, e2eGo,
		"build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-p=1", "-o", binary, "./cmd/countershape")
	command.Dir = repositoryRoot
	command.Env = e2eGoBuildEnvironment()
	configureE2EProcessGroup(command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || ctx.Err() != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("build non-race product: err=%v context=%v stdout=%q stderr=%q", err, ctx.Err(), stdout.Bytes(), stderr.Bytes())
	}
	metadata, err := os.Lstat(binary)
	if err != nil || !metadata.Mode().IsRegular() || metadata.Mode()&os.ModeSymlink != 0 || metadata.Mode().Perm()&0o111 == 0 || metadata.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		t.Fatalf("compiled product mode = %v err=%v", metadata, err)
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatalf("read compiled product build info: %v", err)
	}
	for _, setting := range info.Settings {
		if setting.Key == "-race" && setting.Value == "true" {
			t.Fatal("complete physical product child unexpectedly carries race instrumentation")
		}
	}
	return binary
}

func runE2ECompiledStudy(
	t *testing.T,
	binary string,
	fixtureRoot string,
	scratchRoot string,
	evidenceRoot string,
	ordinal int,
	stdout *bytes.Buffer,
	stderr *bytes.Buffer,
) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), e2eStudyTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "study", "http", "--json")
	command.Dir = fixtureRoot
	command.Env = []string{
		"HOME=" + scratchRoot,
		"TMPDIR=" + scratchRoot,
		"PATH=/usr/bin:/bin:/opt/homebrew/bin",
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "COLUMNS=80",
		"NODE_OPTIONS=", "NODE_PATH=",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0",
		"COUNTERSHAPE_STUDY_DOMAIN=http",
		"COUNTERSHAPE_STUDY_ORDINAL=" + fmt.Sprint(ordinal),
		"COUNTERSHAPE_EVIDENCE_ROOT=" + evidenceRoot,
		"COUNTERSHAPE_NODE=" + e2eNode,
		"COUNTERSHAPE_GIT=" + e2eGit,
	}
	configureE2EProcessGroup(command)
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("compiled product exceeded the %s test-only deadline: %v", e2eStudyTimeout, ctx.Err())
	}
	if err == nil {
		return 0
	}
	exit, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run compiled product: %v", err)
	}
	return exit.ExitCode()
}

func e2eGoBuildEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOFLAGS=") || strings.HasPrefix(entry, "GOENV=") || strings.HasPrefix(entry, "GOTOOLCHAIN=") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "GOFLAGS=", "GOENV=off", "GOTOOLCHAIN=local")
}

func resolvedE2EExecutable(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("resolve executable %q: resolved=%q err=%v", path, resolved, err)
	}
	metadata, err := os.Lstat(resolved)
	if err != nil || !metadata.Mode().IsRegular() || metadata.Mode().Perm()&0o111 == 0 || metadata.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("resolved executable %q is inadmissible: mode=%v err=%v", resolved, metadata, err)
	}
	return resolved
}

func configureE2EProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return err
		}
		return nil
	}
	command.WaitDelay = 10 * time.Second
}

func prepareE2EHTTPFixture(t *testing.T, repositoryRoot string, ordinal int) string {
	t.Helper()
	fixtureRoot := filepath.Join(canonicalE2ETemp(t), "fixture")
	driver := filepath.Join(repositoryRoot, "tools", "run-u7-http-study.mjs")
	command := exec.Command(e2eNode, driver,
		"--prepare", "--domain", "http", "--ordinal", fmt.Sprint(ordinal), "--fixture-root", fixtureRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("prepare HTTP fixture: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	return fixtureRoot
}

func e2eRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		t.Fatalf("repository root is not canonical: root=%q resolved=%q err=%v", root, resolved, err)
	}
	return root
}

func privateE2EDirectory(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(canonicalE2ETemp(t), name)
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func canonicalE2ETemp(t *testing.T) string {
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

func e2eGitLine(t *testing.T, root string, args ...string) string {
	t.Helper()
	line := strings.TrimSuffix(string(e2eGitBytes(t, root, args...)), "\n")
	if line == "" || strings.ContainsAny(line, "\r\n\x00") {
		t.Fatalf("git %s did not return one line: %q", args[0], line)
	}
	return line
}

func e2eGitBytes(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	closed := append([]string{"-C", root, "--no-pager", "--no-replace-objects", "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null"}, args...)
	command := exec.Command(e2eGit, closed...)
	command.Env = []string{
		"HOME=" + root, "TMPDIR=" + root, "PATH=/usr/bin:/bin",
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("git %s: err=%v stderr=%q", args[0], err, stderr.Bytes())
	}
	return append([]byte(nil), stdout.Bytes()...)
}
