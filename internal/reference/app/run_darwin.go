package app

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	maximumStudyOrdinal                      = 3
	studyNodeRunnerCloseBudget               = 20 * time.Second
	unexpectedStudyProcessGroupCleanupBudget = 4 * time.Second
)

// StudyRequest is the admitted, domain-neutral authority passed to an
// installed study handler. It intentionally carries no stdout or stderr
// writer: only app may emit the frozen process result.
type StudyRequest struct {
	Context          context.Context
	Domain           string
	Ordinal          int
	EvidenceRoot     string
	WorkingDirectory string
	ScratchRoot      string
	NodeExecutable   string
	GitExecutable    string
	Workspace        *EvidenceWorkspace
	nodeTestRunner   *studyNodeTestRunner
}

// StudyNodeTestInput is the closed, domain-neutral direct-Node test surface
// available to an admitted study. The app fixes the executable, argv shape,
// environment, capture limits, process group, and teardown policy.
type StudyNodeTestInput struct {
	WorkingDirectory   string
	TestFile           string
	HomeDirectory      string
	TemporaryDirectory string
	Timeout            time.Duration
}

// StudyNodeTestObservation contains bounded parent observations only. It has
// no process, retry, semantic-classification, or filesystem authority.
type StudyNodeTestObservation struct {
	Stdout           []byte
	Stderr           []byte
	InvocationSHA256 string
	TestFileSHA256   string
	ExitCode         int
}

type studyNodeTestRunner struct {
	node       admittedExecutable
	scratch    admittedDirectory
	context    context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	closed     bool
	active     bool
	activeDone chan struct{}
	poison     error
}

type admittedRegularFile struct {
	path string
	info os.FileInfo
	sum  [sha256.Size]byte
}

type studyPipeCapture struct {
	bytes    []byte
	observed int64
	overflow bool
	err      error
}

type studyWaitResult struct {
	state   *os.ProcessState
	err     error
	endedAt time.Time
}

type admittedStudyBoundary struct {
	evidenceRoot     admittedDirectory
	workingDirectory admittedDirectory
	scratchRoot      admittedDirectory
	node             admittedExecutable
	git              admittedExecutable
}

type admittedDirectory struct {
	path string
	info os.FileInfo
}

type admittedExecutable struct {
	requested string
	resolved  string
	info      os.FileInfo
}

func runMachineStudy(domain string, runtime Runtime) int {
	fail := func(err error) int {
		response := failureEnvelope("study", err)
		var input *InputError
		if errors.As(err, &input) {
			response.ExitCode = ExitRefused
		}
		response.NextAction = "Leave this attempt unclaimed, repair the frozen study boundary, and rerun it only through the repository harness."
		if renderErr := renderJSON(runtime.Stdout, response); renderErr != nil {
			return ExitInternal
		}
		return response.ExitCode
	}
	if runtime.studyConfigurationError != nil {
		return fail(runtime.studyConfigurationError)
	}
	handler, present := runtime.Studies[domain]
	if !present || handler == nil {
		return fail(&InputError{Code: "STUDY_HANDLER_UNAVAILABLE", Detail: "no installed handler owns the requested study domain"})
	}
	request, boundary, err := admitStudyRequest(domain, runtime)
	if err != nil {
		return fail(err)
	}
	handlerErr := invokeStudyHandler(handler, request)
	runnerErr := request.closeNodeTestRunner()
	closeErr := request.Workspace.close()
	if err := errors.Join(handlerErr, runnerErr, closeErr); err != nil {
		return fail(err)
	}
	if err := verifyStudyBoundary(boundary); err != nil {
		return fail(err)
	}
	if runtime.Context.Err() != nil {
		return fail(&InputError{Code: "STUDY_CONTEXT_CANCELED", Detail: "the admitted study context ended before result publication"})
	}
	if err := renderStudyDomainResult(runtime.Stdout, domain, request.Ordinal); err != nil {
		return ExitInternal
	}
	return ExitOK
}

func invokeStudyHandler(handler StudyHandler, request StudyRequest) (returnErr error) {
	defer func() {
		if recover() != nil {
			returnErr = errors.New("study handler panicked and the attempt is not publishable")
		}
	}()
	if request.Context.Err() != nil {
		return &InputError{Code: "STUDY_CONTEXT_CANCELED", Detail: "the admitted study context ended before execution"}
	}
	return handler(request)
}

func admitStudyRequest(domain string, runtime Runtime) (StudyRequest, admittedStudyBoundary, error) {
	if !validStudyDomain(domain) {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_DOMAIN_INVALID", Detail: "the requested study domain is invalid"}
	}
	lookup := runtime.LookupEnvironment
	environmentDomain, present := lookup("COUNTERSHAPE_STUDY_DOMAIN")
	if !present || environmentDomain != domain {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_ENVIRONMENT_INVALID", Detail: "COUNTERSHAPE_STUDY_DOMAIN is absent or differs from the requested domain"}
	}
	rawOrdinal, present := lookup("COUNTERSHAPE_STUDY_ORDINAL")
	ordinal, err := strconv.Atoi(rawOrdinal)
	if !present || err != nil || ordinal < 1 || ordinal > maximumStudyOrdinal || rawOrdinal != strconv.Itoa(ordinal) {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_ENVIRONMENT_INVALID", Detail: "COUNTERSHAPE_STUDY_ORDINAL must be the canonical integer 1, 2, or 3"}
	}

	workingDirectory := runtime.WorkingDirectory
	if workingDirectory == "" {
		workingDirectory, err = os.Getwd()
		if err != nil {
			return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_WORKSPACE_INVALID", Detail: "the study working directory is unavailable"}
		}
	}
	working, err := admitExactDirectory(workingDirectory, false, false)
	if err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_WORKSPACE_INVALID", Detail: "the study working directory must be one existing exact absolute directory"}
	}
	rawScratch, scratchPresent := lookup("TMPDIR")
	scratch, err := admitExactDirectory(rawScratch, true, false)
	if !scratchPresent || err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_SCRATCH_INVALID", Detail: "TMPDIR must name one existing exact private directory"}
	}
	rawEvidence, evidencePresent := lookup("COUNTERSHAPE_EVIDENCE_ROOT")
	evidence, err := admitExactDirectory(rawEvidence, true, true)
	if !evidencePresent || err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_EVIDENCE_ROOT_INVALID", Detail: "COUNTERSHAPE_EVIDENCE_ROOT must name one existing exact-empty private directory"}
	}
	if os.SameFile(working.info, scratch.info) || os.SameFile(working.info, evidence.info) || os.SameFile(scratch.info, evidence.info) {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_ROOT_ALIAS", Detail: "working, scratch, and evidence roots must be distinct directories"}
	}

	rawNode, nodePresent := lookup("COUNTERSHAPE_NODE")
	node, err := admitExecutable(rawNode)
	if !nodePresent || err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_RUNTIME_INVALID", Detail: "COUNTERSHAPE_NODE must name one absolute executable resolving to a regular file"}
	}
	rawGit, gitPresent := lookup("COUNTERSHAPE_GIT")
	git, err := admitExecutable(rawGit)
	if !gitPresent || err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_RUNTIME_INVALID", Detail: "COUNTERSHAPE_GIT must name one absolute executable resolving to a regular file"}
	}
	if os.SameFile(node.info, git.info) {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_RUNTIME_INVALID", Detail: "Node and Git runtime tools must not alias"}
	}

	workspace, err := newEvidenceWorkspace(evidence.path, evidence.info)
	if err != nil {
		return StudyRequest{}, admittedStudyBoundary{}, &InputError{Code: "STUDY_EVIDENCE_ROOT_INVALID", Detail: "the admitted evidence root changed while it was opened"}
	}
	runnerContext := runtime.Context
	if runnerContext == nil {
		runnerContext = context.Background()
	}
	retainedContext, cancelRunner := context.WithCancel(runnerContext)
	request := StudyRequest{
		Context: runtime.Context, Domain: domain, Ordinal: ordinal,
		EvidenceRoot: evidence.path, WorkingDirectory: working.path, ScratchRoot: scratch.path,
		NodeExecutable: node.resolved, GitExecutable: git.resolved, Workspace: workspace,
		nodeTestRunner: &studyNodeTestRunner{
			node: node, scratch: scratch, context: retainedContext, cancel: cancelRunner,
		},
	}
	boundary := admittedStudyBoundary{
		evidenceRoot: evidence, workingDirectory: working, scratchRoot: scratch, node: node, git: git,
	}
	return request, boundary, nil
}

// RunNodeTest runs only Node's exact test entrypoint under the authority that
// admitStudyRequest retained. Success requires the direct child, captured
// streams, helper goroutines, and owned outer process group to be terminal. A
// failed bounded cleanup retains the runner lease until its helpers actually
// return, so application closure cannot promote the failed return to terminal
// process authority. Generated tests may apply their own stricter descendant
// lifecycle and semantic checks.
func (request StudyRequest) RunNodeTest(ctx context.Context, input StudyNodeTestInput) (observation StudyNodeTestObservation, returnErr error) {
	runner := request.nodeTestRunner
	if ctx == nil || runner == nil {
		return StudyNodeTestObservation{}, errors.New("study Node test authority is absent or invalid")
	}
	runnerContext, release, err := runner.acquire()
	if err != nil {
		return StudyNodeTestObservation{}, err
	}
	var helpersDone <-chan struct{}
	defer func() {
		if helpersDone == nil {
			release(returnErr)
			return
		}
		select {
		case <-helpersDone:
			release(returnErr)
		default:
			// A failed bounded cleanup may return before an uninterruptible
			// child wait or pipe read does. Keep the runner lease active until
			// every helper has actually returned, so close cannot mistake the
			// RunNodeTest return boundary for process/goroutine terminality.
			go func(runErr error) {
				<-helpersDone
				release(runErr)
			}(returnErr)
		}
	}()
	if input.Timeout <= 0 || input.Timeout > 2*time.Minute {
		return StudyNodeTestObservation{}, errors.New("study Node test input is invalid")
	}
	if runnerContext.Err() != nil || ctx.Err() != nil {
		return StudyNodeTestObservation{}, errors.New("study Node test lifetime ended before execution")
	}
	if err := verifyExecutable(runner.node); err != nil {
		return StudyNodeTestObservation{}, errors.New("admitted Node executable changed before the direct test")
	}
	if err := verifyExactDirectory(runner.scratch, true); err != nil {
		return StudyNodeTestObservation{}, errors.New("admitted study scratch changed before the direct test")
	}

	working, err := admitNestedPrivateDirectory(runner.scratch.path, input.WorkingDirectory, false)
	if err != nil {
		return StudyNodeTestObservation{}, fmt.Errorf("direct test working directory: %w", err)
	}
	home, err := admitNestedPrivateDirectory(runner.scratch.path, input.HomeDirectory, true)
	if err != nil {
		return StudyNodeTestObservation{}, fmt.Errorf("direct test home directory: %w", err)
	}
	temporary, err := admitNestedPrivateDirectory(runner.scratch.path, input.TemporaryDirectory, true)
	if err != nil {
		return StudyNodeTestObservation{}, fmt.Errorf("direct test temporary directory: %w", err)
	}
	testFile, err := admitNestedRegularFile(runner.scratch.path, input.TestFile)
	if err != nil || filepath.Base(testFile.path) != "contract.test.mjs" {
		return StudyNodeTestObservation{}, errors.New("direct test file is not one admitted contract.test.mjs")
	}
	bundleRoot, err := admitNestedPrivateDirectory(runner.scratch.path, filepath.Dir(testFile.path), false)
	if err != nil {
		return StudyNodeTestObservation{}, fmt.Errorf("direct test bundle root: %w", err)
	}
	for _, pair := range [][2]admittedDirectory{
		{working, home}, {working, temporary}, {working, bundleRoot},
		{home, temporary}, {home, bundleRoot}, {temporary, bundleRoot},
	} {
		if os.SameFile(pair[0].info, pair[1].info) ||
			strictlyWithin(pair[0].path, pair[1].path) || strictlyWithin(pair[1].path, pair[0].path) {
			return StudyNodeTestObservation{}, errors.New("direct test roots alias or overlap")
		}
	}

	argv := []string{"node", "--test", "--test-reporter=tap", testFile.path}
	environment := []string{
		"HOME=" + home.path,
		"LANG=C",
		"LC_ALL=C",
		"NODE_OPTIONS=--no-warnings",
		"NO_COLOR=1",
		"PATH=/usr/bin:/bin",
		"TMPDIR=" + temporary.path,
		"TZ=UTC",
	}
	runContext, cancelRun := context.WithCancel(ctx)
	stopRequestCancellation := context.AfterFunc(runnerContext, cancelRun)
	defer func() {
		stopRequestCancellation()
		cancelRun()
	}()
	testSHA256 := hex.EncodeToString(testFile.sum[:])
	invocationSHA256 := hashStudyNodeInvocation(runner.node.resolved, argv, environment, working.path, testSHA256, input.Timeout)
	observation, helpersDone, runErr := startStudyNodeTest(
		runContext, runner.node.resolved, argv, environment, working.path, input.Timeout,
	)
	terminalErr := verifyStudyNodeTestTerminal(runner, working, home, temporary, bundleRoot, testFile)
	if runContext.Err() != nil {
		terminalErr = errors.Join(terminalErr, errors.New("study Node test lifetime ended during execution"))
	}
	if runErr != nil || terminalErr != nil {
		return StudyNodeTestObservation{}, errors.Join(runErr, terminalErr)
	}
	observation.InvocationSHA256 = invocationSHA256
	observation.TestFileSHA256 = testSHA256
	return observation, nil
}

func (runner *studyNodeTestRunner) acquire() (context.Context, func(error), error) {
	if runner == nil {
		return nil, nil, errors.New("study Node test runner is absent")
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if runner.closed || runner.context == nil || runner.cancel == nil {
		return nil, nil, errors.New("study Node test runner is closed")
	}
	if runner.poison != nil {
		return nil, nil, errors.New("study Node test runner is poisoned")
	}
	if runner.active {
		err := errors.New("concurrent study Node tests are forbidden")
		runner.poison = errors.Join(runner.poison, err)
		return nil, nil, err
	}
	runner.active = true
	runner.activeDone = make(chan struct{})
	released := false
	release := func(runErr error) {
		runner.mu.Lock()
		defer runner.mu.Unlock()
		if released {
			return
		}
		released = true
		if runErr != nil {
			runner.poison = errors.Join(runner.poison, runErr)
		}
		if runner.active {
			runner.active = false
			close(runner.activeDone)
			runner.activeDone = nil
		}
	}
	return runner.context, release, nil
}

func (request StudyRequest) closeNodeTestRunner() error {
	runner := request.nodeTestRunner
	if runner == nil {
		return errors.New("study Node test runner is absent")
	}
	var cancel context.CancelFunc
	runner.mu.Lock()
	if !runner.closed {
		runner.closed = true
		if runner.context == nil || runner.cancel == nil {
			runner.poison = errors.Join(runner.poison, errors.New("study Node test runner lifetime is incomplete"))
		} else {
			cancel = runner.cancel
		}
	}
	done := runner.activeDone
	runner.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		timer := time.NewTimer(studyNodeRunnerCloseBudget)
		select {
		case <-done:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			runner.mu.Lock()
			runner.poison = errors.Join(runner.poison, errors.New("study Node test runner could not prove terminal shutdown"))
			runner.mu.Unlock()
		}
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	return runner.poison
}

func verifyStudyNodeTestTerminal(
	runner *studyNodeTestRunner,
	working, home, temporary, bundleRoot admittedDirectory,
	testFile admittedRegularFile,
) error {
	var terminalErr error
	if err := verifyExecutable(runner.node); err != nil {
		terminalErr = errors.Join(terminalErr, errors.New("admitted Node executable changed during the direct test"))
	}
	if err := verifyExactDirectory(runner.scratch, true); err != nil {
		terminalErr = errors.Join(terminalErr, errors.New("admitted study scratch changed during the direct test"))
	}
	for _, directory := range []admittedDirectory{working, home, temporary, bundleRoot} {
		if err := verifyExactDirectory(directory, true); err != nil {
			terminalErr = errors.Join(terminalErr, errors.New("direct test directory identity changed"))
		}
	}
	for _, directory := range []admittedDirectory{home, temporary} {
		entries, readErr := os.ReadDir(directory.path)
		if readErr != nil || len(entries) != 0 {
			terminalErr = errors.Join(terminalErr, errors.New("direct test left private runtime residue"))
		}
	}
	if err := verifyAdmittedRegularFile(testFile); err != nil {
		terminalErr = errors.Join(terminalErr, errors.New("direct test file changed during execution"))
	}
	return terminalErr
}

func admitNestedPrivateDirectory(root, path string, empty bool) (admittedDirectory, error) {
	if !strictlyWithin(root, path) {
		return admittedDirectory{}, errors.New("directory is outside the admitted scratch root")
	}
	return admitExactDirectory(path, true, empty)
}

func admitNestedRegularFile(root, path string) (admittedRegularFile, error) {
	if !strictlyWithin(root, path) || !validExactAbsolutePath(path) {
		return admittedRegularFile{}, errors.New("file is outside the admitted scratch root")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return admittedRegularFile{}, errors.New("file path is not canonical")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		hasSpecialMode(info.Mode()) || info.Mode().Perm() != 0o644 || info.Size() < 1 || info.Size() > 640<<10 || !singleLink(info) {
		return admittedRegularFile{}, errors.New("file is not one bounded exact regular file")
	}
	body, err := os.ReadFile(path)
	if err != nil || int64(len(body)) != info.Size() {
		return admittedRegularFile{}, errors.New("file could not be read exactly")
	}
	return admittedRegularFile{path: path, info: info, sum: sha256.Sum256(body)}, nil
}

func verifyAdmittedRegularFile(expected admittedRegularFile) error {
	resolved, err := filepath.EvalSymlinks(expected.path)
	if err != nil || resolved != expected.path {
		return errors.New("file resolution changed")
	}
	info, err := os.Lstat(expected.path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || hasSpecialMode(info.Mode()) ||
		info.Mode().Perm() != 0o644 || !singleLink(info) || !os.SameFile(expected.info, info) ||
		info.Mode() != expected.info.Mode() || info.Size() != expected.info.Size() || info.ModTime() != expected.info.ModTime() {
		return errors.New("file identity changed")
	}
	body, err := os.ReadFile(expected.path)
	if err != nil || sha256.Sum256(body) != expected.sum {
		return errors.New("file bytes changed")
	}
	return nil
}

func strictlyWithin(root, path string) bool {
	if !validExactAbsolutePath(root) || !validExactAbsolutePath(path) || path == root {
		return false
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func singleLink(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink == 1
}

func startStudyNodeTest(
	ctx context.Context,
	executable string,
	argv []string,
	environment []string,
	cwd string,
	executionBudget time.Duration,
) (StudyNodeTestObservation, <-chan struct{}, error) {
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return StudyNodeTestObservation{}, nil, errors.New("direct test stdin could not be opened")
	}
	defer stdin.Close()
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return StudyNodeTestObservation{}, nil, errors.New("direct test stdout pipe could not be opened")
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		stdoutReader.Close()
		stdoutWriter.Close()
		return StudyNodeTestObservation{}, nil, errors.New("direct test stderr pipe could not be opened")
	}
	closePipes := func() {
		stdoutReader.Close()
		stdoutWriter.Close()
		stderrReader.Close()
		stderrWriter.Close()
	}

	deadline := time.Now().Add(executionBudget)
	process, err := os.StartProcess(executable, argv, &os.ProcAttr{
		Dir: cwd, Env: append([]string(nil), environment...),
		Files: []*os.File{stdin, stdoutWriter, stderrWriter},
		Sys:   &syscall.SysProcAttr{Setpgid: true},
	})
	if err != nil {
		closePipes()
		return StudyNodeTestObservation{}, nil, errors.New("direct Node test could not start")
	}
	pgid, pgidErr := syscall.Getpgid(process.Pid)
	if pgidErr != nil || pgid != process.Pid {
		stdoutWriter.Close()
		stderrWriter.Close()
		stdoutReader.Close()
		stderrReader.Close()
		cleanupErr, cleanupDone := stopUnexpectedStudyProcess(process, unexpectedStudyProcessGroupCleanupBudget)
		return StudyNodeTestObservation{}, cleanupDone, errors.Join(
			errors.New("direct Node test process group was not owned exactly"), cleanupErr,
		)
	}
	stdoutWriter.Close()
	stderrWriter.Close()
	stdoutC := make(chan studyPipeCapture, 1)
	stderrC := make(chan studyPipeCapture, 1)
	overflowC := make(chan struct{}, 1)
	waitC := make(chan studyWaitResult, 1)
	var helpers sync.WaitGroup
	helpers.Add(3)
	go func() {
		defer helpers.Done()
		drainStudyPipe(stdoutReader, 512<<10, stdoutC, overflowC)
	}()
	go func() {
		defer helpers.Done()
		drainStudyPipe(stderrReader, 64<<10, stderrC, overflowC)
	}()
	go func() {
		defer helpers.Done()
		state, waitErr := process.Wait()
		waitC <- studyWaitResult{state: state, err: waitErr, endedAt: time.Now()}
	}()
	helpersDone := make(chan struct{})
	go func() {
		helpers.Wait()
		close(helpersDone)
	}()

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	var waited studyWaitResult
	forced := false
	select {
	case waited = <-waitC:
		if ctx.Err() != nil || !waited.endedAt.Before(deadline) {
			forced = true
		}
		select {
		case <-overflowC:
			forced = true
		default:
		}
	case <-ctx.Done():
		forced = true
		waited = stopStudyProcessGroup(process.Pid, waitC)
	case <-timer.C:
		forced = true
		waited = stopStudyProcessGroup(process.Pid, waitC)
	case <-overflowC:
		forced = true
		waited = stopStudyProcessGroup(process.Pid, waitC)
	}
	groupExited := waitForStudyProcessGroupExit(process.Pid, 2*time.Second)
	if !groupExited {
		forced = true
		_ = syscall.Kill(-process.Pid, syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)
		_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
		groupExited = waitForStudyProcessGroupExit(process.Pid, 2*time.Second)
	}
	stdout := waitStudyPipe(stdoutReader, stdoutC)
	stderr := waitStudyPipe(stderrReader, stderrC)
	if forced || waited.state == nil || waited.err != nil || stdout.err != nil || stderr.err != nil ||
		stdout.overflow || stderr.overflow || !groupExited {
		return StudyNodeTestObservation{}, helpersDone, errors.New("direct Node test did not reach one clean bounded terminal state")
	}
	// Receiving all three results proves their work is complete, but each
	// helper closes its lifecycle only after sending. Join that final edge
	// before exposing a successful observation or releasing the runner lease.
	<-helpersDone
	exitCode := waited.state.ExitCode()
	if exitCode < 0 {
		return StudyNodeTestObservation{}, helpersDone, errors.New("direct Node test ended by signal")
	}
	return StudyNodeTestObservation{
		Stdout: append([]byte(nil), stdout.bytes...), Stderr: append([]byte(nil), stderr.bytes...), ExitCode: exitCode,
	}, helpersDone, nil
}

func stopUnexpectedStudyProcess(process *os.Process, budget time.Duration) (error, <-chan struct{}) {
	if process == nil || process.Pid < 1 || budget <= 0 {
		return errors.New("unexpected direct test child could not be cleaned"), nil
	}
	waitC := make(chan studyWaitResult, 1)
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		state, waitErr := process.Wait()
		waitC <- studyWaitResult{state: state, err: waitErr, endedAt: time.Now()}
	}()
	_ = process.Kill()
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case waited := <-waitC:
		if waited.state == nil || waited.err != nil {
			return errors.New("unexpected direct test child wait was inconclusive"), waitDone
		}
		<-waitDone
		return nil, waitDone
	case <-timer.C:
		_ = process.Kill()
		return errors.New("unexpected direct test child cleanup exceeded its budget"), waitDone
	}
}

func drainStudyPipe(file *os.File, limit int64, result chan<- studyPipeCapture, overflow chan<- struct{}) {
	defer file.Close()
	captured := studyPipeCapture{}
	buffer := make([]byte, 32<<10)
	for {
		count, err := file.Read(buffer)
		if count > 0 {
			captured.observed += int64(count)
			remaining := limit - int64(len(captured.bytes))
			if remaining > int64(count) {
				remaining = int64(count)
			}
			if remaining > 0 {
				captured.bytes = append(captured.bytes, buffer[:remaining]...)
			}
			if int64(count) > remaining {
				if !captured.overflow {
					captured.overflow = true
					select {
					case overflow <- struct{}{}:
					default:
					}
				}
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				captured.err = err
			}
			result <- captured
			return
		}
	}
}

func waitStudyPipe(file *os.File, result <-chan studyPipeCapture) studyPipeCapture {
	select {
	case captured := <-result:
		return captured
	case <-time.After(2 * time.Second):
		_ = file.Close()
		select {
		case captured := <-result:
			captured.err = errors.Join(captured.err, errors.New("direct test stream drain exceeded its budget"))
			return captured
		case <-time.After(time.Second):
			return studyPipeCapture{err: errors.New("direct test stream did not close")}
		}
	}
}

func stopStudyProcessGroup(pid int, waitC <-chan studyWaitResult) studyWaitResult {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case waited := <-waitC:
		return waited
	case <-time.After(2 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		select {
		case waited := <-waitC:
			return waited
		case <-time.After(2 * time.Second):
			return studyWaitResult{err: errors.New("direct test child could not be waited")}
		}
	}
}

func waitForStudyProcessGroupExit(pid int, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for {
		err := syscall.Kill(-pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return true
		}
		if budget <= 0 || !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func hashStudyNodeInvocation(executable string, argv, environment []string, cwd, testSHA256 string, timeout time.Duration) string {
	digest := sha256.New()
	writeStudyHashFrame := func(name, value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(name)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(name))
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(value))
	}
	writeStudyHashFrame("executable", executable)
	for _, value := range argv {
		writeStudyHashFrame("argv", value)
	}
	for _, value := range environment {
		writeStudyHashFrame("environment", value)
	}
	writeStudyHashFrame("cwd", cwd)
	writeStudyHashFrame("test-sha256", testSHA256)
	writeStudyHashFrame("timeout-ns", strconv.FormatInt(timeout.Nanoseconds(), 10))
	return hex.EncodeToString(digest.Sum(nil))
}

func admitExactDirectory(path string, private, empty bool) (admittedDirectory, error) {
	if !validExactAbsolutePath(path) {
		return admittedDirectory{}, errors.New("directory path is not exact and absolute")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return admittedDirectory{}, errors.New("directory path is not canonical")
	}
	metadata, err := os.Lstat(path)
	if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 || hasSpecialMode(metadata.Mode()) {
		return admittedDirectory{}, errors.New("path is not a directory")
	}
	if private && metadata.Mode().Perm() != 0o700 {
		return admittedDirectory{}, errors.New("directory is not private")
	}
	if empty {
		entries, readErr := os.ReadDir(path)
		if readErr != nil || len(entries) != 0 {
			return admittedDirectory{}, errors.New("directory is not empty")
		}
	}
	return admittedDirectory{path: path, info: metadata}, nil
}

func admitExecutable(path string) (admittedExecutable, error) {
	if !validExactAbsolutePath(path) {
		return admittedExecutable{}, errors.New("executable path is not exact and absolute")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !validExactAbsolutePath(resolved) {
		return admittedExecutable{}, errors.New("executable path cannot be resolved")
	}
	metadata, err := os.Lstat(resolved)
	if err != nil || metadata.Mode()&os.ModeSymlink != 0 || !metadata.Mode().IsRegular() ||
		metadata.Mode().Perm()&0o111 == 0 || hasSpecialMode(metadata.Mode()) {
		return admittedExecutable{}, errors.New("resolved tool is not a regular executable")
	}
	return admittedExecutable{requested: path, resolved: resolved, info: metadata}, nil
}

func validExactAbsolutePath(path string) bool {
	return path != "" && validInputPath(path) && path != "-" && filepath.IsAbs(path) && filepath.Clean(path) == path
}

func verifyStudyBoundary(boundary admittedStudyBoundary) error {
	if err := verifyExactDirectory(boundary.evidenceRoot, true); err != nil {
		return &InputError{Code: "STUDY_EVIDENCE_ROOT_CHANGED", Detail: "the evidence root identity changed during the study"}
	}
	entries, err := os.ReadDir(boundary.evidenceRoot.path)
	if err != nil || len(entries) == 0 {
		return &InputError{Code: "STUDY_EVIDENCE_INCOMPLETE", Detail: "the handler returned without publishing evidence artifacts"}
	}
	if err := verifyExactDirectory(boundary.workingDirectory, false); err != nil {
		return &InputError{Code: "STUDY_WORKSPACE_CHANGED", Detail: "the study working-directory identity changed"}
	}
	if err := verifyExactDirectory(boundary.scratchRoot, true); err != nil {
		return &InputError{Code: "STUDY_SCRATCH_CHANGED", Detail: "the study scratch-root identity changed"}
	}
	if err := verifyExecutable(boundary.node); err != nil {
		return &InputError{Code: "STUDY_RUNTIME_CHANGED", Detail: "the admitted Node executable identity changed"}
	}
	if err := verifyExecutable(boundary.git); err != nil {
		return &InputError{Code: "STUDY_RUNTIME_CHANGED", Detail: "the admitted Git executable identity changed"}
	}
	return nil
}

func verifyExactDirectory(expected admittedDirectory, private bool) error {
	resolved, err := filepath.EvalSymlinks(expected.path)
	if err != nil || resolved != expected.path {
		return errors.New("directory resolution changed")
	}
	metadata, err := os.Lstat(expected.path)
	if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 || hasSpecialMode(metadata.Mode()) ||
		!os.SameFile(expected.info, metadata) || metadata.Mode() != expected.info.Mode() {
		return errors.New("directory identity changed")
	}
	if private && metadata.Mode().Perm() != 0o700 {
		return errors.New("directory privacy changed")
	}
	return nil
}

func verifyExecutable(expected admittedExecutable) error {
	resolved, err := filepath.EvalSymlinks(expected.requested)
	if err != nil || resolved != expected.resolved {
		return errors.New("executable resolution changed")
	}
	metadata, err := os.Lstat(expected.resolved)
	if err != nil || !metadata.Mode().IsRegular() || metadata.Mode()&os.ModeSymlink != 0 || metadata.Mode().Perm()&0o111 == 0 ||
		hasSpecialMode(metadata.Mode()) || metadata.Mode() != expected.info.Mode() || !os.SameFile(expected.info, metadata) ||
		metadata.Size() != expected.info.Size() || metadata.ModTime() != expected.info.ModTime() {
		return fmt.Errorf("executable identity changed: %s", expected.requested)
	}
	return nil
}

func hasSpecialMode(mode os.FileMode) bool {
	return mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0
}
