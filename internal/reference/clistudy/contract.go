package clistudy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	contractrunner "github.com/nelsonwerd/countershape/internal/contractexec/runner"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/reference"
)

const (
	officialCLITrialCount  = 10
	officialCLIWorkerLimit = 4
)

// officialTrial is an inert snapshot with separate sealed pure and live
// validators. Both closures capture inferred authorities without exposing
// protected model types or accepting caller-supplied result authority.
type officialTrial struct {
	attemptDigest                 domain.Digest
	targetDigest                  domain.Digest
	targetCanonicalSHA256         string
	bundleDigest                  domain.Digest
	residueHeadDigest             domain.Digest
	runDigest                     domain.Digest
	runCanonicalSHA256            string
	classificationDigest          domain.Digest
	classificationCanonicalSHA256 string
	result                        string
	exitCode                      int
	recoveryEqual                 bool
	validateSnapshot              func(officialTrial) error
	revalidate                    func(context.Context, officialTrial) error
}

// officialTrialSnapshot contains only inert scalars copied after the typed
// construction graph has been proven. Its comparisons cannot reach a store,
// filesystem, process, or capability-bearing value.
type officialTrialSnapshot struct {
	attemptDigest                 domain.Digest
	targetDigest                  domain.Digest
	targetCanonicalSHA256         string
	bundleDigest                  domain.Digest
	residueHeadDigest             domain.Digest
	runDigest                     domain.Digest
	runCanonicalSHA256            string
	classificationDigest          domain.Digest
	classificationCanonicalSHA256 string
	result                        string
	exitCode                      int
	recoveryEqual                 bool
}

func (snapshot officialTrialSnapshot) matches(current officialTrial) bool {
	return current.attemptDigest == snapshot.attemptDigest &&
		current.targetDigest == snapshot.targetDigest &&
		current.targetCanonicalSHA256 == snapshot.targetCanonicalSHA256 &&
		current.bundleDigest == snapshot.bundleDigest &&
		current.residueHeadDigest == snapshot.residueHeadDigest &&
		current.runDigest == snapshot.runDigest &&
		current.runCanonicalSHA256 == snapshot.runCanonicalSHA256 &&
		current.classificationDigest == snapshot.classificationDigest &&
		current.classificationCanonicalSHA256 == snapshot.classificationCanonicalSHA256 &&
		current.result == snapshot.result && current.exitCode == snapshot.exitCode &&
		current.recoveryEqual == snapshot.recoveryEqual
}

func newOfficialTrialSnapshotValidator(snapshot officialTrialSnapshot) func(officialTrial) error {
	return func(current officialTrial) error {
		if !snapshot.matches(current) {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_SNAPSHOT_GRAPH_REFUSED")
		}
		return nil
	}
}

func executeOfficialTrials(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
) (trials []officialTrial, returnErr error) {
	if ctx == nil || objectStore == nil || !residue.Valid() || !bundle.Valid() ||
		residue.BundleDigest() != bundle.Digest() || !cleanAbsolute(authorityRoot) ||
		!cleanAbsolute(config.RepositoryRoot) || !cleanAbsolute(config.GitExecutable) ||
		!cleanAbsolute(config.NodeExecutable) || config.Ordinal < 1 || config.Ordinal > 3 {
		return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_INPUT_REFUSED")
	}
	gitScratch, err := privateTempDirectory(
		authorityRoot,
		fmt.Sprintf("official-fixture-%d-", config.Ordinal),
	)
	if err != nil {
		return nil, err
	}
	fixture, err := reference.OpenCLIFixture(ctx, reference.CLIFixtureConfig{
		Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
		RepositoryCount: officialCLIWorkerLimit,
	})
	if err != nil {
		return nil, err
	}
	fixtureOpen := true
	defer func() {
		if fixtureOpen {
			returnErr = errors.Join(returnErr, fixture.Close())
		}
	}()
	displayRef, err := fixture.Ref(reference.ArgvFirst)
	if err != nil {
		return nil, err
	}
	repositories := fixture.Repositories()
	if len(repositories) != officialCLIWorkerLimit {
		return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_REPOSITORY_REFUSED")
	}
	expectedTree, err := repositories[0].Pin(ctx, displayRef)
	if err != nil || !expectedTree.Valid() {
		return nil, errors.Join(err, fmt.Errorf("CLI_STUDY_OFFICIAL_TREE_REFUSED"))
	}
	published := make([]contractexec.OfficialTarget, officialCLITrialCount)
	owned := make([]bool, officialCLITrialCount)
	closePublished := func() error {
		var closeErr error
		for index := range published {
			if !owned[index] {
				continue
			}
			owned[index] = false
			closeErr = errors.Join(closeErr, published[index].Close())
		}
		return closeErr
	}
	defer func() { returnErr = errors.Join(returnErr, closePublished()) }()
	publicationErr := runOfficialRepositoryTasks(ctx, repositories, officialCLITrialCount, func(
		taskContext context.Context,
		index int,
		repository gitobj.Repository,
	) error {
		target, publishErr := contractexec.PublishOfficialTarget(taskContext, contractexec.PublishOfficialTargetRequest{
			Store: objectStore, Residue: residue, Repository: repository,
			DisplayRef: expectedTree.Provenance().CommitOID, NodeExecutable: config.NodeExecutable,
		})
		if publishErr != nil {
			if target.Valid() {
				publishErr = errors.Join(publishErr, target.Close())
			}
			return errors.Join(publishErr, fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_%02d_REFUSED", index+1))
		}
		if !target.Valid() {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_%02d_REFUSED", index+1)
		}
		model := target.Model()
		tree := model.Input().Tree
		provenance := expectedTree.Provenance()
		if !model.Valid() || model.ContractBundleDigest() != bundle.Digest() ||
			tree.TreeIdentityDigest != expectedTree.IdentityDigest() ||
			tree.CommitOID != provenance.CommitOID || tree.TreeOID != provenance.TreeOID ||
			tree.ObjectFormat != string(repository.ObjectFormat()) {
			return errors.Join(target.Close(), fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_TREE_%02d_REFUSED", index+1))
		}
		published[index] = target
		owned[index] = true
		return nil
	})
	if publicationErr != nil {
		return nil, publicationErr
	}
	seenTargets := make(map[domain.Digest]struct{}, officialCLITrialCount)
	seenAttempts := make(map[domain.Digest]struct{}, officialCLITrialCount)
	for index, target := range published {
		if !owned[index] || !target.Valid() {
			return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_ROSTER_%02d_REFUSED", index+1)
		}
		attempt := target.TargetRecord().AttemptDigest()
		if _, duplicate := seenTargets[target.Digest()]; duplicate {
			return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_TARGET_ALIAS_REFUSED")
		}
		if _, duplicate := seenAttempts[attempt]; duplicate {
			return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_PUBLICATION_ATTEMPT_ALIAS_REFUSED")
		}
		seenTargets[target.Digest()] = struct{}{}
		seenAttempts[attempt] = struct{}{}
	}

	trials = make([]officialTrial, 0, officialCLITrialCount)
	for index := range published {
		trial, trialErr := executeOfficialTrial(
			ctx, config, authorityRoot, objectStore, residue, bundle, published[index],
		)
		if trialErr != nil {
			return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_%02d_REFUSED: %w", index+1, trialErr)
		}
		trials = append(trials, trial)
	}
	terminalResidue, err := nodeemit.ReopenResidue(ctx, objectStore, residue)
	if err != nil || !terminalResidue.Valid() || terminalResidue.HeadDigest() != residue.HeadDigest() ||
		terminalResidue.BundleDigest() != bundle.Digest() ||
		!bytes.Equal(terminalResidue.Bundle().CanonicalBytes(), bundle.CanonicalBytes()) {
		return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_TERMINAL_RESIDUE_REFUSED: %w", err)
	}
	if err := validateOfficialTrialRoster(trials, bundle.Digest(), residue.HeadDigest()); err != nil {
		return nil, err
	}
	if err := closePublished(); err != nil {
		return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_TARGET_ROSTER_CLOSE_REFUSED: %w", err)
	}
	if err := fixture.Close(); err != nil {
		return nil, fmt.Errorf("CLI_STUDY_OFFICIAL_FIXTURE_CLOSE_REFUSED: %w", err)
	}
	fixtureOpen = false
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("CLI_STUDY_CONTEXT_ENDED_AFTER_OFFICIAL_CLOSURE: %w", err)
	}
	return trials, nil
}

func executeOfficialTrial(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
	published contractexec.OfficialTarget,
) (trial officialTrial, returnErr error) {
	if ctx == nil || objectStore == nil || !residue.Valid() || !bundle.Valid() ||
		residue.BundleDigest() != bundle.Digest() || !published.Valid() {
		return officialTrial{}, fmt.Errorf("CLI_STUDY_OFFICIAL_TRIAL_INPUT_REFUSED")
	}
	// ExecuteCLI owns the first execution-adjacent full target reopen and then
	// repeats the preparation-, spawn-, and terminal-adjacent reopens. Passing
	// the sealed publication here avoids adding a caller-side reconstruction
	// that carries no distinct freshness edge.
	executed, err := contractrunner.ExecuteCLI(ctx, published)
	if err != nil || !executed.Valid() {
		return officialTrial{}, errors.Join(err, fmt.Errorf("official CLI execution failed"))
	}
	reopenedRun, err := store.OpenFinalizedRunRecord(ctx, published.TargetRecord(), published.Model())
	if err != nil || !reopenedRun.Valid() {
		return officialTrial{}, errors.Join(err, fmt.Errorf("official durable run reopen failed"))
	}
	recovered, err := contractrunner.ResumeCLIClassification(ctx, published)
	if err != nil || !recovered.Valid() {
		return officialTrial{}, errors.Join(err, fmt.Errorf("official classification recovery failed"))
	}

	target := published.Model()
	targetRecord := published.TargetRecord()
	run := reopenedRun.Model()
	classification := executed.Model()
	recovery := recovered.Model()
	tuple, projected := run.Witness().Observation().Tuple()
	recoveryEqual := recovered.Digest() == executed.Digest() && recovery.Equal(classification) &&
		bytes.Equal(recovery.CanonicalBytes(), classification.CanonicalBytes())
	if !target.Valid() || target.Digest() != published.Digest() ||
		target.ContractBundleDigest() != bundle.Digest() || !targetRecord.Valid() ||
		targetRecord.Digest() != target.Digest() || targetRecord.AttemptDigest() != target.AttemptArtifactDigest() ||
		!published.ContractBundle().Valid() || published.ContractBundle().Digest() != bundle.Digest() ||
		!bytes.Equal(published.ContractBundle().CanonicalBytes(), bundle.CanonicalBytes()) ||
		!run.Valid() || !run.MatchesTarget(target) || run.TargetDigest() != target.Digest() ||
		run.AttemptArtifactDigest() != targetRecord.AttemptDigest() ||
		!classification.Valid() || classification.TargetDigest() != target.Digest() ||
		classification.FinalizedRunDigest() != run.Digest() || string(classification.Result()) != "CONFORMS" ||
		!projected || !officialTupleHasDefaultExit(tuple) || !recoveryEqual {
		return officialTrial{}, fmt.Errorf("CLI_STUDY_OFFICIAL_GRAPH_REFUSED")
	}

	trial = officialTrial{
		attemptDigest: targetRecord.AttemptDigest(), targetDigest: target.Digest(),
		targetCanonicalSHA256: canonicalSHA256(target.CanonicalBytes()),
		bundleDigest:          bundle.Digest(), residueHeadDigest: residue.HeadDigest(),
		runDigest: run.Digest(), runCanonicalSHA256: canonicalSHA256(run.CanonicalBytes()),
		classificationDigest:          classification.Digest(),
		classificationCanonicalSHA256: canonicalSHA256(classification.CanonicalBytes()),
		result:                        string(classification.Result()), exitCode: 0, recoveryEqual: recoveryEqual,
	}
	snapshot := officialTrialSnapshot{
		attemptDigest: targetRecord.AttemptDigest(), targetDigest: target.Digest(),
		targetCanonicalSHA256: canonicalSHA256(target.CanonicalBytes()),
		bundleDigest:          bundle.Digest(), residueHeadDigest: residue.HeadDigest(),
		runDigest: run.Digest(), runCanonicalSHA256: canonicalSHA256(run.CanonicalBytes()),
		classificationDigest:          classification.Digest(),
		classificationCanonicalSHA256: canonicalSHA256(classification.CanonicalBytes()),
		result:                        string(classification.Result()), exitCode: 0, recoveryEqual: recoveryEqual,
	}
	trial.validateSnapshot = newOfficialTrialSnapshotValidator(snapshot)
	trial.revalidate = func(revalidationContext context.Context, current officialTrial) error {
		return revalidateOfficialTrial(
			revalidationContext, config, authorityRoot, objectStore, residue, bundle, current,
		)
	}
	if err := trial.validate(bundle.Digest(), residue.HeadDigest()); err != nil {
		return officialTrial{}, fmt.Errorf("CLI_STUDY_OFFICIAL_SNAPSHOT_REFUSED: %w", err)
	}
	return trial, nil
}

func runOfficialRepositoryTasks(
	ctx context.Context,
	repositories []gitobj.Repository,
	taskCount int,
	task func(context.Context, int, gitobj.Repository) error,
) error {
	if ctx == nil || task == nil || taskCount < 1 || len(repositories) != officialCLIWorkerLimit {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REPOSITORY_TASKS_REFUSED")
	}
	fingerprint := repositories[0].Fingerprint()
	format := repositories[0].ObjectFormat()
	leases := make(chan gitobj.Repository, len(repositories))
	seen := make(map[gitobj.Repository]struct{}, len(repositories))
	for _, repository := range repositories {
		if !repository.Valid() || repository.Fingerprint() != fingerprint || repository.ObjectFormat() != format {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_REPOSITORY_ROSTER_REFUSED")
		}
		if _, duplicate := seen[repository]; duplicate {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_REPOSITORY_ALIAS_REFUSED")
		}
		seen[repository] = struct{}{}
		leases <- repository
	}
	tasks := make([]cliWorkflowTask, taskCount)
	for index := range tasks {
		index := index
		tasks[index] = func(taskContext context.Context) error {
			var repository gitobj.Repository
			select {
			case repository = <-leases:
			case <-taskContext.Done():
				return taskContext.Err()
			}
			defer func() { leases <- repository }()
			return task(taskContext, index, repository)
		}
	}
	return runBoundedCLIWorkflowTasks(ctx, len(repositories), tasks)
}

func revalidateOfficialTrial(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
	expected officialTrial,
) error {
	if ctx == nil || objectStore == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_INPUT_REFUSED")
	}
	if err := expected.validate(bundle.Digest(), residue.HeadDigest()); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_INPUT_REFUSED: %w", err)
	}
	freshResidue, err := nodeemit.ReopenResidue(ctx, objectStore, residue)
	if err != nil || !freshResidue.Valid() || freshResidue.HeadDigest() != expected.residueHeadDigest ||
		freshResidue.BundleDigest() != expected.bundleDigest {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_RESIDUE_REFUSED: %w", err)
	}
	return withOfficialRevalidationRoot(authorityRoot, config.Ordinal, func(gitScratch string) error {
		return revalidateOfficialTrialInRoot(ctx, config, objectStore, freshResidue, expected, gitScratch)
	})
}

func revalidateOfficialTrialInRoot(
	ctx context.Context,
	config Config,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	expected officialTrial,
	gitScratch string,
) (returnErr error) {
	if ctx == nil || objectStore == nil || !residue.Valid() || !cleanAbsolute(gitScratch) {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_SCOPE_REFUSED")
	}
	fixture, err := reference.OpenCLIFixture(ctx, reference.CLIFixtureConfig{
		Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
	})
	if err != nil {
		return errors.Join(err, fmt.Errorf("official fixture recovery failed"))
	}
	defer func() { returnErr = errors.Join(returnErr, fixture.Close()) }()
	if !fixture.Valid() {
		return fmt.Errorf("official fixture recovery failed")
	}
	return revalidateOfficialTrialInRepository(ctx, objectStore, residue, expected, fixture.Repository())
}

func revalidateOfficialTrialInRepository(
	ctx context.Context,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	expected officialTrial,
	repository gitobj.Repository,
) (returnErr error) {
	if ctx == nil || objectStore == nil || !residue.Valid() || !repository.Valid() ||
		expected.validate(residue.BundleDigest(), residue.HeadDigest()) != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_REPOSITORY_REFUSED")
	}
	opened, err := contractexec.OpenOfficialTarget(ctx, contractexec.OpenOfficialTargetRequest{
		Store: objectStore, Residue: residue, Repository: repository,
		TargetDigest: expected.targetDigest,
	})
	if err != nil || !opened.Valid() {
		return errors.Join(err, fmt.Errorf("official target recovery failed"))
	}
	defer func() { returnErr = errors.Join(returnErr, opened.Close()) }()
	target := opened.Model()
	targetRecord := opened.TargetRecord()
	reopenedRun, err := store.OpenFinalizedRunRecord(ctx, targetRecord, target)
	if err != nil || !reopenedRun.Valid() {
		return errors.Join(err, fmt.Errorf("official durable run recovery failed"))
	}
	recovered, err := contractrunner.ResumeCLIClassification(ctx, opened)
	if err != nil || !recovered.Valid() {
		return errors.Join(err, fmt.Errorf("official classification recovery failed"))
	}
	run := reopenedRun.Model()
	classification := recovered.Model()
	tuple, projected := run.Witness().Observation().Tuple()
	if !target.Valid() || target.Digest() != expected.targetDigest ||
		targetRecord.AttemptDigest() != expected.attemptDigest ||
		target.ContractBundleDigest() != expected.bundleDigest ||
		canonicalSHA256(target.CanonicalBytes()) != expected.targetCanonicalSHA256 ||
		!run.Valid() || !run.MatchesTarget(target) || run.Digest() != expected.runDigest ||
		canonicalSHA256(run.CanonicalBytes()) != expected.runCanonicalSHA256 ||
		!classification.Valid() || classification.Digest() != expected.classificationDigest ||
		classification.TargetDigest() != expected.targetDigest ||
		classification.FinalizedRunDigest() != expected.runDigest ||
		canonicalSHA256(classification.CanonicalBytes()) != expected.classificationCanonicalSHA256 ||
		string(classification.Result()) != expected.result || expected.exitCode != 0 ||
		!expected.recoveryEqual || !projected || !officialTupleHasDefaultExit(tuple) {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_GRAPH_REFUSED")
	}
	return nil
}

func (trial officialTrial) validate(bundleDigest, residueHeadDigest domain.Digest) error {
	if !trial.staticValid(bundleDigest, residueHeadDigest) ||
		trial.validateSnapshot == nil || trial.revalidate == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_TRIAL_REFUSED")
	}
	if err := trial.validateSnapshot(trial); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_SNAPSHOT_REVALIDATION_REFUSED: %w", err)
	}
	return nil
}

func (trial officialTrial) staticValid(bundleDigest, residueHeadDigest domain.Digest) bool {
	return trial.attemptDigest.Valid() && trial.targetDigest.Valid() && trial.runDigest.Valid() &&
		trial.classificationDigest.Valid() && trial.bundleDigest == bundleDigest &&
		trial.residueHeadDigest == residueHeadDigest && trial.result == "CONFORMS" &&
		trial.exitCode == 0 && trial.recoveryEqual &&
		trial.attemptDigest != trial.targetDigest && trial.targetDigest != trial.runDigest &&
		trial.runDigest != trial.classificationDigest &&
		validCanonicalSHA256(trial.targetCanonicalSHA256) &&
		validCanonicalSHA256(trial.runCanonicalSHA256) &&
		validCanonicalSHA256(trial.classificationCanonicalSHA256)
}

func validateOfficialTrialRoster(
	trials []officialTrial,
	bundleDigest domain.Digest,
	residueHeadDigest domain.Digest,
) error {
	if len(trials) != 10 || !bundleDigest.Valid() || !residueHeadDigest.Valid() {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_COUNT_REFUSED")
	}
	seenDigests := make(map[domain.Digest]struct{}, 40)
	seenCanonical := make(map[string]struct{}, 30)
	for _, trial := range trials {
		if err := trial.validate(bundleDigest, residueHeadDigest); err != nil {
			return err
		}
		for _, digest := range []domain.Digest{
			trial.attemptDigest, trial.targetDigest, trial.runDigest, trial.classificationDigest,
		} {
			if _, duplicate := seenDigests[digest]; duplicate {
				return fmt.Errorf("CLI_STUDY_OFFICIAL_AUTHORITY_REUSE_REFUSED")
			}
			seenDigests[digest] = struct{}{}
		}
		for _, digest := range []string{
			trial.targetCanonicalSHA256,
			trial.runCanonicalSHA256,
			trial.classificationCanonicalSHA256,
		} {
			if _, duplicate := seenCanonical[digest]; duplicate {
				return fmt.Errorf("CLI_STUDY_OFFICIAL_CANONICAL_REUSE_REFUSED")
			}
			seenCanonical[digest] = struct{}{}
		}
	}
	return nil
}

func revalidateOfficialTrialRoster(
	ctx context.Context,
	trials []officialTrial,
	bundleDigest domain.Digest,
	residueHeadDigest domain.Digest,
) error {
	return revalidateOfficialTrialRosterWith(
		ctx, trials, bundleDigest, residueHeadDigest,
		func(revalidationContext context.Context, trial officialTrial) error {
			return trial.revalidate(revalidationContext, trial)
		},
	)
}

// revalidateOfficialTrialRosterShared preserves ten distinct
// target/run/classification reopens while amortizing only the outer fixture
// admission. Independent repository capabilities allow these no-spawn terminal
// graphs to close concurrently; the subject executions above remain strictly
// sequential. A second fresh fixture admission after the shared lease closes
// proves the exact repository boundary terminally before publication.
func revalidateOfficialTrialRosterShared(
	ctx context.Context,
	config Config,
	authorityRoot string,
	objectStore *store.ObjectStore,
	residue nodeemit.Residue,
	bundle emitmodel.ContractBundle,
	trials []officialTrial,
) error {
	if ctx == nil || objectStore == nil || !residue.Valid() || !bundle.Valid() ||
		residue.BundleDigest() != bundle.Digest() || !cleanAbsolute(authorityRoot) ||
		!cleanAbsolute(config.RepositoryRoot) || !cleanAbsolute(config.GitExecutable) ||
		config.Ordinal < 1 || config.Ordinal > 3 {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_SHARED_REVALIDATION_INPUT_REFUSED")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_SHARED_REVALIDATION_CONTEXT_REFUSED: %w", err)
	}
	if err := withOfficialRevalidationRoot(authorityRoot, config.Ordinal, func(gitScratch string) (returnErr error) {
		fixture, err := reference.OpenCLIFixture(ctx, reference.CLIFixtureConfig{
			Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
			RepositoryCount: officialCLIWorkerLimit,
		})
		if err != nil {
			return errors.Join(err, fmt.Errorf("official shared fixture recovery failed"))
		}
		fixtureOpen := true
		defer func() {
			if fixtureOpen {
				returnErr = errors.Join(returnErr, fixture.Close())
			}
		}()
		repositories := fixture.Repositories()
		if !fixture.Valid() || len(repositories) != officialCLIWorkerLimit {
			return fmt.Errorf("official shared fixture recovery failed")
		}
		if err := revalidateOfficialTrialRosterConcurrent(
			ctx, trials, bundle.Digest(), residue.HeadDigest(), repositories,
			func(revalidationContext context.Context, trial officialTrial, repository gitobj.Repository) error {
				return revalidateOfficialTrialInRepository(
					revalidationContext, objectStore, residue, trial, repository,
				)
			},
		); err != nil {
			return err
		}
		if err := fixture.Close(); err != nil {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_SHARED_FIXTURE_CLOSE_REFUSED: %w", err)
		}
		fixtureOpen = false
		return nil
	}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_SHARED_REVALIDATION_TERMINAL_CONTEXT_REFUSED: %w", err)
	}
	return withOfficialRevalidationRoot(authorityRoot, config.Ordinal, func(gitScratch string) (returnErr error) {
		fixture, err := reference.OpenCLIFixture(ctx, reference.CLIFixtureConfig{
			Root: config.RepositoryRoot, GitExecutable: config.GitExecutable, ScratchRoot: gitScratch,
		})
		if err != nil {
			return errors.Join(err, fmt.Errorf("official terminal fixture recovery failed"))
		}
		defer func() { returnErr = errors.Join(returnErr, fixture.Close()) }()
		if !fixture.Valid() || !fixture.Repository().Valid() {
			return fmt.Errorf("official terminal fixture recovery failed")
		}
		if _, err := fixture.Ref(reference.ArgvFirst); err != nil {
			return fmt.Errorf("official terminal fixture ref recovery failed: %w", err)
		}
		return ctx.Err()
	})
}

func revalidateOfficialTrialRosterConcurrent(
	ctx context.Context,
	trials []officialTrial,
	bundleDigest domain.Digest,
	residueHeadDigest domain.Digest,
	repositories []gitobj.Repository,
	revalidate func(context.Context, officialTrial, gitobj.Repository) error,
) error {
	if ctx == nil || revalidate == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_CONCURRENT_REVALIDATOR_REFUSED")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_CONCURRENT_PRE_CONTEXT_REFUSED: %w", err)
	}
	if err := validateOfficialTrialRoster(trials, bundleDigest, residueHeadDigest); err != nil {
		return err
	}
	if err := runOfficialRepositoryTasks(ctx, repositories, len(trials), func(
		taskContext context.Context,
		index int,
		repository gitobj.Repository,
	) error {
		if err := revalidate(taskContext, trials[index], repository); err != nil {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_CONCURRENT_REVALIDATION_%02d_REFUSED: %w", index+1, err)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_CONCURRENT_TERMINAL_CONTEXT_REFUSED: %w", err)
	}
	if err := validateOfficialTrialRoster(trials, bundleDigest, residueHeadDigest); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_CONCURRENT_TERMINAL_SNAPSHOT_REFUSED: %w", err)
	}
	return nil
}

func revalidateOfficialTrialRosterWith(
	ctx context.Context,
	trials []officialTrial,
	bundleDigest domain.Digest,
	residueHeadDigest domain.Digest,
	revalidate func(context.Context, officialTrial) error,
) error {
	if ctx == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_CONTEXT_REFUSED")
	}
	if revalidate == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATOR_REFUSED")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_PRE_CONTEXT_REFUSED: %w", err)
	}
	if err := validateOfficialTrialRoster(trials, bundleDigest, residueHeadDigest); err != nil {
		return err
	}
	for index, trial := range trials {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_CONTEXT_%02d_REFUSED: %w", index+1, err)
		}
		if err := revalidate(ctx, trial); err != nil {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_%02d_REFUSED: %w", index+1, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_POST_%02d_CONTEXT_REFUSED: %w", index+1, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_TERMINAL_CONTEXT_REFUSED: %w", err)
	}
	if err := validateOfficialTrialRoster(trials, bundleDigest, residueHeadDigest); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_TERMINAL_SNAPSHOT_REFUSED: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_LIVE_REVALIDATION_CLOSED_CONTEXT_REFUSED: %w", err)
	}
	return nil
}

func withOfficialRevalidationRoot(
	authorityRoot string,
	ordinal int,
	body func(string) error,
) (returnErr error) {
	if body == nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_BODY_REFUSED")
	}
	root, err := privateOfficialRevalidationDirectory(authorityRoot, ordinal)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, removeOfficialRevalidationRoot(authorityRoot, root))
	}()
	return body(root)
}

func privateOfficialRevalidationDirectory(authorityRoot string, ordinal int) (path string, returnErr error) {
	if !cleanAbsolute(authorityRoot) || ordinal < 1 || ordinal > 3 {
		return "", fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_ROOT_REFUSED")
	}
	canonical, err := filepath.EvalSymlinks(authorityRoot)
	if err != nil || canonical != authorityRoot {
		return "", errors.Join(err, fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_ROOT_REFUSED"))
	}
	path, err = os.MkdirTemp(canonical, fmt.Sprintf("official-revalidation-%d-", ordinal))
	if err != nil {
		return "", err
	}
	cleanupPath := path
	keep := false
	defer func() {
		if !keep {
			returnErr = errors.Join(returnErr, removeOfficialRevalidationRoot(authorityRoot, cleanupPath))
		}
	}()
	if err := os.Chmod(path, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return "", errors.Join(err, fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_ROOT_REFUSED"))
	}
	keep = true
	return path, nil
}

func removeOfficialRevalidationRoot(authorityRoot, path string) error {
	if !cleanAbsolute(authorityRoot) || !cleanAbsolute(path) || filepath.Dir(path) != authorityRoot ||
		!strings.HasPrefix(filepath.Base(path), "official-revalidation-") {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_CLEANUP_REFUSED")
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_CLEANUP_REFUSED: %w", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return fmt.Errorf("CLI_STUDY_OFFICIAL_REVALIDATION_CLEANUP_INCOMPLETE: %v", err)
	}
	return nil
}

func officialTupleHasDefaultExit(tuple emitmodel.ExactTuple) bool {
	if !tuple.Valid() {
		return false
	}
	fields := tuple.Fields()
	if len(fields) != 3 || fields[0].FieldID() != string(countercli.CLIFieldExitCode) ||
		fields[1].FieldID() != string(countercli.CLIFieldStdoutJSONMode) ||
		fields[2].FieldID() != string(countercli.CLIFieldStdoutJSONSource) {
		return false
	}
	exit, integer := fields[0].Value().PortableValue().IntegerText()
	mode, modeString := fields[1].Value().PortableValue().StringText()
	source, sourceString := fields[2].Value().PortableValue().StringText()
	return integer && exit == "0" && modeString && sourceString && mode == "argv" && source == "argv"
}

func canonicalSHA256(exact []byte) string {
	digest := sha256.Sum256(exact)
	return hex.EncodeToString(digest[:])
}

func validCanonicalSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}
