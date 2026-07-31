//go:build darwin

package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/contractexec/scope"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodemodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	fixtureRootEnvironment  = "COUNTERSHAPE_FIXTURE_ROOT"
	stateRootEnvironment    = "COUNTERSHAPE_STATE_ROOT"
	evidenceRootEnvironment = "COUNTERSHAPE_EVIDENCE_ROOT"
	attemptIDEnvironment    = "COUNTERSHAPE_ATTEMPT_ID"
	scheduleEnvironment     = "COUNTERSHAPE_SCHEDULE_ORDINAL"
	repetitionEnvironment   = "COUNTERSHAPE_SCHEDULE_REPETITION"
	stimulusEnvironment     = "COUNTERSHAPE_HTTP_STIMULUS_DIGEST"
	readinessFDEnvironment  = "COUNTERSHAPE_HTTP_READINESS_FD"
	listenerFDEnvironment   = "COUNTERSHAPE_HTTP_LISTEN_FD"
	listenerPortEnvironment = "COUNTERSHAPE_HTTP_PORT"
	invocationFilename      = "http-invocation.json"
	seedStagingDirectory    = ".countershape-http-seed-staging"
	maxInvocationBytes      = int64(64 << 10)
)

type executionInput struct {
	target contractexec.OfficialTarget
	executionPreparation
}

type executionPreparation struct {
	targetIdentity preparedTargetIdentity
	source         contractsource.PortableSource
	view           contractsource.HTTPSourceView
	prepared       *preparedService
	seedLease      seedMaterializationLease
	binding        domain.Digest
	environment    []string
	probe          *scope.Probe
	before         scope.Inventory
	capacity       evidenceCapacity
	attemptID      string
	evidenceRoot   string
	receipt        receiptAuthority
	receiptFinding invocationFinding

	sentinelInherited bool
	nodePathDeclared  bool
}

type receiptAuthority struct {
	root   os.FileInfo
	marker os.FileInfo
}

var errSeedMaterializationBusy = errors.New("HTTP seed materialization is already active")

type seedMaterializationLease struct {
	root *os.File
}

func (authority receiptAuthority) valid() bool {
	return authority.root != nil && authority.root.IsDir() && authority.root.Mode().Perm() == 0o700 &&
		authority.marker != nil && authority.marker.Mode().IsRegular()
}

func prepareHTTPExecution(
	target contractexec.OfficialTarget,
) (preparation executionPreparation, resultErr error) {
	model := target.Model()
	bundle := target.ContractBundle()
	if !model.Valid() || !bundle.Valid() || bundle.Digest() != model.ContractBundleDigest() {
		return executionPreparation{}, refuse(CodeInvalidRequest, "a fresh exact target and bundle are required", nil)
	}
	source := bundle.PortableSource()
	view, err := requireStandaloneHTTPProfile(source, bundle.SourceProfile())
	if err != nil {
		return executionPreparation{}, err
	}
	roots := target.Roots()
	identity, err := newPreparedTargetIdentity(model, bundle, roots)
	if err != nil || target.CandidateRoot() != identity.candidateRoot {
		return executionPreparation{}, refuse(CodeTargetChanged, "target roots are unavailable or disagree", err)
	}
	seedLease, err := materializeHTTPSeeds(
		roots.FixtureRoot(),
		roots.TemporaryRoot(),
		view.Stimulus().Seeds(),
	)
	if err != nil {
		if errors.Is(err, errSeedMaterializationBusy) {
			return executionPreparation{}, refuse(
				CodeAdmissionRefused,
				"another HTTP preparation owns seed materialization",
				err,
			)
		}
		return executionPreparation{}, refuse(CodeFixtureRejected, "HTTP seeds did not materialize exactly", err)
	}
	transferSeedLease := false
	defer func() {
		if !transferSeedLease {
			resultErr = errors.Join(resultErr, seedLease.release())
		}
	}()
	receipt, err := preflightInvocationEvidence(roots.EvidenceRoot(), roots.MarkerPath())
	if err != nil {
		return executionPreparation{}, refuse(CodeFixtureRejected, "HTTP evidence root is not ready for one fixed receipt", err)
	}
	attemptID, err := newRuntimeAttemptID(model.Input().Attempt.ArtifactDigest)
	if err != nil {
		return executionPreparation{}, refuse(CodeInvalidRequest, "fresh HTTP attempt identity could not be allocated", err)
	}
	probe, err := scope.NewProbe(roots.TemporaryRoot())
	if err != nil {
		return executionPreparation{}, refuse(CodeEvidenceClosureFailed, "standalone scope probes could not be armed", err)
	}
	closeProbe := true
	defer func() {
		if closeProbe {
			_, _ = probe.Finish(nil)
		}
	}()
	environment, sentinelInherited, nodePathDeclared, err := buildHTTPEnvironment(
		source, roots, attemptID, probe,
	)
	if err != nil {
		return executionPreparation{}, err
	}
	before, err := scope.Snapshot(identity.candidateRoot)
	if err != nil {
		return executionPreparation{}, refuse(CodeEvidenceClosureFailed, "candidate inventory could not be measured before start", err)
	}
	capacity, err := preflightEvidenceCapacity(source, before)
	if err != nil {
		return executionPreparation{}, err
	}
	prepared, binding, err := prepareHTTPService(servicePreparation{
		executable:  model.Input().Runtime.AdmittedExecutablePath,
		runtime:     model.Input().Runtime,
		logicalArgv: view.Start().LogicalArgv(),
		environment: environment,
		cwd:         identity.candidateRoot,
		stimulus:    view.Stimulus(),
		capture:     view.Capture(),
		readiness:   view.Readiness(),
		budgets:     source.Plan().Budgets(),
		sourceBind:  source.ExecutionBindingDigest(),
	})
	if err != nil {
		return executionPreparation{}, refuse(CodeInvalidRequest, "HTTP service could not be frozen", err)
	}
	closeProbe = false
	preparation = executionPreparation{
		targetIdentity: identity, source: source, view: view, prepared: prepared,
		seedLease: seedLease, binding: binding,
		environment: environment, probe: probe, before: before, capacity: capacity,
		attemptID: attemptID, evidenceRoot: roots.EvidenceRoot(), receipt: receipt,
		sentinelInherited: sentinelInherited, nodePathDeclared: nodePathDeclared,
	}
	transferSeedLease = true
	return preparation, nil
}

func requireStandaloneHTTPProfile(
	source contractsource.PortableSource,
	profile nodemodel.SourceProfile,
) (contractsource.HTTPSourceView, error) {
	view, ok := source.HTTPView()
	if !ok || !source.Valid() || !profile.ValidFor(source) ||
		profile.AdapterDomain() != domain.AdapterHTTP || !view.Valid() ||
		source.Adapter() != domain.AdapterHTTP ||
		source.Plan().Adapter().Domain != domain.AdapterHTTP ||
		source.Plan().ExecutionShape() != domain.OneLoopbackHTTPRequest ||
		source.StartProfile() != httpmodel.HTTPPortableStartAuthorityV1 ||
		view.Start().Authority() != httpmodel.HTTPPortableStartAuthorityV1 ||
		view.Readiness().Protocol() != httpmodel.PortableReadinessProtocolV1 ||
		len(source.Plan().SetupArgv()) != 0 || len(source.Plan().SecretSlots()) != 0 ||
		source.Entrypoint() != view.Start().Entrypoint() ||
		view.Start().Executable() != "node" {
		return contractsource.HTTPSourceView{}, refuse(
			CodeUnsupportedProfile,
			"source is outside the standalone child-bind HTTP profile",
			nil,
		)
	}
	return view, nil
}

func (input *executionInput) closePrepared() error {
	if input == nil {
		return nil
	}
	var result error
	if input.prepared != nil {
		result = errors.Join(result, input.prepared.Close())
		input.prepared = nil
	}
	if input.probe != nil {
		_, probeErr := input.probe.Finish(nil)
		result = errors.Join(result, probeErr)
		input.probe = nil
	}
	result = errors.Join(result, input.seedLease.release())
	return result
}

func (input *executionInput) finishProbe(ctx context.Context) (scope.Measurements, error) {
	if input == nil || input.probe == nil {
		return scope.Measurements{
				Ambiguous:  true,
				Diagnostic: scope.NewDiagnostic(scope.CodeProbeAbsent),
			},
			&scope.Error{Diagnostic: scope.NewDiagnostic(scope.CodeProbeAbsent)}
	}
	measurements, err := input.probe.Finish(ctx)
	input.probe = nil
	return measurements, err
}

func newRuntimeAttemptID(forbidden domain.Digest) (string, error) {
	forbiddenID := ""
	if forbidden.Valid() {
		forbiddenID = "attempt:" + strings.TrimPrefix(forbidden.String(), "sha256:")
	}
	for {
		var random [sha256.Size]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", err
		}
		candidate := "attempt:" + hex.EncodeToString(random[:])
		if candidate != forbiddenID {
			return candidate, nil
		}
	}
}

func validRuntimeAttemptID(value string) bool {
	if len(value) != len("attempt:")+sha256.Size*2 || !strings.HasPrefix(value, "attempt:") {
		return false
	}
	text := strings.TrimPrefix(value, "attempt:")
	decoded, err := hex.DecodeString(text)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(text) == text
}

func buildHTTPEnvironment(
	source contractsource.PortableSource,
	roots store.ConformanceAttemptRoots,
	attemptID string,
	probe *scope.Probe,
) ([]string, bool, bool, error) {
	if !source.Valid() || !validRuntimeAttemptID(attemptID) || probe == nil {
		return nil, false, false, refuse(CodeInvalidRequest, "HTTP environment inputs are incomplete", nil)
	}
	values := make(map[string]string, len(source.Plan().Environment())+24)
	put := func(name, value string) error {
		if name == "" || strings.ContainsAny(name, "=\x00") || strings.ContainsRune(value, '\x00') {
			return refuse(CodeInvalidRequest, "HTTP environment contains invalid text", nil)
		}
		if _, duplicate := values[name]; duplicate {
			return refuse(CodeInvalidRequest, "HTTP environment repeats a name", nil)
		}
		values[name] = value
		return nil
	}
	for _, entry := range source.Plan().Environment() {
		if scope.IsParentSentinel(entry.Name) || isOwnedEnvironmentName(entry.Name) {
			return nil, false, false, refuse(CodeInvalidRequest, "HTTP plan environment collides with runner-owned authority", nil)
		}
		if err := put(entry.Name, entry.Value); err != nil {
			return nil, false, false, err
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
		{evidenceRootEnvironment, roots.EvidenceRoot()},
		{fixtureRootEnvironment, roots.FixtureRoot()},
		{attemptIDEnvironment, attemptID},
		{scheduleEnvironment, "0"},
		{repetitionEnvironment, "0"},
		{stimulusEnvironment, source.StimulusDigest().String()},
		{readinessFDEnvironment, "3"},
		{"__CF_USER_TEXT_ENCODING", fmt.Sprintf("0x%X:0x0:0x0", os.Getuid())},
	}
	for name, value := range probe.Environment() {
		fixed = append(fixed, struct{ name, value string }{name, value})
	}
	for _, entry := range fixed {
		if err := put(entry.name, entry.value); err != nil {
			return nil, false, false, err
		}
	}
	sentinelInherited := false
	for _, name := range scope.ParentSentinels() {
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
	environment := make([]string, len(names))
	for index, name := range names {
		environment[index] = name + "=" + values[name]
	}
	return environment, sentinelInherited, nodePathDeclared, nil
}

func isOwnedEnvironmentName(name string) bool {
	switch name {
	case "HOME", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
		fixtureRootEnvironment, stateRootEnvironment, evidenceRootEnvironment, attemptIDEnvironment,
		scheduleEnvironment, repetitionEnvironment, stimulusEnvironment, readinessFDEnvironment,
		listenerFDEnvironment, listenerPortEnvironment, scope.ImportModuleEnvironment,
		scope.ImportSocketEnvironment, scope.ServiceSocketEnvironment, "__CF_USER_TEXT_ENCODING":
		return true
	default:
		return false
	}
}

type seedEntryExpectation struct {
	relative  string
	directory bool
	contents  []byte
}

type seedTopology struct {
	children map[string]map[string]seedEntryExpectation
}

func materializeHTTPSeeds(
	fixtureRoot string,
	temporaryRoot string,
	seeds []httpmodel.HTTPSeedFile,
) (lease seedMaterializationLease, resultErr error) {
	fixtureInfo, err := os.Lstat(fixtureRoot)
	if err != nil || !exactSeedDirectory(fixtureInfo) {
		return seedMaterializationLease{}, errors.Join(
			err,
			errors.New("fixture root is not the exact private directory"),
		)
	}
	temporaryInfo, err := os.Lstat(temporaryRoot)
	if err != nil || !exactSeedDirectory(temporaryInfo) ||
		!sameSeedDevice(fixtureInfo, temporaryInfo) {
		return seedMaterializationLease{}, errors.Join(
			err,
			errors.New("seed staging root is not an exact same-device private directory"),
		)
	}
	topology, err := buildSeedTopology(seeds)
	if err != nil {
		return seedMaterializationLease{}, err
	}
	ownedLease, err := acquireSeedMaterializationLease(temporaryRoot, temporaryInfo)
	if err != nil {
		return seedMaterializationLease{}, err
	}
	releaseOnReturn := true
	defer func() {
		if releaseOnReturn {
			resultErr = errors.Join(resultErr, ownedLease.release())
			lease = seedMaterializationLease{}
		}
	}()
	if err := validateSeedFixtureRoster(
		fixtureRoot,
		fixtureInfo,
		topology,
		false,
	); err != nil {
		return seedMaterializationLease{}, errors.Join(
			err,
			errors.New("fixture contains foreign or contradictory seed residue"),
		)
	}
	stagingRoot, stagingInfo, err := createSeedStagingRoot(
		temporaryRoot,
		temporaryInfo,
	)
	if err != nil {
		return seedMaterializationLease{}, err
	}
	for _, seed := range seeds {
		destination := filepath.Join(fixtureRoot, filepath.FromSlash(seed.Path()))
		if err := ensureSeedParents(fixtureRoot, filepath.Dir(destination)); err != nil {
			cleanupErr := retireSeedStagingRoot(
				temporaryRoot,
				temporaryInfo,
				stagingRoot,
				stagingInfo,
			)
			return seedMaterializationLease{}, errors.Join(err, cleanupErr)
		}
		if err := createOrValidateSeed(
			stagingRoot,
			destination,
			seed.Contents(),
		); err != nil {
			cleanupErr := retireSeedStagingRoot(
				temporaryRoot,
				temporaryInfo,
				stagingRoot,
				stagingInfo,
			)
			return seedMaterializationLease{}, errors.Join(err, cleanupErr)
		}
	}
	if err := retireSeedStagingRoot(
		temporaryRoot,
		temporaryInfo,
		stagingRoot,
		stagingInfo,
	); err != nil {
		return seedMaterializationLease{}, err
	}
	if err := validateSeedFixtureRoster(fixtureRoot, fixtureInfo, topology, true); err != nil {
		return seedMaterializationLease{}, err
	}
	releaseOnReturn = false
	return ownedLease, nil
}

func acquireSeedMaterializationLease(
	temporaryRoot string,
	retainedTemporary os.FileInfo,
) (seedMaterializationLease, error) {
	if !exactSeedDirectory(retainedTemporary) {
		return seedMaterializationLease{}, errors.New("seed materialization parent authority is invalid")
	}
	fd, err := syscall.Open(
		temporaryRoot,
		syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|
			darwinOpenNoFollowAny|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return seedMaterializationLease{}, err
	}
	root := os.NewFile(uintptr(fd), temporaryRoot)
	if root == nil {
		_ = syscall.Close(fd)
		return seedMaterializationLease{}, errors.New("seed materialization root descriptor could not be owned")
	}
	locked := false
	fail := func(cause error) (seedMaterializationLease, error) {
		var unlockErr error
		if locked {
			unlockErr = syscall.Flock(fd, syscall.LOCK_UN)
		}
		closeErr := root.Close()
		return seedMaterializationLease{}, errors.Join(cause, unlockErr, closeErr)
	}
	opened, descriptorErr := root.Stat()
	pathInfo, pathErr := os.Lstat(temporaryRoot)
	if descriptorErr != nil || pathErr != nil ||
		!exactSeedDirectory(opened) || !exactSeedDirectory(pathInfo) ||
		opened.Mode() != retainedTemporary.Mode() ||
		pathInfo.Mode() != retainedTemporary.Mode() ||
		!os.SameFile(retainedTemporary, opened) ||
		!os.SameFile(opened, pathInfo) {
		return fail(errors.Join(
			descriptorErr,
			pathErr,
			errors.New("seed materialization root changed while opening"),
		))
	}
	if lockErr := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); lockErr != nil {
		if errors.Is(lockErr, syscall.EWOULDBLOCK) ||
			errors.Is(lockErr, syscall.EAGAIN) {
			return fail(errors.Join(errSeedMaterializationBusy, lockErr))
		}
		return fail(lockErr)
	}
	locked = true
	afterDescriptor, descriptorErr := root.Stat()
	afterPath, pathErr := os.Lstat(temporaryRoot)
	if descriptorErr != nil || pathErr != nil ||
		!exactSeedDirectory(afterDescriptor) ||
		!exactSeedDirectory(afterPath) ||
		afterDescriptor.Mode() != opened.Mode() ||
		afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) ||
		!os.SameFile(opened, afterPath) {
		return fail(errors.Join(
			descriptorErr,
			pathErr,
			errors.New("seed materialization root changed while acquiring its lease"),
		))
	}
	return seedMaterializationLease{root: root}, nil
}

func (lease *seedMaterializationLease) release() error {
	if lease == nil || lease.root == nil {
		return nil
	}
	root := lease.root
	lease.root = nil
	unlockErr := syscall.Flock(int(root.Fd()), syscall.LOCK_UN)
	closeErr := root.Close()
	return errors.Join(unlockErr, closeErr)
}

func buildSeedTopology(seeds []httpmodel.HTTPSeedFile) (seedTopology, error) {
	topology := seedTopology{
		children: map[string]map[string]seedEntryExpectation{
			"": {},
		},
	}
	for _, seed := range seeds {
		path := seed.Path()
		if !validSeedPath(path) || seed.Mode() != httpmodel.SeedMode0644 {
			return seedTopology{}, errors.New("HTTP seed roster contains an invalid path or mode")
		}
		components := strings.Split(path, "/")
		parent := ""
		for index, component := range components {
			relative := component
			if parent != "" {
				relative = parent + "/" + component
			}
			directory := index < len(components)-1
			expected := seedEntryExpectation{
				relative:  relative,
				directory: directory,
			}
			if !directory {
				expected.contents = seed.Contents()
			}
			children := topology.children[parent]
			if prior, exists := children[component]; exists {
				if prior.directory != expected.directory || !directory {
					return seedTopology{}, errors.New("HTTP seed roster collides by file or directory path")
				}
			} else {
				children[component] = expected
			}
			if directory {
				if topology.children[relative] == nil {
					topology.children[relative] = make(map[string]seedEntryExpectation)
				}
				parent = relative
			}
		}
	}
	return topology, nil
}

func validateSeedFixtureRoster(
	root string,
	retainedRoot os.FileInfo,
	topology seedTopology,
	exact bool,
) error {
	if !exactSeedDirectory(retainedRoot) || topology.children == nil {
		return errors.New("seed fixture roster authority is invalid")
	}
	return validateSeedFixtureDirectory(root, "", retainedRoot, topology, exact)
}

func validateSeedFixtureDirectory(
	root string,
	relative string,
	retainedRoot os.FileInfo,
	topology seedTopology,
	exact bool,
) error {
	path := root
	if relative != "" {
		path = filepath.Join(root, filepath.FromSlash(relative))
	}
	before, err := os.Lstat(path)
	if err != nil || !exactSeedDirectory(before) ||
		(relative == "" && !os.SameFile(retainedRoot, before)) {
		return errors.Join(err, errors.New("seed fixture directory identity differs"))
	}
	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|
			darwinOpenNoFollowAny|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(fd), path)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("seed fixture directory descriptor could not be owned")
	}
	opened, statErr := handle.Stat()
	expectedChildren, present := topology.children[relative]
	if statErr != nil || !present || !exactSeedDirectory(opened) ||
		opened.Mode() != before.Mode() || !os.SameFile(before, opened) {
		_ = handle.Close()
		return errors.Join(statErr, errors.New("seed fixture directory changed while opening"))
	}
	entries := make([]os.DirEntry, 0, len(expectedChildren))
	var readErr error
	for {
		batch, batchErr := handle.ReadDir(len(expectedChildren) + 1 - len(entries))
		entries = append(entries, batch...)
		if len(entries) > len(expectedChildren) {
			_ = handle.Close()
			return errors.New("seed fixture direct roster exceeds its declared bound")
		}
		if errors.Is(batchErr, io.EOF) {
			readErr = batchErr
			break
		}
		if batchErr != nil || len(batch) == 0 {
			_ = handle.Close()
			return errors.Join(batchErr, errors.New("seed fixture direct roster could not close"))
		}
	}
	if !errors.Is(readErr, io.EOF) ||
		(exact && len(entries) != len(expectedChildren)) {
		_ = handle.Close()
		return errors.Join(readErr, errors.New("seed fixture direct roster differs"))
	}
	observed := make(map[string]struct{}, len(entries))
	childDirectories := make([]string, 0, len(entries))
	for _, listed := range entries {
		name := listed.Name()
		expected, ok := expectedChildren[name]
		if !ok || name == "" || name == "." || name == ".." ||
			strings.ContainsRune(name, filepath.Separator) {
			_ = handle.Close()
			return errors.New("seed fixture contains a foreign direct entry")
		}
		if _, duplicate := observed[name]; duplicate {
			_ = handle.Close()
			return errors.New("seed fixture directory repeated an entry")
		}
		observed[name] = struct{}{}
		entryPath := filepath.Join(root, filepath.FromSlash(expected.relative))
		listedInfo, listedErr := listed.Info()
		pathInfo, pathErr := os.Lstat(entryPath)
		if listedErr != nil || pathErr != nil ||
			listedInfo.Mode() != pathInfo.Mode() ||
			!os.SameFile(listedInfo, pathInfo) ||
			!sameSeedDevice(retainedRoot, listedInfo) {
			_ = handle.Close()
			return errors.Join(listedErr, pathErr, errors.New("seed fixture entry identity changed"))
		}
		if expected.directory {
			if !exactSeedDirectory(listedInfo) ||
				!sameSeedDevice(retainedRoot, listedInfo) {
				_ = handle.Close()
				return errors.New("seed fixture parent directory facts differ")
			}
			childDirectories = append(childDirectories, expected.relative)
			continue
		}
		if err := validateSeed(entryPath, expected.contents); err != nil {
			_ = handle.Close()
			return err
		}
	}
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(path)
	if descriptorErr != nil || closeErr != nil || pathErr != nil ||
		afterDescriptor.Mode() != opened.Mode() ||
		afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) ||
		!os.SameFile(opened, afterPath) {
		return errors.Join(
			descriptorErr,
			closeErr,
			pathErr,
			errors.New("seed fixture directory changed while reading"),
		)
	}
	sort.Strings(childDirectories)
	for _, child := range childDirectories {
		if err := validateSeedFixtureDirectory(
			root,
			child,
			retainedRoot,
			topology,
			exact,
		); err != nil {
			return err
		}
	}
	return nil
}

func createSeedStagingRoot(
	temporaryRoot string,
	retainedTemporary os.FileInfo,
) (string, os.FileInfo, error) {
	if !exactSeedDirectory(retainedTemporary) {
		return "", nil, errors.New("seed staging parent authority is invalid")
	}
	stagingRoot := filepath.Join(temporaryRoot, seedStagingDirectory)
	if err := os.Mkdir(stagingRoot, 0o700); err != nil {
		return "", nil, errors.Join(
			err,
			errors.New("exclusive seed staging directory already exists or could not be created"),
		)
	}
	stagingInfo, statErr := os.Lstat(stagingRoot)
	syncErr := syncSeedParent(temporaryRoot)
	afterTemporary, parentErr := os.Lstat(temporaryRoot)
	if statErr != nil || syncErr != nil || parentErr != nil ||
		!exactSeedDirectory(stagingInfo) ||
		!sameSeedDevice(retainedTemporary, stagingInfo) ||
		!exactSeedDirectory(afterTemporary) ||
		afterTemporary.Mode() != retainedTemporary.Mode() ||
		!os.SameFile(retainedTemporary, afterTemporary) {
		return "", nil, errors.Join(
			statErr,
			syncErr,
			parentErr,
			errors.New("seed staging directory did not publish exactly"),
		)
	}
	return stagingRoot, stagingInfo, nil
}

func retireSeedStagingRoot(
	temporaryRoot string,
	retainedTemporary os.FileInfo,
	stagingRoot string,
	retainedStaging os.FileInfo,
) error {
	before, err := os.Lstat(stagingRoot)
	if err != nil || !exactSeedDirectory(before) ||
		!os.SameFile(retainedStaging, before) {
		return errors.Join(err, errors.New("seed staging identity changed before retirement"))
	}
	fd, err := syscall.Open(
		stagingRoot,
		syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|
			darwinOpenNoFollowAny|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(fd), stagingRoot)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("seed staging descriptor could not be owned")
	}
	opened, statErr := handle.Stat()
	entries, readErr := handle.ReadDir(1)
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(stagingRoot)
	if statErr != nil || !errors.Is(readErr, io.EOF) || len(entries) != 0 ||
		descriptorErr != nil || closeErr != nil || pathErr != nil ||
		!exactSeedDirectory(opened) ||
		opened.Mode() != before.Mode() ||
		afterDescriptor.Mode() != opened.Mode() ||
		afterPath.Mode() != opened.Mode() ||
		!os.SameFile(before, opened) ||
		!os.SameFile(opened, afterDescriptor) ||
		!os.SameFile(opened, afterPath) {
		return errors.Join(
			statErr,
			readErr,
			descriptorErr,
			closeErr,
			pathErr,
			errors.New("seed staging directory is not exact and empty"),
		)
	}
	removeErr := os.Remove(stagingRoot)
	_, absenceErr := os.Lstat(stagingRoot)
	syncErr := syncSeedParent(temporaryRoot)
	afterTemporary, parentErr := os.Lstat(temporaryRoot)
	if removeErr != nil || !errors.Is(absenceErr, os.ErrNotExist) ||
		syncErr != nil || parentErr != nil ||
		!exactSeedDirectory(afterTemporary) ||
		afterTemporary.Mode() != retainedTemporary.Mode() ||
		!os.SameFile(retainedTemporary, afterTemporary) {
		return errors.Join(
			removeErr,
			absenceErr,
			syncErr,
			parentErr,
			errors.New("seed staging directory did not retire exactly"),
		)
	}
	return nil
}

func exactSeedDirectory(info os.FileInfo) bool {
	stat, ok := seedStat(info)
	return ok && info.Mode() == os.ModeDir|0o700 &&
		stat.Uid == uint32(os.Geteuid())
}

func exactSeedFile(info os.FileInfo, size int64) bool {
	stat, ok := seedStat(info)
	return ok && info.Mode() == 0o644 && info.Size() == size &&
		stat.Uid == uint32(os.Geteuid()) && stat.Nlink == 1
}

func seedStat(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func sameSeedDevice(left, right os.FileInfo) bool {
	leftStat, leftOK := seedStat(left)
	rightStat, rightOK := seedStat(right)
	return leftOK && rightOK && leftStat.Dev == rightStat.Dev
}

func validSeedPath(value string) bool {
	local := filepath.FromSlash(value)
	return value != "" && len(value) <= 4096 && filepath.IsLocal(local) &&
		filepath.ToSlash(filepath.Clean(local)) == value &&
		!strings.HasPrefix(value, ".") && !strings.Contains(value, "/.") &&
		value != ".git" && !strings.Contains(value, "/.git")
}

func ensureSeedParents(root, destination string) error {
	relative, err := filepath.Rel(root, destination)
	if err != nil {
		return err
	}
	if relative == "." {
		return nil
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !exactSeedDirectory(rootInfo) {
		return errors.Join(err, errors.New("seed root identity changed"))
	}
	cursor := root
	for _, component := range strings.Split(filepath.ToSlash(relative), "/") {
		if component == "" || component == "." || component == ".." {
			return errors.New("seed parent path is invalid")
		}
		parent := cursor
		cursor = filepath.Join(parent, component)
		mkdirErr := os.Mkdir(cursor, 0o700)
		if mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
			return mkdirErr
		}
		if mkdirErr == nil {
			if err := syncSeedParent(parent); err != nil {
				return err
			}
		}
		info, err := os.Lstat(cursor)
		if err != nil || !exactSeedDirectory(info) ||
			!sameSeedDevice(rootInfo, info) {
			return errors.Join(err, errors.New("seed parent is not an exact private directory"))
		}
	}
	return nil
}

func createOrValidateSeed(stagingRoot, path string, expected []byte) error {
	parent := filepath.Dir(path)
	file, err := os.CreateTemp(stagingRoot, ".seed.pending-")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	writeErr := writeExact(file, expected)
	chmodErr := file.Chmod(0o644)
	syncErr := file.Sync()
	temporaryInfo, statErr := file.Stat()
	closeErr := file.Close()
	if writeErr != nil || chmodErr != nil || syncErr != nil || statErr != nil || closeErr != nil {
		cleanupErr := removeExactTemporarySeed(temporaryPath, temporaryInfo)
		stagingSyncErr := syncSeedParent(stagingRoot)
		return errors.Join(
			writeErr,
			chmodErr,
			syncErr,
			statErr,
			closeErr,
			cleanupErr,
			stagingSyncErr,
		)
	}
	linkErr := os.Link(temporaryPath, path)
	if linkErr != nil && !errors.Is(linkErr, os.ErrExist) {
		cleanupErr := removeExactTemporarySeed(temporaryPath, temporaryInfo)
		stagingSyncErr := syncSeedParent(stagingRoot)
		return errors.Join(linkErr, cleanupErr, stagingSyncErr)
	}
	if linkErr == nil {
		published, publishErr := os.Lstat(path)
		if publishErr != nil || published.Mode() != temporaryInfo.Mode() ||
			published.Size() != temporaryInfo.Size() ||
			!os.SameFile(temporaryInfo, published) {
			return errors.Join(
				publishErr,
				errors.New("published seed is not the staged inode"),
			)
		}
	} else if validationErr := validateSeed(path, expected); validationErr != nil {
		cleanupErr := removeExactTemporarySeed(temporaryPath, temporaryInfo)
		stagingSyncErr := syncSeedParent(stagingRoot)
		return errors.Join(validationErr, cleanupErr, stagingSyncErr)
	}
	cleanupErr := removeExactTemporarySeed(temporaryPath, temporaryInfo)
	stagingSyncErr := syncSeedParent(stagingRoot)
	parentSyncErr := syncSeedParent(parent)
	validationErr := validateSeed(path, expected)
	return errors.Join(cleanupErr, stagingSyncErr, parentSyncErr, validationErr)
}

func removeExactTemporarySeed(path string, retained os.FileInfo) error {
	if retained == nil {
		return errors.New("temporary seed identity is absent")
	}
	current, err := os.Lstat(path)
	if err != nil || current.Mode() != retained.Mode() || current.Size() != retained.Size() ||
		!os.SameFile(retained, current) {
		return errors.Join(err, errors.New("temporary seed identity changed before removal"))
	}
	return os.Remove(path)
}

func syncSeedParent(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	return errors.Join(syncErr, closeErr)
}

func writeExact(file *os.File, body []byte) error {
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

func validateSeed(path string, expected []byte) error {
	before, err := os.Lstat(path)
	if err != nil || !exactSeedFile(before, int64(len(expected))) {
		return errors.Join(err, errors.New("seed leaf facts differ"))
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(fd), path)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("seed descriptor could not be owned")
	}
	opened, statErr := handle.Stat()
	body, readErr := io.ReadAll(io.LimitReader(handle, int64(len(expected))+1))
	after, afterErr := handle.Stat()
	closeErr := handle.Close()
	pathAfter, pathErr := os.Lstat(path)
	if statErr != nil || readErr != nil || afterErr != nil || closeErr != nil || pathErr != nil ||
		!exactSeedFile(opened, int64(len(expected))) ||
		!exactSeedFile(after, int64(len(expected))) ||
		!exactSeedFile(pathAfter, int64(len(expected))) ||
		opened.Mode() != before.Mode() ||
		opened.Size() != before.Size() || !os.SameFile(before, opened) ||
		!os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) ||
		!bytes.Equal(body, expected) {
		return errors.Join(statErr, readErr, afterErr, closeErr, pathErr, errors.New("seed bytes or identity changed"))
	}
	return nil
}

func preflightInvocationEvidence(root, markerPath string) (receiptAuthority, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || filepath.Dir(markerPath) != root {
		return receiptAuthority{}, errors.New("evidence root or marker path is not exact")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 ||
		rootInfo.Mode().Perm() != 0o700 {
		return receiptAuthority{}, errors.Join(err, errors.New("evidence root is not an exact private directory"))
	}
	if err := validateEvidenceRoster(
		context.Background(),
		root,
		rootInfo,
		[]string{filepath.Base(markerPath)},
	); err != nil {
		return receiptAuthority{}, errors.Join(err, errors.New("evidence root is not marker-only"))
	}
	marker, err := os.Lstat(markerPath)
	if err != nil || !marker.Mode().IsRegular() || marker.Mode()&os.ModeSymlink != 0 {
		return receiptAuthority{}, errors.Join(err, errors.New("attempt marker is not an exact regular file"))
	}
	if _, err := os.Lstat(filepath.Join(root, invocationFilename)); !errors.Is(err, os.ErrNotExist) {
		return receiptAuthority{}, errors.Join(err, errors.New("invocation receipt already exists"))
	}
	return receiptAuthority{root: rootInfo, marker: marker}, nil
}
