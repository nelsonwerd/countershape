package clistudy

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	runtimedebug "runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

const (
	completedCLIStudyChildEnvironment = "COUNTERSHAPE_U7C_NONRACE_COMPLETION_CHILD"
	completedCLIStudyChildSuccess     = "U7C_COMPLETE_CLI_WORKFLOW_CHILD_OK\n"
	completedCLIStudyBuildTimeout     = 90 * time.Second
	completedCLIStudyChildTimeout     = 12 * time.Minute
	completedCLIStudySharedTimeout    = 12*time.Minute + 30*time.Second
	maximumCLIStudyChildOutput        = 1 << 20
)

type boundedCLIStudyChildOutput struct {
	buffer   bytes.Buffer
	overflow bool
}

func (output *boundedCLIStudyChildOutput) Write(value []byte) (int, error) {
	want := len(value)
	remaining := maximumCLIStudyChildOutput - output.buffer.Len()
	if remaining > 0 {
		if remaining > len(value) {
			remaining = len(value)
		}
		_, _ = output.buffer.Write(value[:remaining])
	}
	if remaining < len(value) {
		output.overflow = true
	}
	return want, nil
}

func (output *boundedCLIStudyChildOutput) String() string {
	return output.buffer.String()
}

func currentCLITestRaceEnabled(t testing.TB) bool {
	t.Helper()
	info, ok := runtimedebug.ReadBuildInfo()
	if !ok {
		t.Fatal("complete CLI workflow build information is unavailable")
	}
	seen := false
	raceEnabled := false
	for _, setting := range info.Settings {
		if setting.Key == "-race" {
			if seen || (setting.Value != "true" && setting.Value != "false") {
				t.Fatalf("complete CLI workflow race build information is invalid: seen=%t value=%q", seen, setting.Value)
			}
			seen = true
			raceEnabled = setting.Value == "true"
		}
	}
	return raceEnabled
}

func runCompletedCLIStudyNonRaceChild(t *testing.T) {
	t.Helper()
	repositoryRoot := testRepositoryRoot(t)
	goExecutable := requireExactExecutable(t, "/opt/homebrew/bin/go")
	childRoot := privateDirectoryForTB(t, "nonrace-completion-child")
	binary := filepath.Join(childRoot, "clistudy.test")

	sharedContext, cancelShared := context.WithTimeout(context.Background(), completedCLIStudySharedTimeout)
	defer cancelShared()
	buildContext, cancelBuild := context.WithTimeout(sharedContext, completedCLIStudyBuildTimeout)
	build := exec.CommandContext(buildContext, goExecutable,
		"test", "-c", "-race=false", "-trimpath", "-mod=readonly", "-buildvcs=false", "-p=1",
		"-o", binary, "./internal/reference/clistudy",
	)
	build.Dir = repositoryRoot
	build.Env = completedCLIStudyChildProcessEnvironment(false)
	configureCLIStudyChildProcess(build)
	var buildStdout, buildStderr boundedCLIStudyChildOutput
	build.Stdout = &buildStdout
	build.Stderr = &buildStderr
	buildErr := runCLIStudyOwnedProcess(build)
	buildContextErr := buildContext.Err()
	cancelBuild()
	if buildErr != nil || buildContextErr != nil || buildStdout.overflow || buildStderr.overflow ||
		buildStdout.String() != "" || buildStderr.String() != "" {
		t.Fatalf("build current-tree non-race CLI completion child: err=%v context=%v stdout-overflow=%t stderr-overflow=%t stdout=%q stderr=%q",
			buildErr, buildContextErr, buildStdout.overflow, buildStderr.overflow,
			buildStdout.String(), buildStderr.String())
	}
	metadata, err := os.Lstat(binary)
	if err != nil || !metadata.Mode().IsRegular() || metadata.Mode()&os.ModeSymlink != 0 ||
		metadata.Mode().Perm()&0o111 == 0 || metadata.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		t.Fatalf("non-race CLI completion child mode = %v err=%v", metadata, err)
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatalf("read non-race CLI completion child build info: %v", err)
	}
	for _, setting := range info.Settings {
		if setting.Key == "-race" && setting.Value == "true" {
			t.Fatal("current-tree CLI completion child was built with race instrumentation")
		}
	}

	child := exec.CommandContext(sharedContext, binary,
		"-test.run=^TestCompletedCLIStudyClosesTenDistinctOfficialGraphs$",
		"-test.count=1", "-test.timeout="+completedCLIStudyChildTimeout.String(),
	)
	child.Dir = filepath.Join(repositoryRoot, "internal", "reference", "clistudy")
	child.Env = completedCLIStudyChildProcessEnvironment(true)
	configureCLIStudyChildProcess(child)
	var childStdout, childStderr boundedCLIStudyChildOutput
	child.Stdout = &childStdout
	child.Stderr = &childStderr
	childErr := runCLIStudyOwnedProcess(child)
	if childErr != nil || sharedContext.Err() != nil || childStdout.overflow || childStderr.overflow ||
		childStdout.String() != completedCLIStudyChildSuccess+"PASS\n" || childStderr.String() != "" {
		t.Fatalf("run current-tree non-race CLI completion child: err=%v context=%v stdout-overflow=%t stderr-overflow=%t stdout=%q stderr=%q",
			childErr, sharedContext.Err(), childStdout.overflow, childStderr.overflow,
			childStdout.String(), childStderr.String())
	}
}

func completedCLIStudyChildProcessEnvironment(child bool) []string {
	environment := make([]string, 0, len(os.Environ())+4)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOFLAGS=") || strings.HasPrefix(entry, "GOENV=") ||
			strings.HasPrefix(entry, "GOTOOLCHAIN=") || strings.HasPrefix(entry, "COUNTERSHAPE_") {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment, "GOFLAGS=", "GOENV=off", "GOTOOLCHAIN=local")
	if child {
		environment = append(environment, completedCLIStudyChildEnvironment+"=1")
	}
	return environment
}

func configureCLIStudyChildProcess(command *exec.Cmd) {
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

func runCLIStudyOwnedProcess(command *exec.Cmd) error {
	if err := command.Start(); err != nil {
		return err
	}
	pid := command.Process.Pid
	pgid, ownershipErr := syscall.Getpgid(pid)
	if ownershipErr != nil || pgid != pid {
		killErr := command.Process.Kill()
		waitErr := command.Wait()
		return errors.Join(errors.New("CLI_STUDY_CHILD_PROCESS_GROUP_OWNERSHIP_REFUSED"), ownershipErr, killErr, waitErr)
	}
	waitErr := command.Wait()
	terminalErr := requireCLIStudyProcessGroupTerminal(pid)
	return errors.Join(waitErr, terminalErr)
}

func requireCLIStudyProcessGroupTerminal(pgid int) error {
	probeErr := syscall.Kill(-pgid, 0)
	if probeErr == syscall.ESRCH {
		return nil
	}
	if probeErr != nil {
		return errors.Join(errors.New("CLI_STUDY_CHILD_PROCESS_GROUP_PROBE_REFUSED"), probeErr)
	}
	survivorErr := errors.New("CLI_STUDY_CHILD_PROCESS_GROUP_SURVIVED")
	killErr := syscall.Kill(-pgid, syscall.SIGKILL)
	if killErr != nil && killErr != syscall.ESRCH {
		return errors.Join(survivorErr, killErr)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		probeErr = syscall.Kill(-pgid, 0)
		if probeErr == syscall.ESRCH {
			return survivorErr
		}
		if probeErr != nil {
			return errors.Join(survivorErr, probeErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.Join(survivorErr, errors.New("CLI_STUDY_CHILD_PROCESS_GROUP_CLEANUP_UNCERTAIN"))
}

func TestCLIStudyOwnedProcessRequiresTerminalGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", "trap '' HUP; sleep 30 </dev/null >/dev/null 2>&1 & exit 0")
	configureCLIStudyChildProcess(command)
	err := runCLIStudyOwnedProcess(command)
	if err == nil || !strings.Contains(err.Error(), "CLI_STUDY_CHILD_PROCESS_GROUP_SURVIVED") {
		t.Fatalf("surviving child process group = %v, want exact terminal refusal", err)
	}
	if command.Process == nil {
		t.Fatal("survivor probe lost the admitted process identity")
	}
	if probeErr := syscall.Kill(-command.Process.Pid, 0); probeErr != syscall.ESRCH {
		t.Fatalf("surviving child process group remained after fail-closed cleanup: %v", probeErr)
	}
}

func TestCLIStudyOwnedProcessAcceptsTerminalGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", "exit 0")
	configureCLIStudyChildProcess(command)
	if err := runCLIStudyOwnedProcess(command); err != nil {
		t.Fatalf("terminal child process group was refused: %v", err)
	}
}

func TestReductionCohortTopologyRequiresDistinctGraphsAndSharedWitness(t *testing.T) {
	weakRun := testDigest("reduction-topology", "weak-run")
	strongRun := testDigest("reduction-topology", "strong-run")
	weakTranscript := testDigest("reduction-topology", "weak-transcript")
	strongTranscript := testDigest("reduction-topology", "strong-transcript")
	weakGrade := testDigest("reduction-topology", "weak-grade")
	strongGrade := testDigest("reduction-topology", "strong-grade")
	sharedMinimized := testDigest("reduction-topology", "shared-minimized")
	sharedCanonical := []byte(`{"kind":"shared-minimized"}`)

	validate := func(
		leftRun, rightRun, leftTranscript, rightTranscript, leftGrade, rightGrade,
		leftMinimized, rightMinimized domain.Digest,
		leftCanonical, rightCanonical []byte,
	) error {
		return validateReductionCohortTopology(
			leftRun, rightRun, leftTranscript, rightTranscript, leftGrade, rightGrade,
			leftMinimized, rightMinimized, sharedMinimized,
			leftCanonical, rightCanonical, sharedCanonical,
		)
	}
	if err := validate(
		weakRun, strongRun, weakTranscript, strongTranscript, weakGrade, strongGrade,
		sharedMinimized, sharedMinimized, sharedCanonical, sharedCanonical,
	); err != nil {
		t.Fatalf("distinct reduction graphs over one shared witness were refused: %v", err)
	}

	aliasCases := []struct {
		name                            string
		leftRun, rightRun               domain.Digest
		leftTranscript, rightTranscript domain.Digest
		leftGrade, rightGrade           domain.Digest
		leftMinimized, rightMinimized   domain.Digest
		leftCanonical, rightCanonical   []byte
	}{
		{"run", weakRun, weakRun, weakTranscript, strongTranscript, weakGrade, strongGrade, sharedMinimized, sharedMinimized, sharedCanonical, sharedCanonical},
		{"transcript", weakRun, strongRun, weakTranscript, weakTranscript, weakGrade, strongGrade, sharedMinimized, sharedMinimized, sharedCanonical, sharedCanonical},
		{"grade", weakRun, strongRun, weakTranscript, strongTranscript, weakGrade, weakGrade, sharedMinimized, sharedMinimized, sharedCanonical, sharedCanonical},
	}
	for _, testCase := range aliasCases {
		err := validate(
			testCase.leftRun, testCase.rightRun, testCase.leftTranscript, testCase.rightTranscript,
			testCase.leftGrade, testCase.rightGrade, testCase.leftMinimized, testCase.rightMinimized,
			testCase.leftCanonical, testCase.rightCanonical,
		)
		if err == nil || !strings.Contains(err.Error(), "CLI_STUDY_COMPLETED_REDUCTION_ALIAS_REFUSED") {
			t.Fatalf("%s authority alias = %v, want exact alias refusal", testCase.name, err)
		}
	}

	foreignMinimized := testDigest("reduction-topology", "foreign-minimized")
	for _, testCase := range []struct {
		name                          string
		leftMinimized, rightMinimized domain.Digest
		leftCanonical, rightCanonical []byte
	}{
		{"digest", sharedMinimized, foreignMinimized, sharedCanonical, sharedCanonical},
		{"canonical", sharedMinimized, sharedMinimized, sharedCanonical, []byte(`{"kind":"foreign-minimized"}`)},
		{"expected digest anchor", foreignMinimized, foreignMinimized, sharedCanonical, sharedCanonical},
		{"expected canonical anchor", sharedMinimized, sharedMinimized, []byte(`{"kind":"foreign-minimized"}`), []byte(`{"kind":"foreign-minimized"}`)},
	} {
		err := validate(
			weakRun, strongRun, weakTranscript, strongTranscript, weakGrade, strongGrade,
			testCase.leftMinimized, testCase.rightMinimized, testCase.leftCanonical, testCase.rightCanonical,
		)
		if err == nil || !strings.Contains(err.Error(), "CLI_STUDY_COMPLETED_REDUCTION_CONVERGENCE_REFUSED") {
			t.Fatalf("%s witness divergence = %v, want exact convergence refusal", testCase.name, err)
		}
	}
}

func TestWeakAndStrongReductionConstructionSharesCanonicalMinimizedWitness(t *testing.T) {
	base, err := defaultCLIStimulus(reference.CLIEntrypoint)
	if err != nil {
		t.Fatal(err)
	}
	mainNoisy, err := noisyCLIStimulus(base, "z-main-irrelevant.txt")
	if err != nil {
		t.Fatal(err)
	}
	auxiliaryNoisy, err := noisyCLIStimulus(base, "z-auxiliary-irrelevant.txt")
	if err != nil {
		t.Fatal(err)
	}
	weakPolicy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: mainNoisy, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIFixtureRemove, countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	strongPolicy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: auxiliaryNoisy, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIFixtureRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	weakNeighbors, err := countercli.EnumerateCLINeighbors(mainNoisy, weakPolicy)
	if err != nil {
		t.Fatal(err)
	}
	strongNeighbors, err := countercli.EnumerateCLINeighbors(auxiliaryNoisy, strongPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if len(weakNeighbors) != 2 || len(strongNeighbors) != 1 ||
		mainNoisy.Digest() == auxiliaryNoisy.Digest() || weakPolicy.Digest() == strongPolicy.Digest() ||
		weakNeighbors[0].LogicalNeighbor().Digest() == strongNeighbors[0].LogicalNeighbor().Digest() ||
		weakNeighbors[0].ReplayRecipe().Rule != countercli.CLIFixtureRemove ||
		strongNeighbors[0].ReplayRecipe().Rule != countercli.CLIFixtureRemove ||
		weakNeighbors[0].Stimulus().Digest() != base.Digest() ||
		strongNeighbors[0].Stimulus().Digest() != base.Digest() ||
		!bytes.Equal(weakNeighbors[0].Stimulus().CanonicalBytes(), base.CanonicalBytes()) ||
		!bytes.Equal(strongNeighbors[0].Stimulus().CanonicalBytes(), base.CanonicalBytes()) {
		t.Fatal("independent noisy reduction cohorts did not converge on the exact shared reference stimulus")
	}
}

func TestOfficialTrialScalarSnapshotValidationNeverCallsLiveRevalidation(t *testing.T) {
	trials, bundleDigest, residueHeadDigest := scalarOfficialTrialRoster()
	base := trials[0]
	other := trials[1]

	mutations := []struct {
		name   string
		mutate func(*officialTrial)
	}{
		{"attempt digest", func(trial *officialTrial) { trial.attemptDigest = other.attemptDigest }},
		{"target digest", func(trial *officialTrial) { trial.targetDigest = other.targetDigest }},
		{"target canonical bytes", func(trial *officialTrial) { trial.targetCanonicalSHA256 = other.targetCanonicalSHA256 }},
		{"bundle digest", func(trial *officialTrial) { trial.bundleDigest = testDigest("foreign", "bundle") }},
		{"residue head digest", func(trial *officialTrial) { trial.residueHeadDigest = testDigest("foreign", "residue") }},
		{"run digest", func(trial *officialTrial) { trial.runDigest = other.runDigest }},
		{"run canonical bytes", func(trial *officialTrial) { trial.runCanonicalSHA256 = other.runCanonicalSHA256 }},
		{"classification digest", func(trial *officialTrial) { trial.classificationDigest = other.classificationDigest }},
		{"classification canonical bytes", func(trial *officialTrial) { trial.classificationCanonicalSHA256 = other.classificationCanonicalSHA256 }},
		{"result", func(trial *officialTrial) { trial.result = "CONTRADICTS" }},
		{"exit code", func(trial *officialTrial) { trial.exitCode = 7 }},
		{"recovery equality", func(trial *officialTrial) { trial.recoveryEqual = false }},
	}
	for _, mutation := range mutations {
		forged := base
		mutation.mutate(&forged)
		if err := base.validateSnapshot(forged); err == nil {
			t.Fatalf("scalar snapshot validator accepted mutated %s", mutation.name)
		}
	}

	liveCalls := 0
	for index := range trials {
		trials[index].revalidate = func(context.Context, officialTrial) error {
			liveCalls++
			return errors.New("live revalidation must not run from the pure roster validator")
		}
	}
	if err := validateOfficialTrialRoster(trials, bundleDigest, residueHeadDigest); err != nil {
		t.Fatalf("validate pure scalar official roster: %v", err)
	}
	if liveCalls != 0 {
		t.Fatalf("pure official roster validation performed %d live revalidations", liveCalls)
	}
}

func TestOfficialTrialRosterCancellationStopsAfterFirstLiveRevalidation(t *testing.T) {
	trials, bundleDigest, residueHeadDigest := scalarOfficialTrialRoster()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	seenTargets := make([]domain.Digest, 0, 1)
	err := revalidateOfficialTrialRosterWith(
		ctx, trials, bundleDigest, residueHeadDigest,
		func(_ context.Context, trial officialTrial) error {
			calls++
			seenTargets = append(seenTargets, trial.targetDigest)
			cancel()
			return nil
		},
	)
	if err == nil || !errors.Is(err, context.Canceled) ||
		!strings.Contains(err.Error(), "CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_POST_01_CONTEXT_REFUSED") {
		t.Fatalf("cancel after first official revalidation = %v, want exact post-trial refusal", err)
	}
	if calls != 1 || len(seenTargets) != 1 || seenTargets[0] != trials[0].targetDigest {
		t.Fatalf("canceled roster reached later trials: calls=%d targets=%v", calls, seenTargets)
	}
}

func TestOfficialTrialConcurrentRosterUsesExactRepositoryBoundAndJoins(t *testing.T) {
	node := requireExactExecutable(t, "/opt/homebrew/bin/node")
	git := requireExactExecutable(t, "/usr/bin/git")
	root := prepareCLICompletionFixture(t, node)
	fixture, err := reference.OpenCLIFixture(context.Background(), reference.CLIFixtureConfig{
		Root: root, GitExecutable: git,
		ScratchRoot:     privateDirectoryForTB(t, "official-concurrent-repository-scratch"),
		RepositoryCount: officialCLIWorkerLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	repositories := fixture.Repositories()
	if len(repositories) != officialCLIWorkerLimit {
		t.Fatalf("official concurrent repository roster = %d, want %d", len(repositories), officialCLIWorkerLimit)
	}
	fingerprint := repositories[0].Fingerprint()
	format := repositories[0].ObjectFormat()
	aliased := make([]gitobj.Repository, officialCLIWorkerLimit)
	for index := range aliased {
		aliased[index] = repositories[0]
	}
	if err := runOfficialRepositoryTasks(context.Background(), aliased, 1, func(context.Context, int, gitobj.Repository) error {
		return nil
	}); err == nil || !strings.Contains(err.Error(), "CLI_STUDY_OFFICIAL_REPOSITORY_ALIAS_REFUSED") {
		t.Fatalf("aliased official repository roster = %v, want exact refusal", err)
	}

	trials, bundleDigest, residueHeadDigest := scalarOfficialTrialRoster()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	started := make(chan domain.Digest, len(trials))
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var state sync.Mutex
	active, maximumActive, calls := 0, 0, 0
	seen := make(map[domain.Digest]int, len(trials))

	result := make(chan error, 1)
	go func() {
		result <- revalidateOfficialTrialRosterConcurrent(
			ctx, trials, bundleDigest, residueHeadDigest, repositories,
			func(taskContext context.Context, trial officialTrial, repository gitobj.Repository) error {
				if !repository.Valid() || repository.Fingerprint() != fingerprint || repository.ObjectFormat() != format {
					return errors.New("foreign official repository lease")
				}
				state.Lock()
				active++
				calls++
				seen[trial.targetDigest]++
				if active > maximumActive {
					maximumActive = active
				}
				state.Unlock()
				defer func() {
					state.Lock()
					active--
					state.Unlock()
				}()
				started <- trial.targetDigest
				select {
				case <-release:
					return nil
				case <-taskContext.Done():
					return taskContext.Err()
				}
			},
		)
	}()

	initial := make(map[domain.Digest]struct{}, officialCLIWorkerLimit)
	for len(initial) < officialCLIWorkerLimit {
		select {
		case target := <-started:
			initial[target] = struct{}{}
		case <-ctx.Done():
			t.Fatalf("official concurrent roster did not reach its bound: %v", ctx.Err())
		}
	}
	state.Lock()
	if active != officialCLIWorkerLimit || maximumActive != officialCLIWorkerLimit {
		state.Unlock()
		t.Fatalf("official concurrent state before release = active:%d maximum:%d, want %d", active, maximumActive, officialCLIWorkerLimit)
	}
	state.Unlock()
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("official concurrent roster returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("official concurrent roster did not join: %v", ctx.Err())
	}
	state.Lock()
	defer state.Unlock()
	if active != 0 || maximumActive != officialCLIWorkerLimit || calls != len(trials) || len(seen) != len(trials) {
		t.Fatalf("official concurrent terminal state = active:%d maximum:%d calls:%d seen:%d", active, maximumActive, calls, len(seen))
	}
	for index, trial := range trials {
		if seen[trial.targetDigest] != 1 {
			t.Fatalf("official trial %d revalidation count = %d, want 1", index+1, seen[trial.targetDigest])
		}
	}
}

func TestOfficialRevalidationRootIsRemovedAfterBodyFailure(t *testing.T) {
	authorityRoot := privateDirectory(t, "official-revalidation-authority")
	bodyFailure := errors.New("injected official revalidation failure")
	createdRoot := ""
	err := withOfficialRevalidationRoot(authorityRoot, 1, func(root string) error {
		createdRoot = root
		metadata, statErr := os.Lstat(root)
		if statErr != nil || !metadata.IsDir() || metadata.Mode().Perm() != 0o700 || filepath.Dir(root) != authorityRoot {
			t.Fatalf("private official revalidation root = %q metadata=%v err=%v", root, metadata, statErr)
		}
		if writeErr := os.WriteFile(filepath.Join(root, "leftover"), []byte("must be removed"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		return bodyFailure
	})
	if !errors.Is(err, bodyFailure) {
		t.Fatalf("official revalidation body failure was not preserved: %v", err)
	}
	if createdRoot == "" {
		t.Fatal("official revalidation body did not receive a private root")
	}
	if _, statErr := os.Lstat(createdRoot); !os.IsNotExist(statErr) {
		t.Fatalf("official revalidation root survived body failure: %q err=%v", createdRoot, statErr)
	}
	if entries, readErr := os.ReadDir(authorityRoot); readErr != nil || len(entries) != 0 {
		t.Fatalf("official revalidation authority retained residue: entries=%v err=%v", entries, readErr)
	}
}

func scalarOfficialTrialRoster() ([]officialTrial, domain.Digest, domain.Digest) {
	bundleDigest := testDigest("official-scalar", "bundle")
	residueHeadDigest := testDigest("official-scalar", "residue")
	trials := make([]officialTrial, 0, 10)
	for index := 0; index < 10; index++ {
		label := string(rune('a' + index))
		snapshot := officialTrialSnapshot{
			attemptDigest:                 testDigest("official-scalar", label, "attempt"),
			targetDigest:                  testDigest("official-scalar", label, "target"),
			targetCanonicalSHA256:         strings.TrimPrefix(testDigest("official-scalar", label, "target-bytes").String(), "sha256:"),
			bundleDigest:                  bundleDigest,
			residueHeadDigest:             residueHeadDigest,
			runDigest:                     testDigest("official-scalar", label, "run"),
			runCanonicalSHA256:            strings.TrimPrefix(testDigest("official-scalar", label, "run-bytes").String(), "sha256:"),
			classificationDigest:          testDigest("official-scalar", label, "classification"),
			classificationCanonicalSHA256: strings.TrimPrefix(testDigest("official-scalar", label, "classification-bytes").String(), "sha256:"),
			result:                        "CONFORMS",
			exitCode:                      0,
			recoveryEqual:                 true,
		}
		trial := officialTrial{
			attemptDigest: snapshot.attemptDigest, targetDigest: snapshot.targetDigest,
			targetCanonicalSHA256: snapshot.targetCanonicalSHA256,
			bundleDigest:          snapshot.bundleDigest, residueHeadDigest: snapshot.residueHeadDigest,
			runDigest: snapshot.runDigest, runCanonicalSHA256: snapshot.runCanonicalSHA256,
			classificationDigest:          snapshot.classificationDigest,
			classificationCanonicalSHA256: snapshot.classificationCanonicalSHA256,
			result:                        snapshot.result, exitCode: snapshot.exitCode, recoveryEqual: snapshot.recoveryEqual,
			validateSnapshot: newOfficialTrialSnapshotValidator(snapshot),
			revalidate: func(context.Context, officialTrial) error {
				return nil
			},
		}
		trials = append(trials, trial)
	}
	return trials, bundleDigest, residueHeadDigest
}

func TestCompletedCLIStudyClosesTenDistinctOfficialGraphs(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("the sealed physical runner is qualified on Darwin arm64")
	}
	childAuthority := os.Getenv(completedCLIStudyChildEnvironment)
	raceEnabled := currentCLITestRaceEnabled(t)
	if childAuthority != "" && childAuthority != "1" {
		t.Fatalf("invalid complete CLI workflow child authority %q", childAuthority)
	}
	if childAuthority == "1" {
		if raceEnabled {
			t.Fatal("complete CLI workflow child unexpectedly carries race instrumentation")
		}
		t.Setenv(completedCLIStudyChildEnvironment, "")
	} else if raceEnabled {
		runCompletedCLIStudyNonRaceChild(t)
		return
	} else {
		t.Skip("complete CLI workflow is owned by the frozen race-parent/current-tree non-race-child verification topology")
	}
	node := requireExactExecutable(t, "/opt/homebrew/bin/node")
	git := requireExactExecutable(t, "/usr/bin/git")
	fixtureRoot := prepareCLICompletionFixture(t, node)
	scratchRoot := privateDirectory(t, "completion-scratch")
	evidenceRoot := privateDirectory(t, "completion-evidence")
	ambientHome := privateDirectoryForTB(t, "ambient-home")
	if err := os.WriteFile(filepath.Join(ambientHome, ".countershape-precedence.json"), []byte(`{"mode":"ambient-must-not-cross"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", ambientHome)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	completed, err := CompleteCLIStudy(ctx, Config{
		RepositoryRoot: fixtureRoot,
		ScratchRoot:    scratchRoot,
		EvidenceRoot:   evidenceRoot,
		GitExecutable:  git,
		NodeExecutable: node,
		Ordinal:        1,
	})
	if err != nil || !completed.Valid() {
		t.Fatalf("complete CLI study: valid=%t err=%v", completed.Valid(), err)
	}
	if entries, err := os.ReadDir(evidenceRoot); err != nil || len(entries) != 0 {
		t.Fatalf("completion bypassed the package-sealed publisher: entries=%v err=%v", entries, err)
	}
	assertCLIChoiceClosure(t, completed)
	assertCLIDeterministicChoiceEvidenceProjection(t, completed)
	assertCLISemanticControls(t, completed, ambientHome)
	assertCLIConstructionClosure(t, completed)
	assertCompletedPhaseRoster(t, completed)
	assertPrepublicationInspection(t, completed)
	assertCompletedValidationAndInspectionArePure(t, completed)
	assertCompletedRosterForgeryRefusals(t, completed)
	assertOfficialTrialClosure(t, completed)
	if childAuthority == "1" {
		written, writeErr := os.Stdout.WriteString(completedCLIStudyChildSuccess)
		if writeErr != nil || written != len(completedCLIStudyChildSuccess) {
			t.Fatalf("write complete CLI workflow child success: written=%d want=%d err=%v",
				written, len(completedCLIStudyChildSuccess), writeErr)
		}
	}
}

func assertCLIDeterministicChoiceEvidenceProjection(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	wantSelectedFields := []string{
		string(countercli.CLIFieldExitCode),
		string(countercli.CLIFieldStdoutJSONMode),
		string(countercli.CLIFieldStdoutJSONSource),
	}
	allowed := completed.bundle.Predicate().AllowedTuples()
	if !completed.decision.Valid() || !completed.bundle.Valid() || len(completed.bundle.Files()) != 6 ||
		completed.decision.Action() != choice.ActionAllowObserved || completed.decision.EarlyReveal() ||
		!slices.Equal(completed.decision.SelectedFields(), wantSelectedFields) ||
		len(allowed) != 2 || len(completed.choiceAudit.allowedTupleBytes) != 2 {
		t.Fatal("completed choice authorities cannot support the deterministic semantic projections")
	}
	allowedTupleSHA := make([]string, len(allowed))
	for index, tuple := range allowed {
		allowedTupleSHA[index] = sha256Hex(tuple.CanonicalBytes())
		if allowedTupleSHA[index] != sha256Hex(completed.choiceAudit.allowedTupleBytes[index]) {
			t.Fatalf("deterministic allowed tuple %d does not rejoin the retained exact tuple bytes", index)
		}
	}
	input := evidenceInputForCompleted(completed)
	ruling := input.deterministic.ruling
	decision := input.deterministic.decisionRecord
	bundle := input.deterministic.contractBundle
	planned, _, _, err := planEvidenceFiles(input)
	if err != nil {
		t.Fatalf("plan the completed study evidence: %v", err)
	}
	for _, artifact := range []struct {
		field   string
		path    string
		payload evidencePayload
	}{
		{field: "ruling_sha256", path: "deterministic/ruling.json", payload: ruling},
		{field: "decision_record_sha256", path: "deterministic/decision-record.json", payload: decision},
		{field: "contract_bundle_sha256", path: "deterministic/contract-bundle.json", payload: bundle},
	} {
		raw, rawErr := canonicalEvidencePayload(artifact.field, artifact.payload)
		if rawErr != nil {
			t.Fatalf("canonicalize %s deterministic projection: %v", artifact.field, rawErr)
		}
		exact, exactErr := canonicalJSONLine(deterministicArtifactLine{
			SchemaVersion: evidenceArtifactSchema,
			Domain:        evidenceDomainCLI,
			Artifact:      artifact.field,
			Payload:       raw,
		})
		if exactErr != nil || !bytes.Equal(planned[artifact.path], exact) {
			t.Fatalf("%s deterministic projection was not routed to its exact published wrapper: %v", artifact.field, exactErr)
		}
	}

	requireExactKeys := func(label string, payload evidencePayload, want ...string) {
		t.Helper()
		actual := make([]string, 0, len(payload))
		for key := range payload {
			actual = append(actual, key)
		}
		slices.Sort(actual)
		slices.Sort(want)
		if !slices.Equal(actual, want) {
			t.Fatalf("%s deterministic projection keys = %v, want exact %v", label, actual, want)
		}
	}
	requireExactKeys(
		"ruling", ruling,
		"action", "allowed_tuple_sha256", "authority", "predicate_sha256", "projection_kind", "selected_fields",
	)
	requireExactKeys(
		"decision-record", decision,
		"authority", "canonical_record_valid", "early_reveal", "projection_kind", "selected_fields", "semantic_sha256",
	)
	requireExactKeys(
		"contract-bundle", bundle,
		"authority", "canonical_bundle_valid", "member_count", "predicate_sha256", "projection_kind", "selected_fields", "semantic_sha256",
	)

	predicateSHA := sha256Hex(completed.bundle.Predicate().CanonicalBytes())
	earlyRevealFact := "early-reveal=false"
	if completed.decision.EarlyReveal() {
		earlyRevealFact = "early-reveal=true"
	}
	wantDecisionSemanticSHA := semanticProjectionSHA256(
		"decision-record",
		string(completed.decision.Action()),
		strings.Join(wantSelectedFields, "\x00"),
		predicateSHA,
		earlyRevealFact,
	)
	wantBundleSemanticSHA := semanticProjectionSHA256(
		"contract-bundle",
		completed.bundle.PortableSource().Digest().String(),
		predicateSHA,
		"six-files",
		string(completed.decision.Action()),
	)
	for label, payload := range map[string]evidencePayload{
		"ruling": ruling, "decision-record": decision, "contract-bundle": bundle,
	} {
		if payload["projection_kind"] != "SEMANTIC_REGRESSION_PROJECTION" {
			t.Fatalf("%s deterministic projection lacks its exact semantic-projection label", label)
		}
		selected, ok := payload["selected_fields"].([]string)
		if !ok || !slices.Equal(selected, wantSelectedFields) {
			t.Fatalf("%s selected fields = %#v, want exact ordered %v", label, payload["selected_fields"], wantSelectedFields)
		}
	}
	projectedAllowed, projectedAllowedOK := ruling["allowed_tuple_sha256"].([]string)
	if ruling["action"] != string(choice.ActionAllowObserved) ||
		ruling["authority"] != "DETERMINISTIC_CORRELATED_TUPLE_RULING_PROJECTION_FROM_SEALED_DECISION" ||
		ruling["predicate_sha256"] != predicateSHA || !projectedAllowedOK ||
		len(projectedAllowed) != 2 || !slices.Equal(projectedAllowed, allowedTupleSHA) {
		t.Fatal("deterministic ruling projection changed its exact correlated-tuple semantics")
	}
	if decision["authority"] != "DETERMINISTIC_DECISION_SEMANTIC_PROJECTION_FROM_STRICT_RECORD" ||
		decision["canonical_record_valid"] != true || decision["early_reveal"] != false ||
		decision["semantic_sha256"] != wantDecisionSemanticSHA {
		t.Fatal("deterministic decision-record projection changed its strict-record semantics")
	}
	if bundle["authority"] != "DETERMINISTIC_CONTRACT_SEMANTIC_PROJECTION_FROM_STRICT_SIX_FILE_BUNDLE" ||
		bundle["canonical_bundle_valid"] != true || bundle["member_count"] != 6 ||
		bundle["predicate_sha256"] != predicateSHA || bundle["semantic_sha256"] != wantBundleSemanticSHA {
		t.Fatal("deterministic contract-bundle projection changed its strict six-file semantics")
	}

	for label, payload := range map[string]evidencePayload{
		"ruling": ruling, "decision-record": decision, "contract-bundle": bundle,
	} {
		for _, forbidden := range []string{
			"decision_digest", "digest", "canonical_sha256", "choicepoint_digest", "bundle_digest", "physical_run_authority",
		} {
			if _, present := payload[forbidden]; present {
				t.Fatalf("%s deterministic projection leaked raw or fresh authority key %q", label, forbidden)
			}
		}
	}
}

func assertCLIConstructionClosure(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	assertCLIEnvelopeClassification(t, completed.recoveryOne.Envelope)
	eligibility := compare.AssessPreservation(completed.recoveryOne.OutcomeMap, completed.recoveryTwo.OutcomeMap)
	if !eligibility.Valid() || eligibility.Relation() != compare.PreservationUnresolved ||
		eligibility.ReasonCode() != "CANDIDATE_ELIGIBILITY_OR_ADMISSION_CHANGED" ||
		completed.recoveryOne.Plan.Digest() != completed.recoveryTwo.Plan.Digest() ||
		completed.recoveryOne.Envelope.Digest() != completed.recoveryTwo.Envelope.Digest() ||
		completed.recoveryOne.CapturePolicy.Digest() != completed.recoveryTwo.CapturePolicy.Digest() ||
		completed.recoveryOne.ProjectionDefinition.Binding().Digest() != completed.recoveryTwo.ProjectionDefinition.Binding().Digest() ||
		completed.recoveryOne.OutcomeMap.ComparisonBasisDigest() != completed.recoveryTwo.OutcomeMap.ComparisonBasisDigest() ||
		!slices.Equal(completed.recoveryOne.CanonicalCandidateRoster(), completed.recoveryTwo.CanonicalCandidateRoster()) ||
		completed.recoveryOne.CapturePolicy.StdoutBytes() != 30 || completed.recoveryTwo.CapturePolicy.StdoutBytes() != 30 {
		t.Fatal("eligibility drift was replaced by generic plan, envelope, capture, or roster incomparability")
	}
	if len(completed.recoveryOne.OutcomeMap.Entries()) != 3 || len(completed.recoveryOne.OutcomeMap.Exclusions()) != 0 ||
		len(completed.recoveryTwo.OutcomeMap.Entries()) != 1 || len(completed.recoveryTwo.OutcomeMap.Exclusions()) != 2 {
		t.Fatalf("cap-30 eligibility pair has wrong disposition: reference=%d/%d drift=%d/%d",
			len(completed.recoveryOne.OutcomeMap.Entries()), len(completed.recoveryOne.OutcomeMap.Exclusions()),
			len(completed.recoveryTwo.OutcomeMap.Entries()), len(completed.recoveryTwo.OutcomeMap.Exclusions()))
	}
	for _, trial := range completed.recoveryOne.Trials {
		if !trial.Projected || trial.ProjectionRejection != nil || !trial.Observation.ProjectionEligible() {
			t.Fatalf("cap-30 reference excluded role %s", trial.Role)
		}
		mode, source, ok := projectedModeAndSource(trial.Projection)
		want := map[reference.CLIRole]string{
			reference.ConfigFirst: "c", reference.EnvironmentFirst: "e", reference.ArgvFirst: "a",
		}[trial.Role]
		wantSource := map[reference.CLIRole]string{
			reference.ConfigFirst: "config", reference.EnvironmentFirst: "env", reference.ArgvFirst: "argv",
		}[trial.Role]
		if !ok || mode != want || source != wantSource {
			t.Fatalf("cap-30 reference role %s = mode:%q source:%q", trial.Role, mode, source)
		}
	}
	for _, trial := range completed.recoveryTwo.Trials {
		if trial.Role == reference.EnvironmentFirst {
			if !trial.Projected || trial.ProjectionRejection != nil || !trial.Observation.ProjectionEligible() ||
				trial.Result.FinalizedAttempt().HasControls() {
				t.Fatal("cap-30 drift lost its exact eligible environment role")
			}
			continue
		}
		primary, controlled := trial.Result.FinalizedAttempt().PrimaryControl()
		if trial.Projected || trial.ProjectionRejection != nil || trial.Observation.ProjectionEligible() ||
			!controlled || primary != domain.ControlOutputLimit ||
			trial.Observation.Stdout().State() != countercli.ChannelTruncated ||
			len(trial.Observation.Stdout().Bytes()) != 30 {
			t.Fatalf("cap-30 drift role %s did not remain an output-limit lifecycle exclusion", trial.Role)
		}
	}
	for _, exclusion := range completed.recoveryTwo.OutcomeMap.Exclusions() {
		role, present := completed.recoveryTwo.CandidateRoles[exclusion.CandidateKey]
		if !present || role == reference.EnvironmentFirst || exclusion.Classification != observe.Uncomparable {
			t.Fatalf("eligibility drift exclusion = role:%s status:%s", role, exclusion.Classification)
		}
	}

	shape := compare.AssessPreservation(completed.shapeReference.OutcomeMap, completed.shapeChanged.OutcomeMap)
	baseline := compare.AssessPreservation(completed.baselineCheckpoint.OutcomeMap, completed.baseline.OutcomeMap)
	mainEvaluation := compare.AssessPreservation(completed.baseline.OutcomeMap, completed.mainEvaluation.OutcomeMap)
	auxiliaryEvaluation := compare.AssessPreservation(completed.auxiliaryBaseline.OutcomeMap, completed.auxiliaryEvaluation.OutcomeMap)
	referenceShapeFixtures := completed.shapeReference.Stimulus.Fixtures()
	changedShapeFixtures := completed.shapeChanged.Stimulus.Fixtures()
	if !shape.Valid() || shape.Relation() != compare.PreservationDifferent ||
		shape.ReasonCode() != "EXACT_PRESERVATION_MAP_CHANGED" ||
		completed.shapeReference.OutcomeMap.DistinctProjectionCount() != 3 ||
		completed.shapeChanged.OutcomeMap.DistinctProjectionCount() != 3 ||
		completed.shapeReference.Plan.Digest() != completed.shapeChanged.Plan.Digest() ||
		completed.shapeReference.Envelope.Digest() != completed.shapeChanged.Envelope.Digest() ||
		completed.shapeReference.CapturePolicy.Digest() != completed.shapeChanged.CapturePolicy.Digest() ||
		completed.shapeReference.ProjectionDefinition.Binding().Digest() != completed.shapeChanged.ProjectionDefinition.Binding().Digest() ||
		completed.shapeReference.OutcomeMap.ComparisonBasisDigest() != completed.shapeChanged.OutcomeMap.ComparisonBasisDigest() ||
		!slices.Equal(completed.shapeReference.CanonicalCandidateRoster(), completed.shapeChanged.CanonicalCandidateRoster()) ||
		completed.shapeReference.Stimulus.Digest() == completed.shapeChanged.Stimulus.Digest() ||
		completed.shapeReference.Stimulus.Executable() != completed.shapeChanged.Stimulus.Executable() ||
		!slices.Equal(completed.shapeReference.Stimulus.BaseArgv(), completed.shapeChanged.Stimulus.BaseArgv()) ||
		!slices.Equal(completed.shapeReference.Stimulus.Argv(), completed.shapeChanged.Stimulus.Argv()) ||
		!slices.Equal(completed.shapeReference.Stimulus.Environment(), completed.shapeChanged.Stimulus.Environment()) ||
		completed.shapeReference.Stimulus.Stdin().Presence() != completed.shapeChanged.Stimulus.Stdin().Presence() ||
		!bytes.Equal(completed.shapeReference.Stimulus.Stdin().Bytes(), completed.shapeChanged.Stimulus.Stdin().Bytes()) ||
		completed.shapeReference.Stimulus.CWDPolicy() != completed.shapeChanged.Stimulus.CWDPolicy() ||
		len(referenceShapeFixtures) != 1 || len(changedShapeFixtures) != 1 ||
		referenceShapeFixtures[0].Path() != "config.json" || changedShapeFixtures[0].Path() != "config.json" ||
		referenceShapeFixtures[0].Mode() != changedShapeFixtures[0].Mode() ||
		!bytes.Equal(referenceShapeFixtures[0].Contents(), []byte(`{"mode":"config"}`)) ||
		!bytes.Equal(changedShapeFixtures[0].Contents(), []byte(`{"mode":"shape-changed"}`)) {
		t.Fatal("same-cardinality shape trap did not isolate one config.json value under an identical comparison basis")
	}
	referenceShapeProjection := make(map[reference.CLIRole]countercli.CLIProjectionResult, 3)
	changedShapeProjection := make(map[reference.CLIRole]countercli.CLIProjectionResult, 3)
	for _, trial := range completed.shapeReference.Trials {
		referenceShapeProjection[trial.Role] = trial.Projection
	}
	for _, trial := range completed.shapeChanged.Trials {
		changedShapeProjection[trial.Role] = trial.Projection
	}
	for _, role := range reference.CLIRoles() {
		referenceProjection, referencePresent := referenceShapeProjection[role]
		changedProjection, changedPresent := changedShapeProjection[role]
		referenceFingerprint, referenceErr := domain.NewProjectionFingerprint(referenceProjection.ProjectionBytes())
		changedFingerprint, changedErr := domain.NewProjectionFingerprint(changedProjection.ProjectionBytes())
		if !referencePresent || !changedPresent || referenceErr != nil || changedErr != nil {
			t.Fatalf("shape trap lacks exact %s projections: %v %v", role, referenceErr, changedErr)
		}
		if role == reference.ConfigFirst {
			mode, source, exact := projectedModeAndSource(changedProjection)
			if !exact || mode != "shape-changed" || source != "config" || referenceFingerprint == changedFingerprint {
				t.Fatal("shape trap did not change only the config-first projection fingerprint")
			}
		} else if referenceFingerprint != changedFingerprint ||
			!bytes.Equal(referenceProjection.ProjectionBytes(), changedProjection.ProjectionBytes()) {
			t.Fatalf("shape trap changed non-config role %s", role)
		}
	}
	if !baseline.Valid() || baseline.Relation() != compare.PreservationEqual ||
		bytes.Equal(completed.baselineCheckpoint.OutcomeMap.CanonicalBytes(), completed.baseline.OutcomeMap.CanonicalBytes()) {
		t.Fatal("fresh checkpoint/baseline reconstruction did not replay as an evidence-distinct PRESERVES map")
	}
	if !mainEvaluation.Valid() || mainEvaluation.Relation() != compare.PreservationEqual ||
		!auxiliaryEvaluation.Valid() || auxiliaryEvaluation.Relation() != compare.PreservationEqual ||
		completed.confirmed.Confirmation.Assessment().Relation() != compare.PreservationEqual ||
		completed.auxiliaryConfirmed.Confirmation.Assessment().Relation() != compare.PreservationEqual {
		t.Fatal("accepted reductions or fresh confirmations do not replay exact labeled-map preservation")
	}

	results := []struct {
		label  string
		result Result
	}{
		{"main decisive", completed.discovery},
		{"eligibility reference", completed.recoveryOne},
		{"eligibility drift", completed.recoveryTwo},
		{"shape reference", completed.shapeReference},
		{"shape changed", completed.shapeChanged},
		{"baseline checkpoint", completed.baselineCheckpoint},
		{"main baseline", completed.baseline},
		{"main accepted evaluation", completed.mainEvaluation},
		{"auxiliary baseline", completed.auxiliaryBaseline},
		{"auxiliary accepted evaluation", completed.auxiliaryEvaluation},
		{"main confirmation", completed.confirmed},
		{"auxiliary confirmation", completed.auxiliaryConfirmed},
	}
	seenWorlds := make(map[domain.Digest]string, 72)
	seenAttempts := make(map[domain.Digest]string, 72)
	seenObservations := make(map[domain.Digest]string, 72)
	for _, item := range results {
		assertCLIResultGraph(t, item.label, item.result)
		assertFreshDigestSet(t, item.label+" world", item.result.OutcomeMap.EvidenceWorldDigests(), seenWorlds)
		assertFreshDigestSet(t, item.label+" attempt", item.result.OutcomeMap.EvidenceAttemptDigests(), seenAttempts)
		observations := make([]domain.Digest, len(item.result.Trials))
		for index, trial := range item.result.Trials {
			observations[index] = trial.Observation.Digest()
		}
		assertFreshDigestSet(t, item.label+" observation", observations, seenObservations)
	}
	if completed.minimized.OutcomeMap.ArtifactDigest() != completed.mainEvaluation.OutcomeMap.ArtifactDigest() ||
		completed.minimized.Stimulus.Digest() != completed.mainEvaluation.Stimulus.Digest() ||
		completed.auxiliaryMinimized.OutcomeMap.ArtifactDigest() != completed.auxiliaryEvaluation.OutcomeMap.ArtifactDigest() ||
		completed.auxiliaryMinimized.Stimulus.Digest() != completed.auxiliaryEvaluation.Stimulus.Digest() {
		t.Fatal("retained minimized witnesses do not alias the exact accepted physical evaluations")
	}
	assertCLIReductionClosure(t, completed)
}

func assertCLIEnvelopeClassification(t testing.TB, envelope domain.ComparisonEnvelope) {
	t.Helper()
	config := envelope.Config()
	measuredCount, toleratedCount, requiredCount := 0, 0, 0
	for _, measured := range config.Measured {
		if measured.Name == dimensionLogicalArgv {
			measuredCount++
			if measured.Source != domain.MeasuredProcessReceipt || measured.Comparison != domain.CompareRecordedOnly {
				t.Fatalf("logical argv measurement = source:%s comparison:%s, want process/RECORDED_ONLY", measured.Source, measured.Comparison)
			}
		}
	}
	for _, tolerated := range config.Tolerated {
		if tolerated.Name == dimensionLogicalArgv {
			toleratedCount++
			if tolerated.Tolerance != domain.MayDifferRecorded {
				t.Fatalf("logical argv tolerance = %s, want MAY_DIFFER_RECORDED", tolerated.Tolerance)
			}
		}
	}
	for _, required := range config.RequiredEqual {
		if required.Name == dimensionLogicalArgv {
			requiredCount++
		}
	}
	if measuredCount != 1 || toleratedCount != 1 || requiredCount != 0 {
		t.Fatalf("logical argv envelope classification = measured:%d tolerated:%d required:%d, want exact 1/1/0", measuredCount, toleratedCount, requiredCount)
	}

	for index := range config.Measured {
		if config.Measured[index].Name == dimensionLogicalArgv {
			config.Measured[index].Comparison = domain.CompareExact
		}
	}
	filtered := config.Tolerated[:0]
	for _, tolerated := range config.Tolerated {
		if tolerated.Name != dimensionLogicalArgv {
			filtered = append(filtered, tolerated)
		}
	}
	config.Tolerated = filtered
	config.RequiredEqual = append(config.RequiredEqual, domain.RequiredEqualDimension{Name: dimensionLogicalArgv})
	forged, err := domain.NewComparisonEnvelope(config)
	if err != nil || !forged.Digest().Valid() || forged.Digest() == envelope.Digest() {
		t.Fatalf("required-equal logical argv hostile did not construct as a distinct envelope: valid=%t err=%v", forged.Digest().Valid(), err)
	}
}

func assertCLIResultGraph(t testing.TB, label string, result Result) {
	t.Helper()
	outcome := result.OutcomeMap
	if !result.HasOutcomeMap || !outcome.ArtifactDigest().Valid() ||
		outcome.PlanDigest() != result.Plan.Digest() || outcome.StimulusDigest() != result.Stimulus.Digest() ||
		outcome.EnvelopeDigest() != result.Envelope.Digest() || outcome.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
		outcome.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
		result.Binding.PlanDigest() != result.Plan.Digest() || result.Binding.StimulusDigest() != result.Stimulus.Digest() ||
		result.Binding.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
		result.Binding.ProjectionDefinitionBinding().Digest() != result.ProjectionDefinition.Binding().Digest() {
		t.Fatalf("%s outcome map does not rejoin its plan/stimulus/envelope/capture/projection construction", label)
	}
	roster := result.CanonicalCandidateRoster()
	if !slices.Equal(outcome.CandidateRoster(), roster) || len(result.CandidateBindings) != len(roster) ||
		len(result.CandidateRoles) != len(roster) || len(outcome.Entries())+len(outcome.Exclusions()) != len(roster) {
		t.Fatalf("%s candidate roster is incomplete: map=%d+%d bindings=%d roles=%d roster=%d", label,
			len(outcome.Entries()), len(outcome.Exclusions()), len(result.CandidateBindings), len(result.CandidateRoles), len(roster))
	}
	bindings := make(map[domain.CandidateExecutionKey]domain.CandidateExecutionBinding, len(roster))
	for _, binding := range result.CandidateBindings {
		key := binding.Key()
		identity := binding.Identity()
		if !binding.Valid() || identity.WorldPlanDigest != result.Plan.Digest() ||
			identity.MaterializationPolicyDigest != result.Plan.MaterializationPolicyDigest() ||
			identity.AdapterDigest != result.Plan.AdapterDigest() ||
			identity.ProjectionDefinitionDigest != result.Plan.ProjectionDefinitionDigest() {
			t.Fatalf("%s candidate binding %s does not rejoin the compiled plan", label, key.String())
		}
		if _, duplicate := bindings[key]; duplicate {
			t.Fatalf("%s repeats candidate binding %s", label, key.String())
		}
		bindings[key] = binding
	}
	entries := make(map[domain.CandidateExecutionKey]compare.Entry, len(outcome.Entries()))
	for _, entry := range outcome.Entries() {
		entries[entry.CandidateKey] = entry
	}
	exclusions := make(map[domain.CandidateExecutionKey]compare.Exclusion, len(outcome.Exclusions()))
	for _, exclusion := range outcome.Exclusions() {
		exclusions[exclusion.CandidateKey] = exclusion
	}
	expectedWorlds := make([]domain.Digest, 0, len(result.Trials))
	expectedAttempts := make([]domain.Digest, 0, len(result.Trials))
	expectedObservations := make([]domain.Digest, 0, len(result.Trials))
	for _, trial := range result.Trials {
		key := trial.Slot.CandidateKey()
		role, hasRole := result.CandidateRoles[key]
		_, hasBinding := bindings[key]
		world := trial.Result.World()
		attempt := trial.Result.FinalizedAttempt()
		if !trial.Admitted || !hasRole || !hasBinding || role != trial.Role ||
			world.CandidateKey() != key || world.PlanDigest() != result.Plan.Digest() ||
			world.StimulusDigest() != result.Stimulus.Digest() || world.EnvelopeDigest() != result.Envelope.Digest() ||
			world.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
			world.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
			world.ScheduleOrdinal() != trial.Slot.Ordinal() || world.AttemptArtifactDigest() != attempt.ArtifactDigest() ||
			trial.Measurements.SubjectDigest() != world.Digest() || trial.Observation.WorldDigest() != world.Digest() ||
			trial.Observation.PlanDigest() != result.Plan.Digest() || trial.Observation.CandidateKey() != key ||
			trial.Observation.StimulusDigest() != result.Stimulus.Digest() ||
			trial.Observation.AttemptDigest() != attempt.ArtifactDigest() || trial.Observation.TrialIndex() != trial.Slot.Ordinal() ||
			trial.Observation.CapturePolicyDigest() != result.CapturePolicy.Digest() ||
			trial.Observation.ProjectionDefinitionDigest() != result.ProjectionDefinition.Binding().Digest() {
			t.Fatalf("%s trial %d does not rejoin slot/candidate/binding/role/world/capture authority", label, trial.Slot.Ordinal())
		}
		if trial.Projected {
			derivation := trial.Projection.Derivation()
			replayed, replayErr := result.ProjectionDefinition.Project(trial.Observation)
			fingerprint, err := domain.NewProjectionFingerprint(trial.Projection.ProjectionBytes())
			entry, eligible := entries[key]
			if err != nil || replayErr != nil || trial.ProjectionRejection != nil || !derivation.Valid() ||
				derivation.ObservationDigest() != trial.Observation.Digest() ||
				derivation.DefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
				derivation.AdapterDefinitionDigest() != result.ProjectionDefinition.Digest() ||
				replayed.DerivationDigest() != trial.Projection.DerivationDigest() ||
				!bytes.Equal(replayed.CanonicalBytes(), trial.Projection.CanonicalBytes()) ||
				!bytes.Equal(replayed.ProjectionBytes(), trial.Projection.ProjectionBytes()) ||
				!eligible || entry.ProjectionFingerprint != fingerprint {
				t.Fatalf("%s projected trial %d does not rejoin its labeled map entry: projection=%v replay=%v", label, trial.Slot.Ordinal(), err, replayErr)
			}
		} else if primary, controlled := attempt.PrimaryControl(); controlled {
			exclusion, excluded := exclusions[key]
			if label != "eligibility drift" || primary != domain.ControlOutputLimit ||
				trial.ProjectionRejection != nil || trial.Observation.ProjectionEligible() ||
				!excluded || exclusion.Classification != observe.Uncomparable {
				t.Fatalf("%s controlled trial %d is not an exact output-limit map exclusion", label, trial.Slot.Ordinal())
			}
		} else {
			rejection := trial.ProjectionRejection
			_, replayErr := result.ProjectionDefinition.Project(trial.Observation)
			var replayed *countercli.ProjectionRejection
			if rejection == nil || !errors.As(replayErr, &replayed) || !rejection.Valid() || !replayed.Valid() ||
				rejection.ObservationDigest() != trial.Observation.Digest() ||
				rejection.DefinitionDigest() != result.ProjectionDefinition.Binding().Digest() ||
				replayed.EvidenceDigest() != rejection.EvidenceDigest() ||
				!bytes.Equal(replayed.CanonicalBytes(), rejection.CanonicalBytes()) {
				t.Fatalf("%s excluded trial %d lacks typed projection-rejection lineage: %v", label, trial.Slot.Ordinal(), replayErr)
			}
			if _, excluded := exclusions[key]; !excluded {
				t.Fatalf("%s excluded trial %d has no exact map exclusion", label, trial.Slot.Ordinal())
			}
		}
		expectedWorlds = append(expectedWorlds, world.Digest())
		expectedAttempts = append(expectedAttempts, attempt.ArtifactDigest())
		if trial.Projected || trial.ProjectionRejection != nil {
			expectedObservations = append(expectedObservations, trial.Observation.Digest())
		}
	}
	if !sameTestDigestSet(expectedWorlds, outcome.EvidenceWorldDigests()) ||
		!sameTestDigestSet(expectedAttempts, outcome.EvidenceAttemptDigests()) ||
		!sameTestDigestSet(expectedObservations, outcome.EvidenceObservationDigests()) {
		t.Fatalf("%s outcome map evidence sets differ from the exact retained admitted trials", label)
	}
}

func assertFreshDigestSet(t testing.TB, label string, values []domain.Digest, seen map[domain.Digest]string) {
	t.Helper()
	for _, value := range values {
		if prior, duplicate := seen[value]; duplicate {
			t.Fatalf("%s reused authority from %s: %s", label, prior, value)
		}
		seen[value] = label
	}
}

func sameTestDigestSet(left, right []domain.Digest) bool {
	if len(left) != len(right) {
		return false
	}
	want := make(map[domain.Digest]int, len(left))
	for _, value := range left {
		want[value]++
	}
	for _, value := range right {
		want[value]--
		if want[value] < 0 {
			return false
		}
	}
	for _, count := range want {
		if count != 0 {
			return false
		}
	}
	return true
}

func projectedModeAndSource(projection countercli.CLIProjectionResult) (string, string, bool) {
	fields := projection.Fields()
	if len(fields) != 3 || fields[0].ID() != countercli.CLIFieldExitCode ||
		fields[1].ID() != countercli.CLIFieldStdoutJSONMode || fields[2].ID() != countercli.CLIFieldStdoutJSONSource {
		return "", "", false
	}
	exit, hasExit := fields[0].Value().Integer()
	mode, hasMode := fields[1].Value().String()
	source, hasSource := fields[2].Value().String()
	return mode, source, hasExit && exit == 0 && hasMode && hasSource
}

func assertCLIReductionClosure(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	// The exact weak/strong assertions are intentionally bound below to the
	// package-private completion fields, not inferred from presentation rows.
	if !completed.weakRun.Valid() || !completed.weakGrade.Valid() ||
		completed.weakGrade.RunDigest() != completed.weakRun.Digest() ||
		completed.weakGrade.TranscriptDigest() != completed.weakRun.Transcript().Digest() ||
		completed.weakGrade.Grade().Status() != grade.StatusBestKnown ||
		completed.weakRun.DraftGrade() != reducer.GradeBestKnown || !completed.weakRun.HasAcceptedReduction() ||
		completed.weakRun.Transcript().FinalSweepState() != reducer.FinalSweepIncomplete {
		t.Fatal("main reduction did not remain one exact incomplete BEST_KNOWN run")
	}
	if completed.weakRun.Budget().ProposalLimit() != 2 || completed.weakRun.Budget().CandidateTrialLimit() != 10 ||
		completed.baseline.Plan.Budgets().ProposedShrinkStimuli != 2 || completed.baseline.Plan.Budgets().TotalCandidateTrials != 25 {
		t.Fatalf("main reduction budget = proposals:%d trials:%d plan:%+v, want 2/10 from exact total 25",
			completed.weakRun.Budget().ProposalLimit(), completed.weakRun.Budget().CandidateTrialLimit(), completed.baseline.Plan.Budgets())
	}
	rules := completed.weakRun.ReducerSet().Rules()
	if len(rules) != 2 || rules[0].Name() != string(countercli.CLIFixtureRemove) ||
		rules[1].Name() != string(countercli.CLIEnvironmentRemove) {
		t.Fatalf("main reducer priority = %v, want fixture.remove then environment.remove", rules)
	}
	entries := completed.weakRun.Transcript().Entries()
	if len(entries) != 2 || len(completed.weakRun.Transcript().AcceptedPath()) != 1 {
		t.Fatalf("main weak transcript = entries:%d accepted:%d, want exact 2/1",
			len(entries), len(completed.weakRun.Transcript().AcceptedPath()))
	}
	accepted := entries[0].Evaluation()
	unresolved := entries[1].Evaluation()
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: completed.baseline.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIFixtureRemove, countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatalf("reconstruct exact main CLI reduction policy: %v", err)
	}
	firstNeighbors, err := countercli.EnumerateCLINeighbors(completed.baseline.Stimulus, policy)
	if err != nil || len(firstNeighbors) != 2 || firstNeighbors[0].Neighbor().Digest() != accepted.Neighbor().Digest() ||
		firstNeighbors[0].ReplayRecipe().Rule != countercli.CLIFixtureRemove || firstNeighbors[0].ReplayRecipe().Index != 1 ||
		len(completed.baseline.Stimulus.Fixtures()) != 2 || completed.baseline.Stimulus.Fixtures()[1].Path() != "z-main-irrelevant.txt" {
		t.Fatalf("first direct neighbor is not exact z-main-irrelevant.txt removal: neighbors=%d err=%v", len(firstNeighbors), err)
	}
	firstReplay, err := countercli.ReplayCLINeighbor(completed.baseline.Stimulus, policy, firstNeighbors[0].ReplayRecipe())
	if err != nil || firstReplay.Digest() != completed.minimized.Stimulus.Digest() ||
		firstReplay.Digest() != accepted.Neighbor().StimulusDigest() {
		t.Fatalf("fixture-removal recipe did not replay to the retained minimized stimulus: %v", err)
	}
	secondNeighbors, err := countercli.EnumerateCLINeighbors(completed.minimized.Stimulus, policy)
	if err != nil || len(secondNeighbors) != 1 || secondNeighbors[0].Neighbor().Digest() != unresolved.Neighbor().Digest() ||
		secondNeighbors[0].ReplayRecipe().Rule != countercli.CLIEnvironmentRemove || secondNeighbors[0].ReplayRecipe().Index != 0 ||
		len(completed.minimized.Stimulus.Environment()) != 1 || completed.minimized.Stimulus.Environment()[0].Name() != "APP_MODE" {
		t.Fatalf("second direct neighbor is not exact APP_MODE removal: neighbors=%d err=%v", len(secondNeighbors), err)
	}
	secondReplay, err := countercli.ReplayCLINeighbor(completed.minimized.Stimulus, policy, secondNeighbors[0].ReplayRecipe())
	if err != nil || secondReplay.Digest() != unresolved.Neighbor().StimulusDigest() {
		t.Fatalf("APP_MODE-removal recipe did not replay to the unresolved neighbor: %v", err)
	}
	for _, invalidRecipe := range []countercli.CLIReplayRecipe{
		{Rule: countercli.CLIFixtureRemove, Index: 0, Transform: firstNeighbors[0].ReplayRecipe().Transform},
		{Rule: countercli.CLIEnvironmentRemove, Index: 1, Transform: secondNeighbors[0].ReplayRecipe().Transform},
	} {
		if _, replayErr := countercli.ReplayCLINeighbor(completed.minimized.Stimulus, policy, invalidRecipe); replayErr == nil {
			t.Fatalf("wrong-rule/index reduction recipe replayed: %+v", invalidRecipe)
		}
	}
	observedMap, observed := accepted.ObservedOutcomeMapDigest()
	observedPreservation, preservationObserved := accepted.ObservedPreservationDigest()
	if accepted.Purpose() != domain.AttemptReduction || accepted.Decision() != reducer.Preserves ||
		accepted.Neighbor().Rule().Name() != string(countercli.CLIFixtureRemove) || accepted.Neighbor().Locus() != "fixture[1]" ||
		entries[0].ProposalCount() != 1 || entries[0].CandidateTrials() != 9 || entries[0].TrialCount() != 9 ||
		completed.weakRun.Transcript().AcceptedPath()[0] != accepted.Neighbor().Digest() ||
		!observed || observedMap != completed.mainEvaluation.OutcomeMap.ArtifactDigest() ||
		!preservationObserved || observedPreservation != completed.mainEvaluation.OutcomeMap.PreservationDigest() ||
		!sameTestDigestSet(accepted.ObservedAttemptDigests(), completed.mainEvaluation.OutcomeMap.EvidenceAttemptDigests()) ||
		!sameTestDigestSet(accepted.ObservedWorldDigests(), completed.mainEvaluation.OutcomeMap.EvidenceWorldDigests()) ||
		!sameTestDigestSet(accepted.ObservedObservationDigests(), completed.mainEvaluation.OutcomeMap.EvidenceObservationDigests()) {
		t.Fatal("main accepted fixture-removal entry lost its exact 9-trial preservation evidence")
	}
	if unresolved.Purpose() != domain.AttemptReduction || unresolved.Decision() != reducer.Unresolved ||
		unresolved.ReasonCode() != string(reducer.ReasonEvaluatorError) ||
		unresolved.Neighbor().Rule().Name() != string(countercli.CLIEnvironmentRemove) ||
		unresolved.Neighbor().Locus() != "environment[0]" || entries[1].ProposalCount() != 2 ||
		entries[1].CandidateTrials() != 0 || entries[1].TrialCount() != 9 ||
		len(unresolved.ObservedAttemptDigests()) != 0 || len(unresolved.ObservedWorldDigests()) != 0 ||
		len(unresolved.ObservedObservationDigests()) != 0 {
		t.Fatal("main second direct neighbor was not the exact allowance-1 EVALUATOR_ERROR/UNRESOLVED boundary")
	}
	if _, present := unresolved.ObservedOutcomeMapDigest(); present {
		t.Fatal("unresolved evaluator-error neighbor smuggled a product outcome map")
	}
	if _, present := unresolved.ObservedPreservationDigest(); present {
		t.Fatal("unresolved evaluator-error neighbor smuggled a preservation map")
	}
	assertWeakTranscriptForgeryRefusals(t, completed.weakRun, accepted, unresolved)
	if draft, present, err := completed.weakRun.CompletedSweepDraft(); err != nil || present || draft.Valid() {
		t.Fatalf("incomplete main run minted a completed sweep draft: present=%t valid=%t err=%v", present, draft.Valid(), err)
	}
	if !slices.Contains(completed.weakRun.Transcript().Limitations(), "UNRESOLVED_EVALUATOR_ERROR") ||
		!slices.Contains(completed.weakRun.Transcript().Limitations(), "PROPOSAL_BUDGET_EXHAUSTED") {
		t.Fatalf("main weak limitations = %v", completed.weakRun.Transcript().Limitations())
	}
	fixtures := completed.minimized.Stimulus.Fixtures()
	environment := completed.minimized.Stimulus.Environment()
	if len(fixtures) != 1 || fixtures[0].Path() != "config.json" || len(environment) != 1 ||
		environment[0].Name() != "APP_MODE" || !environment[0].Present() || environment[0].Value() != "env" ||
		!slices.Equal(completed.minimized.Stimulus.Argv(), []string{"--mode", "argv"}) {
		t.Fatal("main BEST_KNOWN witness retained irrelevant input or removed a necessary precedence input")
	}
	if !completed.strongRun.Valid() || !completed.strongGrade.Valid() ||
		completed.strongRun.Digest() == completed.weakRun.Digest() ||
		completed.strongGrade.RunDigest() != completed.strongRun.Digest() ||
		completed.strongGrade.Grade().Status() != grade.StatusOneMinimalUnder ||
		completed.strongRun.DraftGrade() != reducer.GradeBestKnown ||
		completed.strongRun.Transcript().FinalSweepState() != reducer.FinalSweepComplete ||
		completed.strongRun.Budget().ProposalLimit() != 2 || completed.strongRun.Budget().CandidateTrialLimit() != 2 ||
		completed.auxiliaryBaseline.Plan.Budgets().ProposedShrinkStimuli != 2 ||
		completed.auxiliaryBaseline.Plan.Budgets().TotalCandidateTrials != 6 {
		t.Fatal("auxiliary reduction is not the sole distinct durable ONE_MINIMAL_UNDER control")
	}
	strongRules := completed.strongRun.ReducerSet().Rules()
	strongEntries := completed.strongRun.Transcript().Entries()
	if len(strongRules) != 1 || strongRules[0].Name() != string(countercli.CLIFixtureRemove) ||
		len(strongEntries) != 1 || strongEntries[0].Evaluation().Decision() != reducer.Preserves ||
		strongEntries[0].Evaluation().Neighbor().Rule().Name() != string(countercli.CLIFixtureRemove) ||
		strongEntries[0].Evaluation().Neighbor().Locus() != "fixture[1]" ||
		strongEntries[0].CandidateTrials() != 2 || len(completed.strongRun.Transcript().AcceptedPath()) != 1 ||
		len(completed.strongRun.Transcript().FinalNeighborDigests()) != 0 ||
		len(completed.strongRun.Transcript().Limitations()) != 0 ||
		len(completed.auxiliaryBaseline.Stimulus.Fixtures()) != 2 ||
		completed.auxiliaryBaseline.Stimulus.Fixtures()[1].Path() != "z-auxiliary-irrelevant.txt" ||
		len(completed.auxiliaryMinimized.Stimulus.Fixtures()) != 1 ||
		completed.auxiliaryMinimized.Stimulus.Fixtures()[0].Path() != "config.json" {
		t.Fatal("auxiliary fixture-only reduction lost its exact accepted zero-neighbor sweep")
	}
	draft, present, err := completed.strongRun.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 ||
		draft.RunDigest() != completed.strongRun.Digest() {
		t.Fatalf("auxiliary zero-neighbor sweep draft = present:%t valid:%t neighbors:%d err:%v",
			present, draft.Valid(), len(draft.NeighborDigests()), err)
	}
	if completed.confirmed.Confirmation.Draft().Record().ReductionRunDigest() != completed.weakRun.Digest() ||
		completed.confirmed.Confirmation.Draft().Record().ReductionGradeDigest() != completed.weakGrade.Grade().Digest() ||
		completed.auxiliaryConfirmed.Confirmation.Draft().Record().ReductionRunDigest() != completed.strongRun.Digest() ||
		completed.auxiliaryConfirmed.Confirmation.Draft().Record().ReductionGradeDigest() != completed.strongGrade.Grade().Digest() {
		t.Fatal("fresh confirmations do not bind the exact weak-main and strong-auxiliary reduction authorities")
	}
}

func assertWeakTranscriptForgeryRefusals(
	t testing.TB,
	run reducer.ReductionRun,
	accepted, unresolved reducer.Evaluation,
) {
	t.Helper()
	original := run.Transcript().CanonicalBytes()
	mutations := []struct {
		name string
		old  string
		new  string
	}{
		{"wrong fixture rule", `"rule_name":"cli.fixture.remove"`, `"rule_name":"cli.argv.remove"`},
		{"wrong fixture index", `"locus":"fixture[1]"`, `"locus":"fixture[0]"`},
		{"wrong unresolved rule", `"rule_name":"cli.environment.remove"`, `"rule_name":"cli.argv.remove"`},
		{"wrong unresolved index", `"locus":"environment[0]"`, `"locus":"environment[1]"`},
		{"same decision wrong reason", `"reason_code":"EVALUATOR_ERROR"`, `"reason_code":"TIMEOUT"`},
		{"wrong accepted proposal digest", accepted.Neighbor().Digest().String(), unresolved.Neighbor().Digest().String()},
	}
	for _, mutation := range mutations {
		forged := bytes.Replace(original, []byte(mutation.old), []byte(mutation.new), 1)
		if bytes.Equal(forged, original) {
			t.Fatalf("%s mutation did not reach exact weak transcript", mutation.name)
		}
		if _, err := reducer.ParseReductionRunRecord(run.CanonicalBytes(), forged); err == nil {
			t.Fatalf("reduction record accepted %s with unchanged decision presentation", mutation.name)
		}
	}
}

func assertPrepublicationInspection(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	inspection, err := Inspect(completed)
	if err != nil || !inspection.Valid() || inspection.Published() {
		t.Fatalf("inspect completed study before publication: valid=%t published=%t err=%v",
			inspection.Valid(), inspection.Published(), err)
	}
	if inspection.Ordinal() != completed.ordinal || inspection.PhysicalRunAuthority() != completed.physicalRunAuthority.String() ||
		len(inspection.Manifest()) != 0 || inspection.ManifestSHA256() != "" ||
		len(inspection.DeterministicSHA256()) != 0 || len(inspection.FreshSHA256()) != 0 {
		t.Fatal("pre-publication inspection smuggled terminal publication facts or lost its completed-run binding")
	}

	artifacts := inspection.Artifacts()
	wantArtifacts := []struct {
		name      string
		authority string
		canonical []byte
	}{
		{name: "source-spec", authority: completed.confirmed.SourceSpecDigest.String(), canonical: completed.confirmed.SourceSpecBytes},
		{name: "world-plan", authority: completed.confirmed.Plan.Digest().String(), canonical: completed.confirmed.Plan.CanonicalBytes()},
		{name: "ruling", authority: completed.decision.Digest().String(), canonical: completed.durableRuling.Record().CanonicalBytes()},
		{name: "decision-record", authority: completed.decision.Digest().String(), canonical: completed.decision.CanonicalBytes()},
		{name: "contract-bundle", authority: completed.bundle.Digest().String(), canonical: completed.bundle.CanonicalBytes()},
	}
	if len(artifacts) != len(wantArtifacts) {
		t.Fatalf("inspection artifact count = %d, want %d", len(artifacts), len(wantArtifacts))
	}
	for index, want := range wantArtifacts {
		fact := artifacts[index]
		if fact.Name() != want.name || fact.Authority() != want.authority ||
			!bytes.Equal(fact.CanonicalBytes(), want.canonical) {
			t.Fatalf("inspection artifact %d does not rejoin %s authority", index, want.name)
		}
	}

	phaseFacts := inspection.PhaseFacts()
	if len(phaseFacts) != len(completed.phaseFacts) {
		t.Fatalf("inspection phase count = %d, want %d", len(phaseFacts), len(completed.phaseFacts))
	}
	for index, fact := range phaseFacts {
		owned := completed.phaseFacts[index]
		if fact.Phase() != owned.phase || fact.Trial() != owned.trial || fact.Kind() != owned.kind ||
			fact.OwnerAuthority() != owned.authority.String() || fact.WorldAuthority() != owned.world.String() ||
			fact.AttemptAuthority() != owned.attempt.String() || fact.MeasurementAuthority() != owned.measurement.String() ||
			fact.CaptureAuthority() != owned.capture.String() || fact.ProjectionAuthority() != owned.projection.String() {
			t.Fatalf("inspection phase fact %d does not rejoin its retained owner", index)
		}
	}

	officialFacts := inspection.OfficialFacts()
	if len(officialFacts) != len(completed.officialTrials) {
		t.Fatalf("inspection official fact count = %d, want %d", len(officialFacts), len(completed.officialTrials))
	}
	for index, fact := range officialFacts {
		trial := completed.officialTrials[index]
		if fact.Trial() != index+1 || fact.AttemptAuthority() != trial.attemptDigest.String() ||
			fact.TargetAuthority() != trial.targetDigest.String() || fact.TargetCanonicalSHA256() != trial.targetCanonicalSHA256 ||
			fact.BundleAuthority() != trial.bundleDigest.String() || fact.ResidueHeadAuthority() != trial.residueHeadDigest.String() ||
			fact.RunAuthority() != trial.runDigest.String() || fact.RunCanonicalSHA256() != trial.runCanonicalSHA256 ||
			fact.ClassificationAuthority() != trial.classificationDigest.String() ||
			fact.ClassificationCanonicalSHA256() != trial.classificationCanonicalSHA256 ||
			fact.Result() != trial.result || fact.ExitCode() != trial.exitCode || !fact.RecoveryEqual() {
			t.Fatalf("inspection official fact %d does not rejoin its exact target/run/classification graph", index+1)
		}
	}
	reductionFacts := inspection.ReductionFacts()
	if len(reductionFacts) != 2 {
		t.Fatalf("inspection reduction fact count = %d, want exact weak/strong pair", len(reductionFacts))
	}
	strongSweep, strongSweepPresent := completed.strongGrade.Grade().CompletedSweepDigest()
	if !strongSweepPresent {
		t.Fatal("retained auxiliary strong grade lacks durable completed-sweep authority")
	}
	wantReductions := []struct {
		name, run, transcript, grade, status, reducerSet, minimized, sweepState, completedSweep string
		limitations                                                                             []string
	}{
		{
			name: "main-weak", run: completed.weakRun.Digest().String(), transcript: completed.weakRun.Transcript().Digest().String(),
			grade: completed.weakGrade.Grade().Digest().String(), status: string(completed.weakGrade.Grade().Status()),
			reducerSet: completed.weakRun.ReducerSet().Digest().String(), minimized: completed.weakRun.MinimizedStimulusDigest().String(),
			sweepState: string(completed.weakRun.Transcript().FinalSweepState()), limitations: completed.weakGrade.Grade().Limitations(),
		},
		{
			name: "auxiliary-strong", run: completed.strongRun.Digest().String(), transcript: completed.strongRun.Transcript().Digest().String(),
			grade: completed.strongGrade.Grade().Digest().String(), status: string(completed.strongGrade.Grade().Status()),
			reducerSet: completed.strongRun.ReducerSet().Digest().String(), minimized: completed.strongRun.MinimizedStimulusDigest().String(),
			sweepState: string(completed.strongRun.Transcript().FinalSweepState()), completedSweep: strongSweep.String(),
			limitations: completed.strongGrade.Grade().Limitations(),
		},
	}
	for index, want := range wantReductions {
		fact := reductionFacts[index]
		if fact.Name() != want.name || fact.RunAuthority() != want.run || fact.TranscriptAuthority() != want.transcript ||
			fact.GradeAuthority() != want.grade || fact.Status() != want.status ||
			fact.ReducerSetAuthority() != want.reducerSet || fact.MinimizedStimulusAuthority() != want.minimized ||
			fact.FinalSweepState() != want.sweepState || fact.CompletedSweepAuthority() != want.completedSweep ||
			!slices.Equal(fact.GradeLimitations(), want.limitations) {
			t.Fatalf("inspection reduction fact %d does not rejoin its exact retained reduction authority", index)
		}
	}
	if reductionFacts[0].RunAuthority() == reductionFacts[1].RunAuthority() ||
		reductionFacts[0].TranscriptAuthority() == reductionFacts[1].TranscriptAuthority() ||
		reductionFacts[0].GradeAuthority() == reductionFacts[1].GradeAuthority() {
		t.Fatal("inspection collapsed weak and strong reduction authority domains")
	}
	if reductionFacts[0].MinimizedStimulusAuthority() != reductionFacts[1].MinimizedStimulusAuthority() ||
		reductionFacts[0].MinimizedStimulusAuthority() != completed.discovery.Stimulus.Digest().String() {
		t.Fatal("inspection lost the exact shared minimized-witness convergence")
	}

	wantNonclaims := []string{
		"BROAD_IMPORTED_REPOSITORY_BEHAVIOR_UNVALIDATED",
		"CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED",
		"FULL_STUDY_RESOURCE_BOUND_UNVALIDATED",
		"FULL_WORKFLOW_RACE_FREEDOM_UNVALIDATED",
		"HOSTILE_CONTAINMENT_UNVALIDATED",
		"NETWORK_DENIAL_UNVALIDATED",
		"POST_SPAWN_OPERATOR_FAILURE_TEXT_UNVALIDATED",
		"RUNTIME_ESCAPED_WRITE_BEHAVIOR_UNPROVEN",
	}
	if !slices.Equal(inspection.Nonclaims(), wantNonclaims) {
		t.Fatalf("inspection nonclaims = %v, want exact bounded roster %v", inspection.Nonclaims(), wantNonclaims)
	}

	artifactBytes := artifacts[0].CanonicalBytes()
	artifactBytes[0] ^= 0xff
	artifacts[0].canonical[0] ^= 0xff
	phaseFacts[0].kind = "forged"
	officialFacts[0].result = "forged"
	reductionFacts[0].name = "forged"
	reductionFacts[0].gradeLimitations[0] = "forged"
	nonclaims := inspection.Nonclaims()
	nonclaims[0] = "forged"
	if !inspection.Valid() || inspection.Published() ||
		bytes.Equal(inspection.Artifacts()[0].CanonicalBytes(), artifactBytes) ||
		inspection.PhaseFacts()[0].Kind() == "forged" || inspection.OfficialFacts()[0].Result() == "forged" ||
		inspection.ReductionFacts()[0].Name() == "forged" || inspection.ReductionFacts()[0].GradeLimitations()[0] == "forged" ||
		inspection.Nonclaims()[0] == "forged" {
		t.Fatal("inspection getters exposed mutable semantic or publication authority")
	}
}

func assertCompletedValidationAndInspectionArePure(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	poisoned := completed
	poisoned.officialTrials = append([]officialTrial(nil), completed.officialTrials...)
	liveCalls := 0
	for index := range poisoned.officialTrials {
		poisoned.officialTrials[index].revalidate = func(context.Context, officialTrial) error {
			liveCalls++
			return errors.New("pure completed validation reached live official revalidation")
		}
	}
	if !poisoned.Valid() {
		t.Fatal("completed Valid rejected an otherwise exact study after replacing only live revalidators")
	}
	inspection, err := Inspect(poisoned)
	if err != nil || !inspection.Valid() || inspection.Published() {
		t.Fatalf("pure Inspect with poisoned live revalidators: valid=%t published=%t err=%v",
			inspection.Valid(), inspection.Published(), err)
	}
	if liveCalls != 0 {
		t.Fatalf("Completed.Valid or Inspect performed %d live official revalidations", liveCalls)
	}
}

func assertCompletedPhaseRoster(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	if len(completed.phaseFacts) != 98 || len(completed.officialTrials) != 10 {
		t.Fatalf("completed phase roster = facts:%d contract-graphs:%d, want 98/10",
			len(completed.phaseFacts), len(completed.officialTrials))
	}
	seen := make(map[string]string, 98)
	seenWorlds := make(map[domain.Digest]string, 88)
	seenAttempts := make(map[domain.Digest]string, 98)
	seenMeasurements := make(map[domain.Digest]string, 88)
	seenCaptures := make(map[domain.Digest]string, 88)
	seenProjections := make(map[domain.Digest]string, 80)
	missingProjections := make(map[string]int, 3)
	phaseCounts := map[string]int{"search": 0, "confirm": 0, "contract": 0}
	kindCounts := make(map[string]int, 29)
	confirmationOwners := append(completed.confirmed.Confirmation.Draft().PhysicalFacts(),
		completed.auxiliaryConfirmed.Confirmation.Draft().PhysicalFacts()...)
	if len(confirmationOwners) != 8 {
		t.Fatalf("retained confirmation owner count = %d, want 8", len(confirmationOwners))
	}
	for index, fact := range completed.phaseFacts {
		wantPhase, wantTrial := "contract", index-88+1
		if index < 80 {
			wantPhase, wantTrial = "search", index+1
		} else if index < 88 {
			wantPhase, wantTrial = "confirm", index-80+1
		}
		if fact.phase != wantPhase || fact.trial != wantTrial || fact.kind == "" {
			t.Fatalf("phase fact %d = phase:%q trial:%d kind:%q, want %s/%d with an explicit kind",
				index, fact.phase, fact.trial, fact.kind, wantPhase, wantTrial)
		}
		digest := fact.authority.String()
		if !strings.HasPrefix(digest, "sha256:") || !isLowerSHA256(strings.TrimPrefix(digest, "sha256:")) {
			t.Fatalf("%s authority %d is not a typed digest: %q", fact.phase, fact.trial, digest)
		}
		if prior, duplicate := seen[digest]; duplicate {
			t.Fatalf("%s authority %d aliases %s at %q", fact.phase, fact.trial, prior, digest)
		}
		seen[digest] = fact.phase
		phaseCounts[fact.phase]++
		kindCounts[fact.phase+"\x00"+fact.kind]++
		switch fact.phase {
		case "search":
			if fact.authority != fact.attempt || !fact.world.Valid() || !fact.attempt.Valid() ||
				!fact.measurement.Valid() || !fact.capture.Valid() {
				t.Fatalf("search phase fact %d is not owned by its retained physical attempt", fact.trial)
			}
			if !fact.projection.Valid() {
				missingProjections[fact.kind]++
				if fact.kind != "MAIN_ELIGIBILITY_DRIFT_WORLD" && fact.kind != "CONTROL_TIMEOUT" && fact.kind != "CONTROL_OUTPUT_LIMIT" {
					t.Fatalf("search phase fact %d unexpectedly lacks projected/rejection authority for %s", fact.trial, fact.kind)
				}
			}
			assertFreshPhaseComponent(t, fact, "world", fact.world, seenWorlds)
			assertFreshPhaseComponent(t, fact, "attempt", fact.attempt, seenAttempts)
			assertFreshPhaseComponent(t, fact, "measurement", fact.measurement, seenMeasurements)
			assertFreshPhaseComponent(t, fact, "capture", fact.capture, seenCaptures)
			if fact.projection.Valid() {
				assertFreshPhaseComponent(t, fact, "projection", fact.projection, seenProjections)
			}
		case "confirm":
			owner := confirmationOwners[fact.trial-1]
			if fact.authority != owner.Digest() || fact.world != owner.WorldDigest() || fact.attempt != owner.AttemptArtifactDigest() ||
				!fact.measurement.Valid() || !fact.capture.Valid() || !fact.projection.Valid() {
				t.Fatalf("confirm phase fact %d does not rejoin its FreshConfirmation physical fact", fact.trial)
			}
			assertFreshPhaseComponent(t, fact, "world", fact.world, seenWorlds)
			assertFreshPhaseComponent(t, fact, "attempt", fact.attempt, seenAttempts)
			assertFreshPhaseComponent(t, fact, "measurement", fact.measurement, seenMeasurements)
			assertFreshPhaseComponent(t, fact, "capture", fact.capture, seenCaptures)
			assertFreshPhaseComponent(t, fact, "projection", fact.projection, seenProjections)
		case "contract":
			if fact.authority != completed.officialTrials[fact.trial-1].classificationDigest {
				t.Fatalf("contract phase fact %d does not belong to its exact classification graph", fact.trial)
			}
			if fact.attempt != completed.officialTrials[fact.trial-1].attemptDigest {
				t.Fatalf("contract phase fact %d does not rejoin its exact attempt", fact.trial)
			}
			assertFreshPhaseComponent(t, fact, "attempt", fact.attempt, seenAttempts)
		}
	}
	if len(seen) != 98 || phaseCounts["search"] != 80 || phaseCounts["confirm"] != 8 || phaseCounts["contract"] != 10 {
		t.Fatalf("distinct phase roster = owners:%d phases:%v, want 98 and 80/8/10", len(seen), phaseCounts)
	}
	if len(seenWorlds) != 88 || len(seenAttempts) != 98 || len(seenMeasurements) != 88 ||
		len(seenCaptures) != 88 || len(seenProjections) != 80 ||
		missingProjections["MAIN_ELIGIBILITY_DRIFT_WORLD"] != 6 ||
		missingProjections["CONTROL_TIMEOUT"] != 1 || missingProjections["CONTROL_OUTPUT_LIMIT"] != 1 ||
		len(missingProjections) != 3 {
		t.Fatalf("physical component roster = worlds:%d attempts:%d measurements:%d captures:%d projections:%d missing:%v, want 88/98/88/88/80 with exact 6+1+1 lifecycle gaps",
			len(seenWorlds), len(seenAttempts), len(seenMeasurements), len(seenCaptures), len(seenProjections), missingProjections)
	}
	wantKinds := map[string]int{
		"search\x00MAIN_DECISIVE_WORLD":                         9,
		"search\x00MAIN_ELIGIBILITY_REFERENCE_WORLD":            9,
		"search\x00MAIN_ELIGIBILITY_DRIFT_WORLD":                9,
		"search\x00MAIN_SHAPE_REFERENCE_WORLD":                  3,
		"search\x00MAIN_SHAPE_CHANGED_WORLD":                    3,
		"search\x00MAIN_BASELINE_CHECKPOINT_WORLD":              9,
		"search\x00MAIN_DIVERGENT_BASELINE_WORLD":               9,
		"search\x00MAIN_ACCEPTED_REDUCTION_WORLD":               9,
		"search\x00AUXILIARY_BASELINE_WORLD":                    2,
		"search\x00AUXILIARY_PRESERVING_EVALUATION_WORLD":       2,
		"search\x00CONTROL_STDIN_ABSENT":                        1,
		"search\x00CONTROL_STDIN_PRESENT_EMPTY":                 1,
		"search\x00CONTROL_APP_MODE_ABSENT":                     1,
		"search\x00CONTROL_APP_MODE_PRESENT_EMPTY":              1,
		"search\x00CONTROL_ORDERED_ARGV_MODE_THEN_STDERR":       1,
		"search\x00CONTROL_ORDERED_ARGV_STDERR_THEN_MODE":       1,
		"search\x00CONTROL_SPARSE_IRRELEVANT_ENV_ALPHA":         1,
		"search\x00CONTROL_SPARSE_IRRELEVANT_ENV_OMEGA":         1,
		"search\x00CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION":    1,
		"search\x00CONTROL_UNSELECTED_STDERR_ALPHA":             1,
		"search\x00CONTROL_UNSELECTED_STDERR_BETA":              1,
		"search\x00CONTROL_EXIT_SEVEN_SUBSTRATE":                1,
		"search\x00CONTROL_SIGNAL_SELECTED_EXIT_MISSING":        1,
		"search\x00CONTROL_TIMEOUT":                             1,
		"search\x00CONTROL_MALFORMED_SELECTED_JSON":             1,
		"search\x00CONTROL_OUTPUT_LIMIT":                        1,
		"confirm\x00MAIN_CONFIRMATION_FACT":                     6,
		"confirm\x00AUXILIARY_CONFIRMATION_FACT":                2,
		"contract\x00OFFICIAL_DEFAULT_EXIT_ZERO_CLASSIFICATION": 10,
	}
	if len(kindCounts) != len(wantKinds) {
		t.Fatalf("phase kind roster has %d labels, want exact %d: %v", len(kindCounts), len(wantKinds), kindCounts)
	}
	for kind, want := range wantKinds {
		if kindCounts[kind] != want {
			t.Fatalf("phase kind %q count = %d, want %d (full=%v)", kind, kindCounts[kind], want, kindCounts)
		}
	}
	if _, aliasesPhase := seen[completed.physicalRunAuthority.String()]; aliasesPhase {
		t.Fatal("completed-run authority was reused as a phase-owner digest")
	}
	assertPhysicalRunBindsPhaseComponents(t, completed)
}

func assertFreshPhaseComponent(t testing.TB, fact phaseFact, component string, value domain.Digest, seen map[domain.Digest]string) {
	t.Helper()
	label := fact.phase + ":" + fact.kind
	if prior, duplicate := seen[value]; duplicate {
		t.Fatalf("%s trial %d reused %s component from %s: %s", fact.phase, fact.trial, component, prior, value)
	}
	seen[value] = label
}

func assertPhysicalRunBindsPhaseComponents(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	mutations := []struct {
		name   string
		mutate func(*phaseFact, phaseFact)
	}{
		{"phase", func(fact *phaseFact, _ phaseFact) { fact.phase = "confirm" }},
		{"trial", func(fact *phaseFact, _ phaseFact) { fact.trial++ }},
		{"kind", func(fact *phaseFact, _ phaseFact) { fact.kind = "MAIN_ELIGIBILITY_REFERENCE_WORLD" }},
		{"world", func(fact *phaseFact, other phaseFact) { fact.world = other.world }},
		{"attempt", func(fact *phaseFact, other phaseFact) { fact.attempt = other.attempt }},
		{"measurement", func(fact *phaseFact, other phaseFact) { fact.measurement = other.measurement }},
		{"capture", func(fact *phaseFact, other phaseFact) { fact.capture = other.capture }},
		{"projection", func(fact *phaseFact, other phaseFact) { fact.projection = other.projection }},
	}
	for _, mutation := range mutations {
		forged := completed
		forged.phaseFacts = append([]phaseFact(nil), completed.phaseFacts...)
		mutation.mutate(&forged.phaseFacts[0], forged.phaseFacts[9])
		authority, err := physicalRunAuthority(forged)
		if err == nil && authority == completed.physicalRunAuthority {
			t.Fatalf("completed physical-run authority did not bind phase %s: authority=%s err=%v",
				mutation.name, authority, err)
		}
	}
}

func rebuildTestPhaseFacts(completed *CompletedCLIStudy) error {
	facts, err := buildPhaseFacts(
		completed.discovery, completed.recoveryOne, completed.recoveryTwo,
		completed.shapeReference, completed.shapeChanged, completed.baselineCheckpoint, completed.baseline,
		completed.mainEvaluation, completed.auxiliaryBaseline, completed.auxiliaryEvaluation,
		completed.semanticControls, completed.confirmed, completed.auxiliaryConfirmed, completed.officialTrials,
	)
	if err != nil {
		return err
	}
	completed.phaseFacts = facts
	return nil
}

func relabelTestControlSubgraphs(completed *CompletedCLIStudy, left, right int) {
	leftKind := completed.semanticControls[left].kind
	rightKind := completed.semanticControls[right].kind
	completed.semanticControls[left], completed.semanticControls[right] =
		completed.semanticControls[right], completed.semanticControls[left]
	completed.semanticControls[left].kind = leftKind
	completed.semanticControls[right].kind = rightKind
	_ = rebuildTestPhaseFacts(completed)
}

func assertCompletedRosterForgeryRefusals(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	assertSourceSpecAuthorityHostiles(t, completed)
	foreignPlanSourceDigest, foreignPlanSourceBytes := canonicalSourceSpecForDifferentPlan(t, completed.confirmed)
	foreignChoicepoint := auxiliaryChoicepointForSubstitution(t, completed)
	if !foreignChoicepoint.Valid() || foreignChoicepoint.Digest() == completed.choicepoint.Digest() ||
		foreignChoicepoint.ConfirmationDigest() == completed.choicepoint.ConfirmationDigest() {
		t.Fatal("auxiliary cohort did not produce a distinct valid substitution Choicepoint")
	}
	targetControl := completed.semanticControls[13]
	sourceControl := completed.semanticControls[15]
	if targetControl.definition.Digest() != sourceControl.definition.Digest() ||
		targetControl.definition.Binding().Digest() != sourceControl.definition.Binding().Digest() ||
		!bytes.Equal(targetControl.definition.CanonicalBytes(), sourceControl.definition.CanonicalBytes()) ||
		targetControl.observation.Digest() == sourceControl.observation.Digest() {
		t.Fatal("same-definition controlled observations are not a valid foreign-rejection test pair")
	}
	_, foreignRejectionErr := targetControl.definition.Project(sourceControl.observation)
	var foreignRejection *countercli.ProjectionRejection
	if !errors.As(foreignRejectionErr, &foreignRejection) || foreignRejection.Code != countercli.CodeProjectionControl ||
		!foreignRejection.Valid() || foreignRejection.DefinitionDigest() != targetControl.definition.Binding().Digest() ||
		foreignRejection.ObservationDigest() != sourceControl.observation.Digest() ||
		foreignRejection.ObservationDigest() == targetControl.observation.Digest() {
		t.Fatalf("construct valid same-definition foreign rejection control: %v", foreignRejectionErr)
	}
	_, bridgeErr := countercli.PrepareProjectionRejectionEvidence(targetControl.observation, foreignRejection)
	var bridgeRefusal *countercli.Refusal
	if !errors.As(bridgeErr, &bridgeRefusal) || bridgeRefusal.Code != countercli.CodeCaptureLineageMismatch {
		t.Fatalf("cross-observation rejection bridge = %v, want exact capture-lineage refusal", bridgeErr)
	}
	if lineageErr := validateProjectionLineage(
		targetControl.definition, targetControl.observation, false,
		countercli.CLIProjectionResult{}, foreignRejection,
	); lineageErr == nil || !strings.Contains(lineageErr.Error(), "CLI_STUDY_REJECTION_LINEAGE_REFUSED") {
		t.Fatalf("cross-observation rejection validation = %v, want exact lineage refusal", lineageErr)
	}
	clone := func() CompletedCLIStudy {
		forged := completed
		forged.phaseFacts = append([]phaseFact(nil), completed.phaseFacts...)
		forged.officialTrials = append([]officialTrial(nil), completed.officialTrials...)
		forged.discovery.Trials = append([]Trial(nil), completed.discovery.Trials...)
		forged.recoveryTwo.Trials = append([]Trial(nil), completed.recoveryTwo.Trials...)
		forged.confirmed.Trials = append([]Trial(nil), completed.confirmed.Trials...)
		forged.semanticControls = append([]physicalControl(nil), completed.semanticControls...)
		forged.choiceAudit.actions = append([]choiceActionAudit(nil), completed.choiceAudit.actions...)
		for index := range forged.choiceAudit.actions {
			forged.choiceAudit.actions[index].destinationBefore = append([]string(nil), completed.choiceAudit.actions[index].destinationBefore...)
			forged.choiceAudit.actions[index].destinationAfter = append([]string(nil), completed.choiceAudit.actions[index].destinationAfter...)
		}
		return forged
	}
	tests := []struct {
		name   string
		mutate func(*CompletedCLIStudy)
	}{
		{name: "dropped-phase-authority", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts = value.phaseFacts[:len(value.phaseFacts)-1]
		}},
		{name: "unlabeled-digest-padding", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts = append(value.phaseFacts, phaseFact{authority: value.physicalRunAuthority})
		}},
		{name: "duplicate-phase-owner", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[1].authority = value.phaseFacts[0].authority
		}},
		{name: "cross-phase-owner", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[80].authority = value.phaseFacts[0].authority
		}},
		{name: "cross-phase-relabel", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[80].phase = "search"
		}},
		{name: "unowned-authority", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[0].authority = value.physicalRunAuthority
		}},
		{name: "unlabeled-owner", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[0].kind = ""
		}},
		{name: "cross-kind-relabel", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[0].kind = "MAIN_ELIGIBILITY_DRIFT_WORLD"
		}},
		{name: "phase-ordinal-drift", mutate: func(value *CompletedCLIStudy) {
			value.phaseFacts[1].trial = value.phaseFacts[0].trial
		}},
		{name: "nonconfirmation-whole-trial-row-reorder", mutate: func(value *CompletedCLIStudy) {
			value.discovery.Trials[0], value.discovery.Trials[1] = value.discovery.Trials[1], value.discovery.Trials[0]
			_ = rebuildTestPhaseFacts(value)
		}},
		{name: "confirmation-whole-trial-row-reorder", mutate: func(value *CompletedCLIStudy) {
			value.confirmed.Trials[0], value.confirmed.Trials[1] = value.confirmed.Trials[1], value.confirmed.Trials[0]
			_ = rebuildTestPhaseFacts(value)
		}},
		{name: "nonconfirmation-observation-run-swap", mutate: func(value *CompletedCLIStudy) {
			value.baselineCheckpoint.Observation = value.baseline.Observation
		}},
		{name: "same-role-same-bytes-projection-swap", mutate: func(value *CompletedCLIStudy) {
			first, second := -1, -1
			for index, trial := range value.discovery.Trials {
				if first < 0 {
					first = index
					continue
				}
				if trial.Role == value.discovery.Trials[first].Role {
					second = index
					break
				}
			}
			if first >= 0 && second >= 0 &&
				bytes.Equal(value.discovery.Trials[first].Projection.ProjectionBytes(), value.discovery.Trials[second].Projection.ProjectionBytes()) {
				value.discovery.Trials[first].Projection, value.discovery.Trials[second].Projection =
					value.discovery.Trials[second].Projection, value.discovery.Trials[first].Projection
				_ = rebuildTestPhaseFacts(value)
			}
		}},
		{name: "stdin-control-subgraph-relabel", mutate: func(value *CompletedCLIStudy) {
			relabelTestControlSubgraphs(value, 0, 1)
		}},
		{name: "environment-control-subgraph-relabel", mutate: func(value *CompletedCLIStudy) {
			relabelTestControlSubgraphs(value, 2, 3)
		}},
		{name: "argv-control-subgraph-relabel", mutate: func(value *CompletedCLIStudy) {
			relabelTestControlSubgraphs(value, 4, 5)
		}},
		{name: "sparse-environment-control-subgraph-relabel", mutate: func(value *CompletedCLIStudy) {
			relabelTestControlSubgraphs(value, 6, 7)
		}},
		{name: "stderr-control-subgraph-relabel", mutate: func(value *CompletedCLIStudy) {
			relabelTestControlSubgraphs(value, 9, 10)
		}},
		{name: "eligibility-reference-substitution", mutate: func(value *CompletedCLIStudy) {
			value.recoveryOne = value.discovery
		}},
		{name: "eligibility-drift-replay", mutate: func(value *CompletedCLIStudy) {
			value.recoveryTwo = value.recoveryOne
		}},
		{name: "fabricated-controlled-projection-rejection", mutate: func(value *CompletedCLIStudy) {
			for index := range value.recoveryTwo.Trials {
				if value.recoveryTwo.Trials[index].Role != reference.EnvironmentFirst {
					value.recoveryTwo.Trials[index].ProjectionRejection = value.semanticControls[14].projectionRejection
					return
				}
			}
		}},
		{name: "fabricated-controlled-phase-projection", mutate: func(value *CompletedCLIStudy) {
			for index, trial := range value.recoveryTwo.Trials {
				if trial.Role != reference.EnvironmentFirst {
					value.phaseFacts[18+index].projection = value.phaseFacts[0].projection
					return
				}
			}
		}},
		{name: "shape-change-replay", mutate: func(value *CompletedCLIStudy) {
			value.shapeChanged = value.shapeReference
		}},
		{name: "baseline-freshness-replay", mutate: func(value *CompletedCLIStudy) {
			value.baselineCheckpoint = value.baseline
		}},
		{name: "accepted-evaluation-replay", mutate: func(value *CompletedCLIStudy) {
			value.mainEvaluation = value.baseline
		}},
		{name: "main-weak-run-reclassified-as-strong", mutate: func(value *CompletedCLIStudy) {
			value.weakGrade = value.strongGrade
		}},
		{name: "main-weak-run-replaced-by-strong", mutate: func(value *CompletedCLIStudy) {
			value.weakRun = value.strongRun
		}},
		{name: "auxiliary-strong-run-reclassified-as-weak", mutate: func(value *CompletedCLIStudy) {
			value.strongGrade = value.weakGrade
		}},
		{name: "auxiliary-strong-run-aliased-to-main", mutate: func(value *CompletedCLIStudy) {
			value.strongRun = value.weakRun
		}},
		{name: "source-spec-digest-substitution", mutate: func(value *CompletedCLIStudy) {
			value.confirmed.SourceSpecDigest = testDigest("foreign", "source-spec")
		}},
		{name: "malformed-source-spec-bytes", mutate: func(value *CompletedCLIStudy) {
			value.confirmed.SourceSpecBytes = []byte("{")
		}},
		{name: "noncanonical-source-spec-byte-encoding", mutate: func(value *CompletedCLIStudy) {
			value.confirmed.SourceSpecBytes = append(append([]byte(nil), value.confirmed.SourceSpecBytes...), '\n')
		}},
		{name: "canonical-source-spec-recompiles-to-foreign-plan", mutate: func(value *CompletedCLIStudy) {
			value.confirmed.SourceSpecDigest = foreignPlanSourceDigest
			value.confirmed.SourceSpecBytes = append([]byte(nil), foreignPlanSourceBytes...)
		}},
		{name: "cross-cohort-choicepoint-confirmation-substitution", mutate: func(value *CompletedCLIStudy) {
			value.choicepoint = foreignChoicepoint
		}},
		{name: "dropped-choice-action", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions = value.choiceAudit.actions[:len(value.choiceAudit.actions)-1]
		}},
		{name: "duplicate-choice-action", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[4] = value.choiceAudit.actions[3]
		}},
		{name: "substituted-choice-action-decision", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[2].decision = value.choiceAudit.actions[0].decision
		}},
		{name: "fabricated-ambiguity-refusal", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.ambiguityCode = ""
		}},
		{name: "fabricated-noncompilable-refusal", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[2].preparationRefusalCode = ""
		}},
		{name: "dropped-noncompilable-error-provenance", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[2].preparationErr = nil
		}},
		{name: "dropped-noncompilable-destination-provenance", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[2].destinationPath = ""
		}},
		{name: "noncompilable-destination-padding", mutate: func(value *CompletedCLIStudy) {
			value.choiceAudit.actions[2].destinationAfter = append(value.choiceAudit.actions[2].destinationAfter, "contract.js")
		}},
		{name: "dropped-contract", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials = value.officialTrials[:len(value.officialTrials)-1]
		}},
		{name: "duplicate-contract-graph", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[len(value.officialTrials)-1] = value.officialTrials[0]
		}},
		{name: "target-run-graph-mismatch", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].targetDigest = value.officialTrials[1].targetDigest
		}},
		{name: "run-classification-graph-mismatch", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].runDigest = value.officialTrials[1].runDigest
		}},
		{name: "recomposed-unrelated-official-graph", mutate: func(value *CompletedCLIStudy) {
			trial := &value.officialTrials[0]
			trial.attemptDigest = value.physicalRunAuthority
			trial.targetDigest = value.weakRun.Digest()
			trial.targetCanonicalSHA256 = strings.Repeat("a", 64)
			trial.runDigest = value.strongGrade.Grade().Digest()
			trial.runCanonicalSHA256 = strings.Repeat("b", 64)
			trial.classificationDigest = value.weakRun.Transcript().Digest()
			trial.classificationCanonicalSHA256 = strings.Repeat("c", 64)
			value.phaseFacts[88].authority = trial.classificationDigest
			value.phaseFacts[88].attempt = trial.attemptDigest
		}},
		{name: "requested-result-smuggling", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].result = "CONTRADICTS"
		}},
		{name: "unenrolled-exit-smuggling", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].exitCode = 7
		}},
		{name: "classification-recovery-divergence", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].recoveryEqual = false
		}},
		{name: "missing-typed-revalidation", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].revalidate = nil
		}},
		{name: "missing-pure-snapshot-validation", mutate: func(value *CompletedCLIStudy) {
			value.officialTrials[0].validateSnapshot = nil
		}},
	}
	type validationOutcome struct {
		index    int
		accepted bool
	}
	jobs := make(chan int)
	outcomes := make(chan validationOutcome, len(tests))
	var workers sync.WaitGroup
	const workerCount = 4
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				forged := clone()
				tests[index].mutate(&forged)
				if rebound, err := physicalRunAuthority(forged); err == nil {
					forged.physicalRunAuthority = rebound
				}
				outcomes <- validationOutcome{index: index, accepted: forged.Valid()}
			}
		}()
	}
	for index := range tests {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	close(outcomes)
	accepted := make([]bool, len(tests))
	seenOutcome := make([]bool, len(tests))
	received := 0
	for outcome := range outcomes {
		if outcome.index < 0 || outcome.index >= len(tests) || seenOutcome[outcome.index] {
			t.Fatalf("completed forgery worker returned invalid or duplicate outcome index %d", outcome.index)
		}
		seenOutcome[outcome.index] = true
		received++
		accepted[outcome.index] = outcome.accepted
	}
	if received != len(tests) {
		t.Fatalf("completed forgery workers returned %d outcomes, want %d", received, len(tests))
	}
	for index, wasAccepted := range accepted {
		if !seenOutcome[index] {
			t.Fatalf("completed forgery workers omitted %s", tests[index].name)
		}
		if wasAccepted {
			t.Fatalf("completed study accepted %s roster forgery", tests[index].name)
		}
	}
}

func assertSourceSpecAuthorityHostiles(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	alternateDigest, alternateBytes := canonicalEquivalentSourceSpec(t, completed.confirmed)
	forged := completed
	forged.confirmed.SourceSpecDigest = alternateDigest
	forged.confirmed.SourceSpecBytes = append([]byte(nil), alternateBytes...)
	if err := validateResultSourceSpec(forged.confirmed); err != nil {
		t.Fatalf("equivalent canonical source no longer validates against retained plan: %v", err)
	}
	rebound, err := physicalRunAuthority(forged)
	if err != nil || !rebound.Valid() || rebound == completed.physicalRunAuthority {
		t.Fatalf("source-only canonical change did not alter physical-run authority: original=%s rebound=%s err=%v",
			completed.physicalRunAuthority, rebound, err)
	}
	if forged.Valid() {
		t.Fatal("completed study accepted source-only canonical change under the retained physical-run authority")
	}
	if inspection, inspectErr := Inspect(forged); inspectErr == nil || inspection.Valid() || inspection.Published() {
		t.Fatalf("Inspect accepted source-only canonical change under retained authority: valid=%t published=%t err=%v",
			inspection.Valid(), inspection.Published(), inspectErr)
	}
}

func canonicalEquivalentSourceSpec(t testing.TB, result Result) (domain.Digest, []byte) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(result.SourceSpecBytes, &top); err != nil {
		t.Fatalf("decode retained source spec: %v", err)
	}
	if _, present := top["setup_argv"]; !present {
		t.Fatal("retained source spec lacks the explicit optional setup_argv member")
	}
	delete(top, "setup_argv")
	exact, err := json.Marshal(top)
	if err != nil {
		t.Fatalf("encode equivalent source spec: %v", err)
	}
	parsed, err := corespec.ParseSource(exact)
	if err != nil {
		t.Fatalf("parse equivalent source spec: %v", err)
	}
	recompiled, err := corespec.Compile(parsed, result.ProjectionDefinition.Binding())
	if err != nil || recompiled.Digest() != result.Plan.Digest() ||
		!bytes.Equal(recompiled.CanonicalBytes(), result.Plan.CanonicalBytes()) ||
		parsed.Digest() == result.SourceSpecDigest || bytes.Equal(parsed.CanonicalBytes(), result.SourceSpecBytes) {
		t.Fatalf("optional source member removal did not preserve exactly one plan under new source authority: source=%s plan=%s err=%v",
			parsed.Digest(), recompiled.Digest(), err)
	}
	return parsed.Digest(), parsed.CanonicalBytes()
}

func canonicalSourceSpecForDifferentPlan(t testing.TB, result Result) (domain.Digest, []byte) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(result.SourceSpecBytes, &top); err != nil {
		t.Fatalf("decode retained source spec: %v", err)
	}
	var budgets map[string]json.RawMessage
	if err := json.Unmarshal(top["budgets"], &budgets); err != nil {
		t.Fatalf("decode retained source budgets: %v", err)
	}
	budgets["probe_ms"] = json.RawMessage("1499")
	encodedBudgets, err := json.Marshal(budgets)
	if err != nil {
		t.Fatalf("encode foreign-plan source budgets: %v", err)
	}
	top["budgets"] = encodedBudgets
	exact, err := json.Marshal(top)
	if err != nil {
		t.Fatalf("encode foreign-plan source spec: %v", err)
	}
	parsed, err := corespec.ParseSource(exact)
	if err != nil {
		t.Fatalf("parse foreign-plan canonical source: %v", err)
	}
	recompiled, err := corespec.Compile(parsed, result.ProjectionDefinition.Binding())
	if err != nil || !recompiled.Digest().Valid() || recompiled.Digest() == result.Plan.Digest() ||
		bytes.Equal(recompiled.CanonicalBytes(), result.Plan.CanonicalBytes()) ||
		parsed.Digest() == result.SourceSpecDigest || bytes.Equal(parsed.CanonicalBytes(), result.SourceSpecBytes) {
		t.Fatalf("valid canonical source did not compile to a distinct plan: source=%s plan=%s retained=%s err=%v",
			parsed.Digest(), recompiled.Digest(), result.Plan.Digest(), err)
	}
	return parsed.Digest(), parsed.CanonicalBytes()
}

func auxiliaryChoicepointForSubstitution(t testing.TB, completed CompletedCLIStudy) choice.ChoicepointRecord {
	t.Helper()
	original, err := choice.NewCanonicalArtifact(
		"CLIStimulus",
		completed.auxiliaryBaseline.Stimulus.Digest(),
		completed.auxiliaryBaseline.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatalf("construct auxiliary original stimulus artifact: %v", err)
	}
	minimized, err := choice.NewCanonicalArtifact(
		"CLIStimulus",
		completed.auxiliaryConfirmed.Stimulus.Digest(),
		completed.auxiliaryConfirmed.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatalf("construct auxiliary minimized stimulus artifact: %v", err)
	}
	reveals := make([]choice.CandidateReveal, len(completed.auxiliaryConfirmed.CandidateBindings))
	for index, binding := range completed.auxiliaryConfirmed.CandidateBindings {
		role, present := completed.auxiliaryConfirmed.CandidateRoles[binding.Key()]
		if !present {
			t.Fatalf("auxiliary cohort lost candidate role for %s", binding.Key())
		}
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(),
			DisplayRef:            string(role),
			ProducerMetadata:      "foreign auxiliary cohort substitution hostile",
		}
	}
	foreign, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario:          "Should this separate auxiliary cohort become the CLI contract?",
		Plan:              completed.auxiliaryConfirmed.Plan,
		Envelope:          completed.auxiliaryConfirmed.Envelope,
		CandidateBindings: completed.auxiliaryConfirmed.CandidateBindings,
		OriginalStimulus:  original,
		MinimizedStimulus: minimized,
		Confirmation:      completed.auxiliaryConfirmed.Confirmation.Draft().Record(),
		CandidateReveals:  reveals,
		EvidenceReceipts:  []domain.ReceiptReference{},
	})
	if err != nil || !foreign.Valid() {
		t.Fatalf("construct foreign auxiliary Choicepoint: valid=%t err=%v", foreign.Valid(), err)
	}
	return foreign
}

func assertOfficialTrialClosure(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	if len(completed.officialTrials) != 10 {
		t.Fatalf("official trial count = %d, want 10", len(completed.officialTrials))
	}
	seenAttempts := make(map[string]struct{}, 10)
	seenTargets := make(map[string]struct{}, 10)
	seenRuns := make(map[string]struct{}, 10)
	seenClassifications := make(map[string]struct{}, 10)
	seenTargetBytes := make(map[string]struct{}, 10)
	seenRunBytes := make(map[string]struct{}, 10)
	seenClassificationBytes := make(map[string]struct{}, 10)
	for index, trial := range completed.officialTrials {
		if !trial.attemptDigest.Valid() || !trial.targetDigest.Valid() || !trial.runDigest.Valid() ||
			!trial.classificationDigest.Valid() || trial.bundleDigest != completed.bundle.Digest() ||
			trial.residueHeadDigest != completed.residue.HeadDigest() || trial.exitCode != 0 ||
			trial.result != "CONFORMS" || !trial.recoveryEqual || trial.validateSnapshot == nil || trial.revalidate == nil {
			t.Fatalf("official trial %d has incomplete joined facts: %+v", index+1, trial)
		}
		if trial.attemptDigest == trial.targetDigest || trial.targetDigest == trial.runDigest ||
			trial.runDigest == trial.classificationDigest {
			t.Fatalf("official trial %d collapsed domain-separated graph identities", index+1)
		}
		for label, digest := range map[string]string{
			"attempt": trial.attemptDigest.String(), "target": trial.targetDigest.String(),
			"run": trial.runDigest.String(), "classification": trial.classificationDigest.String(),
		} {
			if !strings.HasPrefix(digest, "sha256:") || !isLowerSHA256(strings.TrimPrefix(digest, "sha256:")) {
				t.Fatalf("official trial %d %s digest = %q", index+1, label, digest)
			}
		}
		for label, digest := range map[string]string{
			"target bytes":         trial.targetCanonicalSHA256,
			"run bytes":            trial.runCanonicalSHA256,
			"classification bytes": trial.classificationCanonicalSHA256,
		} {
			if !isLowerSHA256(digest) {
				t.Fatalf("official trial %d %s SHA-256 = %q", index+1, label, digest)
			}
		}
		if validationErr := trial.validate(completed.bundle.Digest(), completed.residue.HeadDigest()); validationErr != nil {
			t.Fatalf("official trial %d pure snapshot validation: %v", index+1, validationErr)
		}
		assertUniqueOfficialFact(t, seenAttempts, trial.attemptDigest.String(), index, "attempt")
		assertUniqueOfficialFact(t, seenTargets, trial.targetDigest.String(), index, "target")
		assertUniqueOfficialFact(t, seenRuns, trial.runDigest.String(), index, "run")
		assertUniqueOfficialFact(t, seenClassifications, trial.classificationDigest.String(), index, "classification")
		assertUniqueOfficialFact(t, seenTargetBytes, trial.targetCanonicalSHA256, index, "target canonical bytes")
		assertUniqueOfficialFact(t, seenRunBytes, trial.runCanonicalSHA256, index, "run canonical bytes")
		assertUniqueOfficialFact(t, seenClassificationBytes, trial.classificationCanonicalSHA256, index, "classification canonical bytes")
	}
	for label, count := range map[string]int{
		"attempts": len(seenAttempts), "targets": len(seenTargets), "runs": len(seenRuns),
		"classifications": len(seenClassifications), "target byte bodies": len(seenTargetBytes),
		"run byte bodies": len(seenRunBytes), "classification byte bodies": len(seenClassificationBytes),
	} {
		if count != 10 {
			t.Fatalf("distinct official %s = %d, want 10", label, count)
		}
	}
}

func assertUniqueOfficialFact(t testing.TB, seen map[string]struct{}, value string, index int, label string) {
	t.Helper()
	if _, duplicate := seen[value]; duplicate {
		t.Fatalf("official trial %d reused %s %q", index+1, label, value)
	}
	seen[value] = struct{}{}
}

func prepareCLICompletionFixture(t *testing.T, node string) string {
	t.Helper()
	repositoryRoot := testRepositoryRoot(t)
	driver := filepath.Join(repositoryRoot, "tools", "run-u7-cli-study.mjs")
	if metadata, err := os.Lstat(driver); err != nil || !metadata.Mode().IsRegular() || metadata.Mode().Perm() != 0o644 {
		t.Fatalf("CLI fixture driver is not exact: metadata=%v err=%v", metadata, err)
	}
	parent := privateDirectoryForTB(t, "completion-fixture-parent")
	fixtureRoot := filepath.Join(parent, "fixture")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, node, driver,
		"--prepare", "--domain", "cli", "--ordinal", "1", "--fixture-root", fixtureRoot,
	)
	command.Dir = repositoryRoot
	command.Env = []string{
		"HOME=" + parent,
		"TMPDIR=" + parent,
		"PATH=/usr/bin:/bin:/opt/homebrew/bin",
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("prepare direct CLI fixture: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	resolved, err := filepath.EvalSymlinks(fixtureRoot)
	if err != nil || resolved != fixtureRoot {
		t.Fatalf("prepared fixture root is not canonical: resolved=%q err=%v", resolved, err)
	}
	return fixtureRoot
}

func privateDirectoryForTB(t testing.TB, name string) string {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("private test directory %q is not canonical: %q %v", path, resolved, err)
	}
	return resolved
}

func requireExactExecutable(t testing.TB, path string) string {
	t.Helper()
	metadata, err := os.Stat(path)
	if err != nil {
		t.Fatalf("required executable %s is unavailable: %v", path, err)
	}
	if !metadata.Mode().IsRegular() || metadata.Mode().Perm()&0o111 == 0 {
		t.Fatalf("required executable %s has mode %v", path, metadata.Mode())
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("required executable %s did not resolve canonically: %q %v", path, resolved, err)
	}
	return resolved
}

func isLowerSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, current := range value {
		if (current < '0' || current > '9') && (current < 'a' || current > 'f') {
			return false
		}
	}
	return true
}
