//go:build darwin

package runner

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	fixtureRootEnvironment        = "COUNTERSHAPE_FIXTURE_ROOT"
	stateRootEnvironment          = "COUNTERSHAPE_STATE_ROOT"
	evidenceRootEnvironment       = "COUNTERSHAPE_EVIDENCE_ROOT"
	attemptIDEnvironment          = "COUNTERSHAPE_ATTEMPT_ID"
	scheduleEnvironment           = "COUNTERSHAPE_SCHEDULE_ORDINAL"
	repetitionEnvironment         = "COUNTERSHAPE_SCHEDULE_REPETITION"
	importCanaryModuleEnvironment = "COUNTERSHAPE_C4_IMPORT_CANARY_MODULE"
	importCanarySocketEnvironment = "COUNTERSHAPE_C4_IMPORT_CANARY_SOCKET"
	serviceCanaryEnvironment      = "COUNTERSHAPE_C4_SERVICE_CANARY_SOCKET"
	parentSecretSentinel          = "COUNTERSHAPE_C4_PARENT_SECRET_SENTINEL"
	parentCredentialSentinel      = "COUNTERSHAPE_C4_PARENT_CREDENTIAL_SENTINEL"
)

var parentSentinelRoster = [...]string{parentCredentialSentinel, parentSecretSentinel}

type cliExecutionInput struct {
	target contractexec.OfficialTarget
	cliExecutionPreparation
}

// cliExecutionPreparation contains only inert physical and evidence inputs.
// The live OfficialTarget stays in the caller that received it from the sole
// issuer and is joined to this preparation only inside executeCLI.
type cliExecutionPreparation struct {
	targetIdentity     preparedTargetIdentity
	source             contractsource.PortableSource
	view               contractsource.CLISourceView
	prepared           *processmechanics.Prepared
	binding            domain.Digest
	environment        []string
	stdin              processmechanics.Stdin
	scope              *scopeProbe
	pre                inventorySnapshot
	evidenceCapacity   cliEvidenceCapacity
	attemptID          string
	evidenceRoot       string
	evidenceAuthority  invocationEvidenceAuthority
	logicalArgv        []string
	invocationEvidence invocationEvidence

	sentinelInherited bool
	nodePathDeclared  bool
}

func prepareCLIExecution(target contractexec.OfficialTarget) (cliExecutionPreparation, error) {
	model := target.Model()
	if !model.Valid() {
		return cliExecutionPreparation{}, refuse(CodeInvalidRequest, "a fresh official target is required", nil)
	}
	bundle := target.ContractBundle()
	if !bundle.Valid() {
		return cliExecutionPreparation{}, refuse(CodeTargetChanged, "target bundle is invalid", nil)
	}
	source := bundle.PortableSource()
	view, ok := source.CLIView()
	if !ok || !source.Valid() || !view.Stimulus().Valid() || !view.Capture().Valid() || !view.Projection().Valid() {
		return cliExecutionPreparation{}, refuse(CodeUnsupportedProfile, "target is not an exact CLI portable source", nil)
	}
	if source.Entrypoint() == "" || source.StartProfile() != contractsource.CLIStartProfileV1 ||
		source.Plan().Adapter().Domain != domain.AdapterCLI || source.Plan().ExecutionShape() != domain.OneCLIInvocation {
		return cliExecutionPreparation{}, refuse(CodeUnsupportedProfile, "CLI source profile is outside the C4 runner", nil)
	}
	if view.Stimulus().CWDPolicy() != cli.CWDMaterializedRoot || view.Stimulus().Executable() != "node" {
		return cliExecutionPreparation{}, refuse(CodeUnsupportedProfile, "CLI launch shape is outside the candidate-root Node profile", nil)
	}
	roots := target.Roots()
	identity, err := newPreparedTargetIdentity(model, bundle, roots)
	if err != nil {
		return cliExecutionPreparation{}, refuse(CodeTargetChanged, "target roots are unavailable", nil)
	}
	if err := materializeCLIFixtures(roots.FixtureRoot(), view.Stimulus().Fixtures()); err != nil {
		return cliExecutionPreparation{}, refuse(CodeFixtureRejected, "CLI fixtures did not materialize exactly", err)
	}
	evidenceRoot := roots.EvidenceRoot()
	evidenceAuthority, err := preflightCLIInvocationEvidence(evidenceRoot, roots.MarkerPath())
	if err != nil {
		return cliExecutionPreparation{}, refuse(CodeFixtureRejected, "child evidence root is not ready for one fixed receipt", err)
	}
	attemptID, err := newCLIRuntimeAttemptID(model.Input().Attempt.ArtifactDigest)
	if err != nil {
		return cliExecutionPreparation{}, refuse(CodeInvalidRequest, "fresh CLI attempt identity could not be allocated", err)
	}
	probe, err := newScopeProbe(roots)
	if err != nil {
		return cliExecutionPreparation{}, refuse(CodeEvidenceClosureFailed, "standalone canaries could not be armed", err)
	}
	environment, sentinelInherited, nodePathDeclared, err := buildCLIEnvironment(
		source, view.Stimulus(), roots, attemptID, evidenceRoot, probe,
	)
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, err
	}
	pre, err := snapshotCandidate(identity.candidateRoot)
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, refuse(CodeEvidenceClosureFailed, "candidate inventory could not be measured before start", err)
	}
	evidenceCapacity, err := preflightPrivateEvidenceCapacity(
		model, pre, view.Capture().StdoutBytes(), view.Capture().StderrBytes(),
	)
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, err
	}
	stdin := processmechanics.AbsentStdin()
	if value := view.Stimulus().Stdin(); value.Present() {
		stdin = processmechanics.PresentStdin(value.Bytes())
	}
	logicalArgv := view.Stimulus().LogicalArgv()
	if len(logicalArgv) == 0 || logicalArgv[0] != "node" {
		_ = probe.close()
		return cliExecutionPreparation{}, refuse(CodeUnsupportedProfile, "logical CLI argv is outside the Node profile", nil)
	}
	budgets := source.Plan().Budgets()
	invocation, err := processmechanics.NewInvocation(
		model.Input().Runtime.AdmittedExecutablePath,
		logicalArgv,
		environment,
		stdin,
		identity.candidateRoot,
		processmechanics.Limits{
			StdoutBytes: view.Capture().StdoutBytes(),
			StderrBytes: view.Capture().StderrBytes(),
			Execution:   time.Duration(budgets.ProbeMS) * time.Millisecond,
			Teardown:    time.Duration(budgets.TeardownMS) * time.Millisecond,
		},
	)
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, refuse(CodeInvalidRequest, "physical CLI invocation is invalid", err)
	}
	prepared, err := processmechanics.Prepare(invocation)
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, refuse(CodeInvalidRequest, "physical CLI invocation could not be frozen", err)
	}
	binding, err := domain.ParseDigest(prepared.BindingDigest().String())
	if err != nil {
		_ = probe.close()
		return cliExecutionPreparation{}, refuse(CodeInvalidRequest, "physical invocation binding is invalid", err)
	}
	return cliExecutionPreparation{
		targetIdentity: identity,
		source:         source, view: view, prepared: prepared, binding: binding,
		environment: environment, stdin: stdin, scope: probe, pre: pre, evidenceCapacity: evidenceCapacity,
		attemptID: attemptID, evidenceRoot: evidenceRoot, evidenceAuthority: evidenceAuthority,
		logicalArgv:       append([]string(nil), logicalArgv...),
		sentinelInherited: sentinelInherited, nodePathDeclared: nodePathDeclared,
	}, nil
}

func newCLIRuntimeAttemptID(forbidden domain.Digest) (string, error) {
	forbiddenID := ""
	if forbidden.Valid() {
		forbiddenID = "attempt:" + strings.TrimPrefix(forbidden.String(), "sha256:")
	}
	for {
		var random [32]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", err
		}
		candidate := "attempt:" + hex.EncodeToString(random[:])
		if candidate != forbiddenID {
			return candidate, nil
		}
	}
}

func (input *cliExecutionInput) closeProbe() error {
	if input == nil || input.scope == nil {
		return nil
	}
	err := input.scope.close()
	input.scope = nil
	return err
}

func buildCLIEnvironment(
	source contractsource.PortableSource,
	stimulus climodel.CLIStimulus,
	roots store.ConformanceAttemptRoots,
	attemptID string,
	evidenceRoot string,
	probe *scopeProbe,
) ([]string, bool, bool, error) {
	if probe == nil || !stimulus.Valid() || !source.Valid() || !validCLIRuntimeAttemptID(attemptID) ||
		evidenceRoot != roots.EvidenceRoot() {
		return nil, false, false, refuse(CodeInvalidRequest, "CLI environment inputs are invalid", nil)
	}
	values := make(map[string]string, len(source.Plan().Environment())+len(stimulus.Environment())+20)
	put := func(name, value string) error {
		if name == "" || strings.ContainsAny(name, "=\x00") || strings.ContainsRune(value, '\x00') {
			return refuse(CodeInvalidRequest, "CLI environment contains invalid text", nil)
		}
		if _, duplicate := values[name]; duplicate {
			return refuse(CodeInvalidRequest, "CLI environment repeats a name", nil)
		}
		values[name] = value
		return nil
	}
	for _, entry := range source.Plan().Environment() {
		if parentSentinelName(entry.Name) {
			return nil, false, false, refuse(CodeInvalidRequest, "CLI source environment collides with a parent sentinel name", nil)
		}
		if err := put(entry.Name, entry.Value); err != nil {
			return nil, false, false, err
		}
	}
	for _, binding := range stimulus.Environment() {
		if binding.Present() {
			if parentSentinelName(binding.Name()) {
				return nil, false, false, refuse(CodeInvalidRequest, "CLI stimulus environment collides with a parent sentinel name", nil)
			}
			if err := put(binding.Name(), binding.Value()); err != nil {
				return nil, false, false, err
			}
		}
	}
	fixed := []struct{ name, value string }{
		{"HOME", roots.HomeRoot()},
		{"TMPDIR", roots.TemporaryRoot()},
		{"XDG_CONFIG_HOME", roots.XDGConfigRoot()},
		{"XDG_CACHE_HOME", roots.XDGCacheRoot()},
		{"XDG_DATA_HOME", roots.XDGDataRoot()},
		{"XDG_STATE_HOME", roots.XDGStateRoot()},
		{stateRootEnvironment, roots.StateRoot()},
		{evidenceRootEnvironment, evidenceRoot},
		{fixtureRootEnvironment, roots.FixtureRoot()},
		{attemptIDEnvironment, attemptID},
		{scheduleEnvironment, "0"},
		{repetitionEnvironment, "0"},
		{importCanaryModuleEnvironment, probe.importModule},
		{importCanarySocketEnvironment, probe.importCanary.path},
		{serviceCanaryEnvironment, probe.serviceCanary.path},
		{"__CF_USER_TEXT_ENCODING", fmt.Sprintf("0x%X:0x0:0x0", os.Getuid())},
	}
	for _, entry := range fixed {
		if err := put(entry.name, entry.value); err != nil {
			return nil, false, false, err
		}
	}
	sentinelInherited := false
	for _, name := range parentSentinelRoster {
		if _, present := values[name]; present {
			sentinelInherited = true
		}
	}
	_, nodePathDeclared := values["NODE_PATH"]
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]string, len(names))
	for index, name := range names {
		result[index] = name + "=" + values[name]
	}
	return result, sentinelInherited, nodePathDeclared, nil
}

func parentSentinelName(name string) bool {
	for _, sentinel := range parentSentinelRoster {
		if name == sentinel {
			return true
		}
	}
	return false
}

func materializeCLIFixtures(root string, fixtures []climodel.CLIFixtureFile) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("fixture root is not the exact private directory")
	}
	for _, fixture := range fixtures {
		if !fixturePathValid(fixture.Path()) {
			return errors.New("fixture path escaped the private root")
		}
		destination := filepath.Join(root, filepath.FromSlash(fixture.Path()))
		if err := ensureFixtureParents(root, filepath.Dir(destination)); err != nil {
			return err
		}
		permissions := os.FileMode(0o600)
		if fixture.Mode() == cli.FixtureMode0644 {
			permissions = 0o644
		}
		file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
		if err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
			if err := validateExistingCLIFixture(destination, permissions, fixture.Contents()); err != nil {
				return errors.Join(err, errors.New("existing fixture differs from the exact immutable overlay"))
			}
			continue
		}
		body := fixture.Contents()
		modeErr := file.Chmod(permissions)
		writeErr := writeExactCLIFixture(file, body)
		syncErr := file.Sync()
		closeErr := file.Close()
		if modeErr != nil || writeErr != nil || syncErr != nil || closeErr != nil {
			return errors.Join(modeErr, writeErr, syncErr, closeErr, errors.New("fixture write was incomplete"))
		}
		if err := validateExistingCLIFixture(destination, permissions, body); err != nil {
			return errors.Join(err, errors.New("fixture bytes or facts did not reopen exactly"))
		}
	}
	return nil
}

func writeExactCLIFixture(file *os.File, body []byte) error {
	for offset := 0; offset < len(body); {
		count, err := file.Write(body[offset:])
		offset += count
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func validateExistingCLIFixture(path string, permissions os.FileMode, expected []byte) error {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 ||
		before.Mode().Perm() != permissions || before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		before.Size() != int64(len(expected)) {
		return errors.Join(err, errors.New("fixture leaf facts differ before bounded reopen"))
	}
	descriptor, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(descriptor), path)
	if handle == nil {
		_ = syscall.Close(descriptor)
		return errors.New("fixture descriptor could not be owned")
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Mode() != before.Mode() ||
		opened.Size() != before.Size() || !os.SameFile(before, opened) {
		_ = handle.Close()
		return errors.Join(statErr, errors.New("fixture descriptor facts differ"))
	}
	reopened, readErr := io.ReadAll(io.LimitReader(handle, int64(len(expected))+1))
	afterDescriptor, afterErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(path)
	if readErr != nil || afterErr != nil || closeErr != nil || pathErr != nil ||
		len(reopened) != len(expected) || !bytes.Equal(reopened, expected) ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) ||
		afterPath.Mode() != opened.Mode() || afterPath.Size() != opened.Size() {
		return errors.Join(readErr, afterErr, closeErr, pathErr, errors.New("fixture bytes or identity changed during bounded reopen"))
	}
	return nil
}

func fixturePathValid(value string) bool {
	return value != "" && len(value) <= 1024 && filepath.IsLocal(filepath.FromSlash(value)) &&
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))) == value &&
		!strings.HasPrefix(value, ".") && !strings.Contains(value, "/.")
}

func ensureFixtureParents(root, destination string) error {
	relative, err := filepath.Rel(root, destination)
	if err != nil || relative == "." {
		return err
	}
	cursor := root
	for _, component := range strings.Split(filepath.ToSlash(relative), "/") {
		if component == "" || component == "." || component == ".." {
			return errors.New("fixture parent path is invalid")
		}
		cursor = filepath.Join(cursor, component)
		if err := os.Mkdir(cursor, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err := os.Lstat(cursor)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
			return errors.Join(err, errors.New("fixture parent is not an exact private directory"))
		}
	}
	return nil
}
