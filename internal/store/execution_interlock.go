package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	interlockFilename         = "execution-interlock.json"
	startClaimDirectory       = "start-claims"
	clearReceiptDirectory     = "clear-receipts"
	finalizedReleaseDirectory = "finalized-run-releases"
	interlockKind             = "ExecutionInterlockState"
	interlockVersionV1        = "execution-interlock/v1"
	interlockStateClear       = "CLEAR"
	interlockStateHeld        = "HELD"
	startClaimVersionV1       = "start-claim/v1"
	clearReceiptKind          = "ExecutionInterlockClearReceipt"
	clearReceiptVersionV1     = "execution-interlock-clear-receipt/v1"
	finalizedReleaseKind      = "FinalizedRunRelease"
	finalizedReleaseV1        = "finalized-run-release/v1"
	codeInterlockRefused      = "EXECUTION_INTERLOCK_REFUSED"
	codeInterlockHeld         = "EXECUTION_INTERLOCK_HELD"
	codeInterlockAmbiguous    = "EXECUTION_INTERLOCK_AMBIGUOUS"
	codeStartClaimConsumed    = "TARGET_START_ALREADY_CLAIMED"
)

const (
	faultBeforeInterlockReplace     contractFaultPhase = "before-interlock-replace"
	faultAfterInterlockRename       contractFaultPhase = "after-interlock-rename"
	faultAfterInterlockSync         contractFaultPhase = "after-interlock-sync"
	faultBeforeClaimLink            contractFaultPhase = "before-start-claim-link"
	faultAfterClaimLink             contractFaultPhase = "after-start-claim-link"
	faultAfterClaimSync             contractFaultPhase = "after-start-claim-sync"
	faultBeforeClearReceipt         contractFaultPhase = "before-clear-receipt-create"
	faultAfterClearReceiptTemp      contractFaultPhase = "after-clear-receipt-temporary-sync"
	faultBeforeClearReceiptLink     contractFaultPhase = "before-clear-receipt-link"
	faultAfterClearReceiptLink      contractFaultPhase = "after-clear-receipt-link"
	faultAfterClearReceiptSync      contractFaultPhase = "after-clear-receipt-directory-sync"
	faultBeforeClearReceiptOpen     contractFaultPhase = "before-clear-receipt-reopen"
	faultBeforeFinalizedReleaseTemp contractFaultPhase = "before-finalized-release-temporary-create"
	faultAfterFinalizedReleaseTemp  contractFaultPhase = "after-finalized-release-temporary-sync"
	faultBeforeFinalizedReleaseLink contractFaultPhase = "before-finalized-release-link"
	faultAfterFinalizedReleaseLink  contractFaultPhase = "after-finalized-release-link"
	faultAfterFinalizedReleaseSync  contractFaultPhase = "after-finalized-release-directory-sync"
	faultBeforeFinalizedReleaseOpen contractFaultPhase = "before-finalized-release-reopen"
)

type executionInterlockState struct {
	state         string
	bootDigest    domain.Digest
	targetDigest  domain.Digest
	attemptDigest domain.Digest
	generation    domain.Digest
	revision      int64
	canonical     []byte
	digest        domain.Digest
}

type interlockLease struct {
	storeInstance *objectStoreInstance
	state         executionInterlockState
	seal          *interlockLeaseSeal
}

type interlockLeaseSeal struct{ marker byte }

type startClaimRecord struct {
	storeInstance *objectStoreInstance
	targetDigest  domain.Digest
	attemptDigest domain.Digest
	bootDigest    domain.Digest
	generation    domain.Digest
	digest        domain.Digest
	canonical     []byte
	path          string
}

type startClaimWinner struct {
	lease interlockLease
	claim startClaimRecord
	seal  *startClaimWinnerSeal
}

type startClaimWinnerSeal struct {
	marker   byte
	mu       sync.Mutex
	consumed bool
}

type terminalClosureRecord struct {
	run      finalizedRunStorageRecord
	manifest privateManifestRecord
	seal     *terminalClosureSeal
}

type terminalClosureSeal struct{ marker byte }

type changedBootResetAuthorization struct {
	currentBoot domain.Digest
	seal        *changedBootResetSeal
}

type changedBootResetSeal struct{ marker byte }

func acquireInterlockAndStartClaim(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
	bootDigest domain.Digest,
	fault contractFault,
) (interlockLease, startClaimRecord, startClaimWinner, error) {
	if store == nil || !target.validFor(store) || !bootDigest.Valid() {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockRefused, "target or boot parent is invalid", nil)
	}
	targetBoot, err := targetBootDigest(target.record.object)
	if err != nil || targetBoot != bootDigest {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(
			codeInterlockRefused, "target and caller boot-session identities differ", err,
		)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{},
			refuse(codeInterlockRefused, "target changed before interlock acquisition", nil)
	}
	claimDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, startClaimDirectory)
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	claim, err := expectedStartClaim(store, target, bootDigest, "")
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	claimPath := claimPathFor(claimDirectory, target.record.object.Digest())
	if err := rejectPathCaseAlias(claimPath); err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(
			codeInterlockRefused, "start-claim path aliases an existing name", err,
		)
	}
	if _, err := os.Lstat(claimPath); err == nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeStartClaimConsumed, "target already has durable start intent", nil)
	} else if !errors.Is(err, os.ErrNotExist) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockAmbiguous, "start-claim path cannot be classified", err)
	}
	statePath := filepath.Join(store.contractOps, interlockFilename)
	current, present, err := readExecutionInterlockState(statePath)
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	if present && current.state == interlockStateHeld {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockHeld, "another target or ambiguous owner holds the store-wide interlock", nil)
	}
	if present && current.state == interlockStateClear && !clearReceiptExactLocked(store, current) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockHeld, "clear state lacks an exact durable admission receipt", nil)
	}
	previousDigest := domain.Digest("")
	revision := int64(1)
	if present {
		previousDigest = current.digest
		revision = current.revision + 1
	}
	generation := deriveInterlockGeneration(previousDigest, target.record.object.Digest(), target.record.relation.attemptDigest, bootDigest, revision)
	held, err := newExecutionInterlockState(
		interlockStateHeld, bootDigest, target.record.object.Digest(), target.record.relation.attemptDigest, generation, revision,
	)
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	if err := replaceExecutionInterlock(statePath, held, present, fault); err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	lease := interlockLease{storeInstance: store.instance, state: held, seal: &interlockLeaseSeal{marker: 1}}
	claim, err = expectedStartClaim(store, target, bootDigest, generation)
	if err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, err
	}
	claim.path = claimPath
	if err := injectContractFault(fault, faultBeforeClaimLink); err != nil {
		reconcileErr := reconcileKnownNoClaimLocked(store, statePath, held, fault)
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, errors.Join(
			mutationFailure(codeInterlockRefused, contractKnownNoEffect, faultBeforeClaimLink, err), reconcileErr,
		)
	}
	claimFault := func(phase contractFaultPhase) error {
		switch phase {
		case faultAfterLink:
			return injectContractFault(fault, faultAfterClaimLink)
		case faultAfterParentSync:
			return injectContractFault(fault, faultAfterClaimSync)
		default:
			return nil
		}
	}
	if _, err := createExactPrivateFile(claimDirectory, filepath.Base(claimPath), claim.canonical, claimFault); err != nil {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockAmbiguous, "start-claim create was not exact", err)
	}
	reopened, err := readExactPrivateFile(claimPath, int64(len(claim.canonical)))
	if err != nil || !bytes.Equal(reopened, claim.canonical) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockAmbiguous, "start claim did not reopen exactly", err)
	}
	if !claim.validForLocked(store) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockAmbiguous, "start claim parent or bytes changed", nil)
	}
	if current, present, err := readExecutionInterlockState(statePath); err != nil || !present || !sameInterlockState(current, held) {
		return interlockLease{}, startClaimRecord{}, startClaimWinner{}, refuse(codeInterlockAmbiguous, "held interlock changed before winner issuance", err)
	}
	winner := startClaimWinner{lease: lease, claim: claim, seal: &startClaimWinnerSeal{marker: 1}}
	return lease, claim, winner, nil
}

func openStartClaim(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
) (startClaimRecord, error) {
	if store == nil || !target.validFor(store) {
		return startClaimRecord{}, refuse(codeInterlockRefused, "target parent is invalid or foreign", nil)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return startClaimRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return startClaimRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return startClaimRecord{}, refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) {
		return startClaimRecord{}, refuse(codeInterlockRefused, "target changed before start-claim reopen", nil)
	}
	claimDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, startClaimDirectory)
	if err != nil {
		return startClaimRecord{}, err
	}
	path := claimPathFor(claimDirectory, target.record.object.Digest())
	body, err := readExactPrivateFile(path, -1)
	if err != nil {
		return startClaimRecord{}, refuse(codeInterlockRefused, "start claim is absent or invalid", err)
	}
	claim, err := parseStartClaim(store, body, path)
	if err != nil || !startClaimJoinsTarget(target, claim) {
		return startClaimRecord{}, refuse(codeInterlockRefused, "start claim does not join the target", err)
	}
	if effect, convergeErr := createExactPrivateFile(claimDirectory, filepath.Base(path), body, nil); convergeErr != nil || effect != contractExactConverged {
		return startClaimRecord{}, refuse(codeInterlockAmbiguous, "visible start claim did not converge durably", convergeErr)
	}
	if !claim.validForLocked(store) {
		return startClaimRecord{}, refuse(codeInterlockRefused, "start claim changed after convergence", nil)
	}
	return claim, nil
}

func releaseInterlockAfterFinalizedRun(
	ctx context.Context,
	store *ObjectStore,
	closure terminalClosureRecord,
	claim startClaimRecord,
	fault contractFault,
) error {
	run := closure.run
	manifest := closure.manifest
	if store == nil || !run.validFor(store) || !claim.validFor(store) ||
		closure.seal == nil || closure.seal.marker != 1 || !manifest.validFor(store) ||
		run.record.relation.targetDigest != claim.targetDigest ||
		run.record.relation.attemptDigest != claim.attemptDigest ||
		run.record.relation.startClaimDigest != claim.digest || manifest.targetDigest != claim.targetDigest ||
		manifest.attemptDigest != claim.attemptDigest || manifest.startClaimDigest != claim.digest {
		return refuse(codeInterlockRefused, "release requires an exact durable run and matching claim", nil)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return err
	}
	if !run.validForLocked(store) || !claim.validForLocked(store) || !manifest.validFor(store) ||
		validatePrivateManifestLocked(store, manifest) != nil ||
		validatePrivateManifestRoster(run.record.object, manifest) != nil {
		return refuse(codeInterlockRefused, "terminal closure did not reopen exact run and private evidence", nil)
	}
	statePath := filepath.Join(store.contractOps, interlockFilename)
	current, present, err := readExecutionInterlockState(statePath)
	if err != nil || !present {
		return refuse(codeInterlockRefused, "interlock is absent or unreadable during finalized release", err)
	}
	switch current.state {
	case interlockStateHeld:
		if current.targetDigest != claim.targetDigest || current.attemptDigest != claim.attemptDigest ||
			current.bootDigest != claim.bootDigest || current.generation != claim.generation {
			return refuse(codeInterlockRefused, "held interlock does not match the finalized owner", nil)
		}
		clear, clearErr := newExecutionInterlockState(
			interlockStateClear, current.bootDigest, "", "", deriveInterlockGeneration(
				current.digest, current.targetDigest, current.attemptDigest, current.bootDigest, current.revision+1,
			), current.revision+1,
		)
		if clearErr != nil {
			return clearErr
		}
		if err := replaceExecutionInterlock(statePath, clear, true, fault); err != nil {
			return err
		}
		if err := persistFinalizedRunReleaseLocked(store, clear, current, claim, run.record.object.Digest(), fault); err != nil {
			return err
		}
		return persistClearReceiptLocked(store, clear, current, "FINALIZED_RUN", run.record.object.Digest(), fault)

	case interlockStateClear:
		previous, previousErr := finalizedRunClearPredecessor(current, claim)
		if previousErr != nil {
			return previousErr
		}
		if err := persistFinalizedRunReleaseLocked(store, current, previous, claim, run.record.object.Digest(), fault); err != nil {
			return err
		}
		return persistClearReceiptLocked(store, current, previous, "FINALIZED_RUN", run.record.object.Digest(), fault)

	default:
		return refuse(codeInterlockRefused, "interlock state is not releasable", nil)
	}
}

func finalizedRunClearPredecessor(
	clear executionInterlockState,
	claim startClaimRecord,
) (executionInterlockState, error) {
	if !clear.valid() || clear.state != interlockStateClear || clear.revision < 2 ||
		clear.bootDigest != claim.bootDigest {
		return executionInterlockState{}, refuse(
			codeInterlockRefused, "clear interlock cannot be the finalized owner's immediate successor", nil,
		)
	}
	previous, err := newExecutionInterlockState(
		interlockStateHeld, claim.bootDigest, claim.targetDigest, claim.attemptDigest,
		claim.generation, clear.revision-1,
	)
	if err != nil {
		return executionInterlockState{}, err
	}
	expected, err := newExecutionInterlockState(
		interlockStateClear, claim.bootDigest, "", "", deriveInterlockGeneration(
			previous.digest, claim.targetDigest, claim.attemptDigest, claim.bootDigest, clear.revision,
		), clear.revision,
	)
	if err != nil || !sameInterlockState(expected, clear) {
		return executionInterlockState{}, refuse(
			codeInterlockRefused, "clear interlock differs from the exact matching HELD successor", err,
		)
	}
	return previous, nil
}

func validateFinalizedRunRelease(
	ctx context.Context,
	store *ObjectStore,
	claim startClaimRecord,
	runDigest domain.Digest,
) error {
	if store == nil || !claim.validFor(store) || !runDigest.Valid() {
		return refuse(codeInterlockRefused, "exact start claim and finalized run are required", nil)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if !claim.validForLocked(store) {
		return refuse(codeInterlockRefused, "start claim changed before release validation", nil)
	}
	releaseDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, finalizedReleaseDirectory)
	if err != nil {
		return err
	}
	runHex, err := strictDigestHex(runDigest)
	if err != nil {
		return err
	}
	releasePath := filepath.Join(releaseDirectory, runHex)
	if err := rejectPathCaseAlias(releasePath); err != nil {
		return refuse(codeInterlockRefused, "finalized-run release path changed exact spelling", err)
	}
	releaseBody, err := readExactPrivateFile(releasePath, -1)
	if err != nil {
		return refuse(codeInterlockRefused, "finalized-run release link is absent or invalid", err)
	}
	clearDigest, receiptDigest, err := parseFinalizedRunRelease(releaseBody, claim, runDigest)
	if err != nil {
		return refuse(codeInterlockRefused, "finalized-run release link differs", err)
	}
	receiptDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, clearReceiptDirectory)
	if err != nil {
		return err
	}
	clearHex, err := strictDigestHex(clearDigest)
	if err != nil {
		return err
	}
	receiptPath := filepath.Join(receiptDirectory, clearHex)
	if err := rejectPathCaseAlias(receiptPath); err != nil {
		return refuse(codeInterlockRefused, "finalized-run receipt path changed exact spelling", err)
	}
	receiptBody, err := readExactPrivateFile(receiptPath, -1)
	if err != nil || !finalizedRunReceiptExact(receiptBody, claim, runDigest, clearDigest, receiptDigest) {
		return refuse(codeInterlockRefused, "exact finalized-run release receipt is absent or mismatched", err)
	}
	if effect, convergeErr := createExactPrivateFile(releaseDirectory, runHex, releaseBody, nil); convergeErr != nil || effect != contractExactConverged {
		return refuse(codeInterlockAmbiguous, "finalized-run release link did not converge durably", convergeErr)
	}
	if effect, convergeErr := createExactPrivateFile(receiptDirectory, clearHex, receiptBody, nil); convergeErr != nil || effect != contractExactConverged {
		return refuse(codeInterlockAmbiguous, "finalized-run release receipt did not converge durably", convergeErr)
	}
	return store.assertReady()
}

func persistFinalizedRunReleaseLocked(
	store *ObjectStore,
	clear executionInterlockState,
	previous executionInterlockState,
	claim startClaimRecord,
	runDigest domain.Digest,
	fault contractFault,
) error {
	if store == nil || !clear.valid() || !previous.valid() || !claim.digest.Valid() || !runDigest.Valid() ||
		clear.state != interlockStateClear || previous.state != interlockStateHeld ||
		previous.targetDigest != claim.targetDigest || previous.attemptDigest != claim.attemptDigest ||
		previous.bootDigest != claim.bootDigest || previous.generation != claim.generation {
		return refuse(codeInterlockRefused, "finalized-run release-link inputs are invalid", nil)
	}
	receiptBody, err := clearReceiptBytes(clear, previous, "FINALIZED_RUN", runDigest)
	if err != nil {
		return err
	}
	receiptDigestValue, err := canon.DigestBytes(clearReceiptKind, receiptBody)
	if err != nil {
		return err
	}
	receiptDigest, err := domain.ParseDigest(receiptDigestValue.String())
	if err != nil {
		return err
	}
	releaseBody, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version":       domain.SchemaVersion,
		"kind":                 finalizedReleaseKind,
		"release_version":      finalizedReleaseV1,
		"target_digest":        claim.targetDigest.String(),
		"attempt_digest":       claim.attemptDigest.String(),
		"boot_digest":          claim.bootDigest.String(),
		"interlock_generation": claim.generation.String(),
		"start_claim_digest":   claim.digest.String(),
		"finalized_run_digest": runDigest.String(),
		"clear_state_digest":   clear.digest.String(),
		"clear_receipt_digest": receiptDigest.String(),
	})
	if err != nil {
		return err
	}
	directory, err := store.ensureContractDirectoryLocked(store.contractOps, finalizedReleaseDirectory)
	if err != nil {
		return err
	}
	hex, err := strictDigestHex(runDigest)
	if err != nil {
		return err
	}
	releaseFault := func(phase contractFaultPhase) error {
		switch phase {
		case faultBeforeTemporary:
			return injectContractFault(fault, faultBeforeFinalizedReleaseTemp)
		case faultAfterTempSync:
			return injectContractFault(fault, faultAfterFinalizedReleaseTemp)
		case faultBeforeLink:
			return injectContractFault(fault, faultBeforeFinalizedReleaseLink)
		case faultAfterLink:
			return injectContractFault(fault, faultAfterFinalizedReleaseLink)
		case faultAfterParentSync:
			return injectContractFault(fault, faultAfterFinalizedReleaseSync)
		case faultBeforeReopen:
			return injectContractFault(fault, faultBeforeFinalizedReleaseOpen)
		default:
			return nil
		}
	}
	if _, err := createExactPrivateFile(directory, hex, releaseBody, releaseFault); err != nil {
		return refuse(codeInterlockAmbiguous, "finalized-run release link did not persist exactly", err)
	}
	reopened, err := readExactPrivateFile(filepath.Join(directory, hex), int64(len(releaseBody)))
	if err != nil || !bytes.Equal(reopened, releaseBody) {
		return refuse(codeInterlockAmbiguous, "finalized-run release link did not reopen exactly", err)
	}
	return nil
}

func parseFinalizedRunRelease(
	body []byte,
	claim startClaimRecord,
	runDigest domain.Digest,
) (domain.Digest, domain.Digest, error) {
	value, err := canon.Parse(body)
	if err != nil || !contractRoster(value,
		"schema_version", "kind", "release_version", "target_digest", "attempt_digest",
		"boot_digest", "interlock_generation", "start_claim_digest", "finalized_run_digest",
		"clear_state_digest", "clear_receipt_digest",
	) {
		return "", "", errors.Join(err, errors.New("finalized-run release roster differs"))
	}
	readText := func(name string) (string, bool) {
		member, ok := value.LookupMember(name)
		if !ok {
			return "", false
		}
		result, ok := member.Text()
		return result, ok
	}
	schema, schemaOK := readText("schema_version")
	kind, kindOK := readText("kind")
	version, versionOK := readText("release_version")
	targetRaw, targetOK := readText("target_digest")
	attemptRaw, attemptOK := readText("attempt_digest")
	bootRaw, bootOK := readText("boot_digest")
	generationRaw, generationOK := readText("interlock_generation")
	claimRaw, claimOK := readText("start_claim_digest")
	runRaw, runOK := readText("finalized_run_digest")
	clearRaw, clearOK := readText("clear_state_digest")
	receiptRaw, receiptOK := readText("clear_receipt_digest")
	clearDigest, clearErr := domain.ParseDigest(clearRaw)
	receiptDigest, receiptErr := domain.ParseDigest(receiptRaw)
	if !schemaOK || !kindOK || !versionOK || !targetOK || !attemptOK || !bootOK || !generationOK ||
		!claimOK || !runOK || !clearOK || !receiptOK || clearErr != nil || receiptErr != nil ||
		schema != domain.SchemaVersion || kind != finalizedReleaseKind || version != finalizedReleaseV1 ||
		targetRaw != claim.targetDigest.String() || attemptRaw != claim.attemptDigest.String() ||
		bootRaw != claim.bootDigest.String() || generationRaw != claim.generation.String() ||
		claimRaw != claim.digest.String() || runRaw != runDigest.String() {
		return "", "", errors.New("finalized-run release parents differ")
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return "", "", errors.Join(err, errors.New("finalized-run release is not exact canonical JSON"))
	}
	return clearDigest, receiptDigest, nil
}

func finalizedRunReceiptExact(
	body []byte,
	claim startClaimRecord,
	runDigest domain.Digest,
	clearDigest domain.Digest,
	receiptDigest domain.Digest,
) bool {
	value, err := canon.Parse(body)
	if err != nil || !contractRoster(value,
		"schema_version", "kind", "receipt_version", "cause", "clear_state_digest",
		"previous_held_digest", "previous_target_digest", "previous_attempt_digest",
		"previous_boot_digest", "previous_generation", "previous_revision", "boot_digest", "terminal_run_digest",
	) {
		return false
	}
	readText := func(name string) (string, bool) {
		member, ok := value.LookupMember(name)
		if !ok {
			return "", false
		}
		result, ok := member.Text()
		return result, ok
	}
	schema, schemaOK := readText("schema_version")
	kind, kindOK := readText("kind")
	version, versionOK := readText("receipt_version")
	cause, causeOK := readText("cause")
	clearRaw, clearOK := readText("clear_state_digest")
	previousRaw, previousOK := readText("previous_held_digest")
	previousTargetRaw, targetOK := readText("previous_target_digest")
	previousAttemptRaw, attemptOK := readText("previous_attempt_digest")
	previousBootRaw, bootOK := readText("previous_boot_digest")
	previousGenerationRaw, generationOK := readText("previous_generation")
	currentBootRaw, currentBootOK := readText("boot_digest")
	runRaw, runOK := readText("terminal_run_digest")
	revisionValue, revisionPresent := value.LookupMember("previous_revision")
	previousRevision, revisionOK := revisionValue.Int64()
	if !schemaOK || !kindOK || !versionOK || !causeOK || !clearOK || !previousOK ||
		!targetOK || !attemptOK || !bootOK || !generationOK || !currentBootOK || !runOK ||
		!revisionPresent || !revisionOK || previousRevision < 1 ||
		schema != domain.SchemaVersion || kind != clearReceiptKind || version != clearReceiptVersionV1 ||
		cause != "FINALIZED_RUN" || clearRaw != clearDigest.String() ||
		previousTargetRaw != claim.targetDigest.String() || previousAttemptRaw != claim.attemptDigest.String() ||
		previousBootRaw != claim.bootDigest.String() || previousGenerationRaw != claim.generation.String() ||
		currentBootRaw != claim.bootDigest.String() || runRaw != runDigest.String() {
		return false
	}
	previous, err := newExecutionInterlockState(
		interlockStateHeld, claim.bootDigest, claim.targetDigest, claim.attemptDigest,
		claim.generation, previousRevision,
	)
	if err != nil || previousRaw != previous.digest.String() {
		return false
	}
	clear, err := newExecutionInterlockState(
		interlockStateClear, claim.bootDigest, "", "", deriveInterlockGeneration(
			previous.digest, claim.targetDigest, claim.attemptDigest, claim.bootDigest, previousRevision+1,
		), previousRevision+1,
	)
	if err != nil || clear.digest != clearDigest {
		return false
	}
	expected, err := clearReceiptBytes(clear, previous, "FINALIZED_RUN", runDigest)
	if err != nil || !bytes.Equal(expected, body) {
		return false
	}
	digestValue, err := canon.DigestBytes(clearReceiptKind, body)
	return err == nil && digestValue.String() == receiptDigest.String()
}

func resetInterlockAfterBootChange(
	ctx context.Context,
	store *ObjectStore,
	authorization changedBootResetAuthorization,
	fault contractFault,
) error {
	if store == nil || authorization.seal == nil || authorization.seal.marker != 1 || !authorization.currentBoot.Valid() {
		return refuse(codeInterlockRefused, "changed-boot reset lacks test-authorized witness", nil)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return err
	}
	statePath := filepath.Join(store.contractOps, interlockFilename)
	current, present, err := readExecutionInterlockState(statePath)
	if err != nil || !present || current.state != interlockStateHeld {
		return refuse(codeInterlockRefused, "changed-boot reset requires an exact held state", err)
	}
	if current.bootDigest == authorization.currentBoot {
		return refuse(codeInterlockRefused, "same-boot reset is always refused", nil)
	}
	clear, err := newExecutionInterlockState(
		interlockStateClear, authorization.currentBoot, "", "", deriveInterlockGeneration(
			current.digest, current.targetDigest, current.attemptDigest, authorization.currentBoot, current.revision+1,
		), current.revision+1,
	)
	if err != nil {
		return err
	}
	if err := replaceExecutionInterlock(statePath, clear, true, fault); err != nil {
		return err
	}
	return persistClearReceiptLocked(store, clear, current, "CHANGED_BOOT", "", fault)
}

func (lease interlockLease) validFor(store *ObjectStore) bool {
	return store != nil && lease.storeInstance == store.instance && lease.seal != nil && lease.seal.marker == 1 &&
		lease.state.state == interlockStateHeld && lease.state.valid()
}

func (record startClaimRecord) validFor(store *ObjectStore) bool {
	if store == nil || store.instance == nil || record.storeInstance != store.instance || !record.targetDigest.Valid() ||
		!record.attemptDigest.Valid() || !record.bootDigest.Valid() || !record.generation.Valid() ||
		!record.digest.Valid() || len(record.canonical) == 0 {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return record.validForLocked(store)
}

func (record startClaimRecord) validForLocked(store *ObjectStore) bool {
	if store == nil || store.instance == nil || record.storeInstance != store.instance || !record.targetDigest.Valid() ||
		!record.attemptDigest.Valid() || !record.bootDigest.Valid() || !record.generation.Valid() ||
		!record.digest.Valid() || len(record.canonical) == 0 || store.assertReady() != nil {
		return false
	}
	claimDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, startClaimDirectory)
	if err != nil || claimDirectory != filepath.Dir(record.path) ||
		record.path != claimPathFor(claimDirectory, record.targetDigest) {
		return false
	}
	body, err := readExactPrivateFile(record.path, int64(len(record.canonical)))
	return err == nil && bytes.Equal(body, record.canonical) && store.assertReady() == nil
}

func startClaimJoinsTarget(target targetStorageRecord, claim startClaimRecord) bool {
	if !target.record.object.Valid() || claim.targetDigest != target.record.object.Digest() ||
		claim.attemptDigest != target.record.relation.attemptDigest {
		return false
	}
	targetBoot, err := targetBootDigest(target.record.object)
	return err == nil && targetBoot == claim.bootDigest
}

func (winner startClaimWinner) validFor(store *ObjectStore) bool {
	if winner.seal == nil {
		return false
	}
	winner.seal.mu.Lock()
	defer winner.seal.mu.Unlock()
	return winner.validForSealLocked(store)
}

func (winner startClaimWinner) validForSealLocked(store *ObjectStore) bool {
	if store == nil || store.instance == nil || winner.seal.marker != 1 || winner.seal.consumed ||
		!winner.lease.validFor(store) ||
		winner.claim.generation != winner.lease.state.generation {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return winner.validForStoreLocked(store)
}

func (winner startClaimWinner) validForStoreLocked(store *ObjectStore) bool {
	if store == nil || store.instance == nil || winner.seal == nil || winner.seal.marker != 1 || winner.seal.consumed ||
		!winner.lease.validFor(store) || winner.claim.generation != winner.lease.state.generation ||
		winner.claim.targetDigest != winner.lease.state.targetDigest ||
		winner.claim.attemptDigest != winner.lease.state.attemptDigest ||
		winner.claim.bootDigest != winner.lease.state.bootDigest ||
		store.assertReady() != nil || !winner.claim.validForLocked(store) {
		return false
	}
	current, present, err := readExecutionInterlockState(filepath.Join(store.contractOps, interlockFilename))
	return err == nil && present && sameInterlockState(current, winner.lease.state)
}

func consumeStartClaimWinner(ctx context.Context, store *ObjectStore, winner *startClaimWinner) error {
	if winner == nil || winner.seal == nil {
		return refuse(codeInterlockRefused, "winner is absent", nil)
	}
	winner.seal.mu.Lock()
	defer winner.seal.mu.Unlock()
	if !winner.validForSealLocked(store) {
		return refuse(codeInterlockRefused, "winner is stale, foreign, or already consumed", nil)
	}
	if err := storeContextRefusal(ctx, codeInterlockRefused); err != nil {
		return err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return refuse(codeInterlockRefused, "store-wide contract lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil || !winner.validForStoreLocked(store) {
		return refuse(codeInterlockRefused, "winner lost its exact held generation", err)
	}
	winner.seal.consumed = true
	return nil
}

func expectedStartClaim(
	store *ObjectStore,
	target targetStorageRecord,
	bootDigest domain.Digest,
	generation domain.Digest,
) (startClaimRecord, error) {
	if !target.validForLocked(store) || !bootDigest.Valid() {
		return startClaimRecord{}, refuse(codeInterlockRefused, "start claim parents are invalid", nil)
	}
	if !generation.Valid() {
		return startClaimRecord{
			storeInstance: store.instance, targetDigest: target.record.object.Digest(),
			attemptDigest: target.record.relation.attemptDigest, bootDigest: bootDigest,
		}, nil
	}
	body, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": contractClaimKind,
		"claim_version": startClaimVersionV1, "intent": "SUBJECT_SPAWN_ATTEMPT",
		"target_digest":  target.record.object.Digest().String(),
		"attempt_digest": target.record.relation.attemptDigest.String(),
		"boot_digest":    bootDigest.String(), "interlock_generation": generation.String(),
	})
	if err != nil {
		return startClaimRecord{}, err
	}
	digestValue, err := canon.DigestBytes(contractClaimKind, body)
	if err != nil {
		return startClaimRecord{}, err
	}
	digest, err := domain.ParseDigest(digestValue.String())
	if err != nil {
		return startClaimRecord{}, err
	}
	return startClaimRecord{
		storeInstance: store.instance, targetDigest: target.record.object.Digest(),
		attemptDigest: target.record.relation.attemptDigest, bootDigest: bootDigest,
		generation: generation, digest: digest, canonical: body,
	}, nil
}

func parseStartClaim(store *ObjectStore, body []byte, path string) (startClaimRecord, error) {
	if store == nil || store.instance == nil {
		return startClaimRecord{}, errors.New("start claim store is invalid")
	}
	value, err := canon.Parse(body)
	if err != nil {
		return startClaimRecord{}, err
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return startClaimRecord{}, errors.Join(err, errors.New("start claim is not canonical"))
	}
	readDigest := func(name string) (domain.Digest, error) {
		member, ok := value.LookupMember(name)
		if !ok {
			return "", fmt.Errorf("missing %s", name)
		}
		raw, ok := member.Text()
		if !ok {
			return "", fmt.Errorf("%s is not text", name)
		}
		return domain.ParseDigest(raw)
	}
	target, err := readDigest("target_digest")
	if err != nil {
		return startClaimRecord{}, err
	}
	attempt, err := readDigest("attempt_digest")
	if err != nil {
		return startClaimRecord{}, err
	}
	boot, err := readDigest("boot_digest")
	if err != nil {
		return startClaimRecord{}, err
	}
	generation, err := readDigest("interlock_generation")
	if err != nil {
		return startClaimRecord{}, err
	}
	expected, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": contractClaimKind,
		"claim_version": startClaimVersionV1, "intent": "SUBJECT_SPAWN_ATTEMPT",
		"target_digest": target.String(), "attempt_digest": attempt.String(),
		"boot_digest": boot.String(), "interlock_generation": generation.String(),
	})
	if err != nil || !bytes.Equal(expected, body) {
		return startClaimRecord{}, errors.Join(err, errors.New("start claim envelope or roster differs"))
	}
	digestValue, err := canon.DigestBytes(contractClaimKind, body)
	if err != nil {
		return startClaimRecord{}, err
	}
	digest, _ := domain.ParseDigest(digestValue.String())
	return startClaimRecord{
		storeInstance: store.instance, targetDigest: target, attemptDigest: attempt,
		bootDigest: boot, generation: generation, digest: digest, canonical: append([]byte(nil), body...), path: path,
	}, nil
}

func newExecutionInterlockState(
	state string,
	boot, target, attempt, generation domain.Digest,
	revision int64,
) (executionInterlockState, error) {
	if (state != interlockStateClear && state != interlockStateHeld) || !boot.Valid() || !generation.Valid() ||
		revision < 1 || (state == interlockStateHeld && (!target.Valid() || !attempt.Valid())) ||
		(state == interlockStateClear && (target.Valid() || attempt.Valid())) {
		return executionInterlockState{}, refuse(codeInterlockRefused, "interlock state fields are invalid", nil)
	}
	body, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": interlockKind,
		"interlock_version": interlockVersionV1, "state": state,
		"boot_digest": boot.String(), "target_digest": target.String(),
		"attempt_digest": attempt.String(), "generation": generation.String(), "revision": revision,
	})
	if err != nil {
		return executionInterlockState{}, err
	}
	digestValue, err := canon.DigestBytes(interlockKind, body)
	if err != nil {
		return executionInterlockState{}, err
	}
	digest, _ := domain.ParseDigest(digestValue.String())
	return executionInterlockState{
		state: state, bootDigest: boot, targetDigest: target, attemptDigest: attempt,
		generation: generation, revision: revision, canonical: body, digest: digest,
	}, nil
}

func (state executionInterlockState) valid() bool {
	rebuilt, err := newExecutionInterlockState(
		state.state, state.bootDigest, state.targetDigest, state.attemptDigest, state.generation, state.revision,
	)
	return err == nil && rebuilt.digest == state.digest && bytes.Equal(rebuilt.canonical, state.canonical)
}

func readExecutionInterlockState(path string) (executionInterlockState, bool, error) {
	body, err := readExactPrivateFile(path, -1)
	if errors.Is(err, os.ErrNotExist) {
		return executionInterlockState{}, false, nil
	}
	if err != nil {
		return executionInterlockState{}, false, refuse(codeInterlockAmbiguous, "interlock path is invalid", err)
	}
	value, err := canon.Parse(body)
	if err != nil {
		return executionInterlockState{}, false, refuse(codeInterlockAmbiguous, "interlock body is invalid", err)
	}
	text := func(name string) (string, error) {
		member, ok := value.LookupMember(name)
		if !ok {
			return "", fmt.Errorf("missing %s", name)
		}
		result, ok := member.Text()
		if !ok {
			return "", fmt.Errorf("%s is not text", name)
		}
		return result, nil
	}
	stateText, err := text("state")
	if err != nil {
		return executionInterlockState{}, false, err
	}
	bootRaw, _ := text("boot_digest")
	targetRaw, _ := text("target_digest")
	attemptRaw, _ := text("attempt_digest")
	generationRaw, _ := text("generation")
	boot, bootErr := domain.ParseDigest(bootRaw)
	generation, generationErr := domain.ParseDigest(generationRaw)
	target, targetErr := optionalInterlockDigest(targetRaw)
	attempt, attemptErr := optionalInterlockDigest(attemptRaw)
	revisionValue, ok := value.LookupMember("revision")
	revision, revisionOK := revisionValue.Int64()
	if bootErr != nil || generationErr != nil || targetErr != nil || attemptErr != nil || !ok || !revisionOK {
		return executionInterlockState{}, false, refuse(codeInterlockAmbiguous, "interlock fields differ", errors.Join(bootErr, generationErr, targetErr, attemptErr))
	}
	rebuilt, err := newExecutionInterlockState(stateText, boot, target, attempt, generation, revision)
	if err != nil || !bytes.Equal(rebuilt.canonical, body) {
		return executionInterlockState{}, false, refuse(codeInterlockAmbiguous, "interlock does not reconstruct exactly", err)
	}
	return rebuilt, true, nil
}

func replaceExecutionInterlock(path string, state executionInterlockState, replacing bool, fault contractFault) error {
	if !state.valid() {
		return refuse(codeInterlockRefused, "replacement interlock is invalid", nil)
	}
	if err := rejectPathCaseAlias(path); err != nil {
		return mutationFailure(codeInterlockRefused, contractKnownNoEffect, faultBeforeInterlockReplace, err)
	}
	if err := injectContractFault(fault, faultBeforeInterlockReplace); err != nil {
		return mutationFailure(codeInterlockRefused, contractKnownNoEffect, faultBeforeInterlockReplace, err)
	}
	directory := filepath.Dir(path)
	temporary, err := writeSyncedTemporary(directory, ".interlock-", state.canonical)
	if err != nil {
		return mutationFailure(codeInterlockRefused, contractKnownNoEffect, faultBeforeInterlockReplace, err)
	}
	defer os.Remove(temporary)
	if replacing {
		if _, present, err := readExecutionInterlockState(path); err != nil || !present {
			return errors.Join(err, errors.New("replacement interlock is absent"))
		}
	} else if _, err := os.Lstat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		return refuse(codeInterlockAmbiguous, "initial interlock destination is not absent", err)
	}
	if err := renameHeadNoFollow(temporary, path); err != nil {
		return mutationFailure(codeInterlockRefused, contractKnownNoEffect, faultBeforeInterlockReplace, err)
	}
	if err := injectContractFault(fault, faultAfterInterlockRename); err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultAfterInterlockRename, err)
	}
	if err := syncDirectory(directory); err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultAfterInterlockSync, err)
	}
	if err := injectContractFault(fault, faultAfterInterlockSync); err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultAfterInterlockSync, err)
	}
	reopened, present, err := readExecutionInterlockState(path)
	if err != nil || !present || !sameInterlockState(reopened, state) {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	return nil
}

func reconcileKnownNoClaimLocked(store *ObjectStore, path string, held executionInterlockState, fault contractFault) error {
	current, present, err := readExecutionInterlockState(path)
	if err != nil || !present || !sameInterlockState(current, held) {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen,
			refuse(codeInterlockAmbiguous, "known-no-claim reconciliation lost its exact generation", err))
	}
	clear, err := newExecutionInterlockState(
		interlockStateClear, held.bootDigest, "", "", deriveInterlockGeneration(
			held.digest, held.targetDigest, held.attemptDigest, held.bootDigest, held.revision+1,
		), held.revision+1,
	)
	if err != nil {
		return err
	}
	if err := replaceExecutionInterlock(path, clear, true, fault); err != nil {
		return err
	}
	return persistClearReceiptLocked(store, clear, held, "KNOWN_NO_CLAIM", "", fault)
}

func persistClearReceiptLocked(
	store *ObjectStore,
	clear executionInterlockState,
	previous executionInterlockState,
	cause string,
	runDigest domain.Digest,
	fault contractFault,
) error {
	if store == nil || !clear.valid() || clear.state != interlockStateClear || !previous.valid() ||
		previous.state != interlockStateHeld ||
		(cause != "FINALIZED_RUN" && cause != "CHANGED_BOOT" && cause != "KNOWN_NO_CLAIM") ||
		(cause == "FINALIZED_RUN" && !runDigest.Valid()) || (cause != "FINALIZED_RUN" && runDigest.Valid()) ||
		clear.revision != previous.revision+1 ||
		clear.generation != deriveInterlockGeneration(
			previous.digest, previous.targetDigest, previous.attemptDigest, clear.bootDigest, clear.revision,
		) ||
		(cause == "CHANGED_BOOT" && clear.bootDigest == previous.bootDigest) ||
		(cause != "CHANGED_BOOT" && clear.bootDigest != previous.bootDigest) {
		return refuse(codeInterlockAmbiguous, "clear receipt inputs are invalid", nil)
	}
	directory, err := store.ensureContractDirectoryLocked(store.contractOps, clearReceiptDirectory)
	if err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	body, err := clearReceiptBytes(clear, previous, cause, runDigest)
	if err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	hex, _ := strictDigestHex(clear.digest)
	receiptFault := func(phase contractFaultPhase) error {
		switch phase {
		case faultBeforeTemporary:
			return injectContractFault(fault, faultBeforeClearReceipt)
		case faultAfterTempSync:
			return injectContractFault(fault, faultAfterClearReceiptTemp)
		case faultBeforeLink:
			return injectContractFault(fault, faultBeforeClearReceiptLink)
		case faultAfterLink:
			return injectContractFault(fault, faultAfterClearReceiptLink)
		case faultAfterParentSync:
			return injectContractFault(fault, faultAfterClearReceiptSync)
		case faultBeforeReopen:
			return injectContractFault(fault, faultBeforeClearReceiptOpen)
		default:
			return nil
		}
	}
	if _, err := createExactPrivateFile(directory, hex, body, receiptFault); err != nil {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if !clearReceiptExactLocked(store, clear) {
		return mutationFailure(codeInterlockAmbiguous, contractAmbiguous, faultBeforeReopen,
			refuse(codeInterlockAmbiguous, "clear receipt did not reopen exactly", nil))
	}
	return nil
}

func clearReceiptBytes(
	clear executionInterlockState,
	previous executionInterlockState,
	cause string,
	runDigest domain.Digest,
) ([]byte, error) {
	return canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": clearReceiptKind,
		"receipt_version": clearReceiptVersionV1, "cause": cause,
		"clear_state_digest": clear.digest.String(), "previous_held_digest": previous.digest.String(),
		"previous_target_digest": previous.targetDigest.String(), "previous_attempt_digest": previous.attemptDigest.String(),
		"previous_boot_digest": previous.bootDigest.String(), "previous_generation": previous.generation.String(),
		"previous_revision": previous.revision, "boot_digest": clear.bootDigest.String(),
		"terminal_run_digest": runDigest.String(),
	})
}

func clearReceiptExact(store *ObjectStore, clear executionInterlockState) bool {
	if store == nil || store.instance == nil || !clear.valid() || clear.state != interlockStateClear {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return clearReceiptExactLocked(store, clear)
}

func clearReceiptExactLocked(store *ObjectStore, clear executionInterlockState) bool {
	if store == nil || !clear.valid() || clear.state != interlockStateClear {
		return false
	}
	hex, err := strictDigestHex(clear.digest)
	if err != nil {
		return false
	}
	directory, err := store.ensureContractDirectoryLocked(store.contractOps, clearReceiptDirectory)
	if err != nil {
		return false
	}
	body, err := readExactPrivateFile(filepath.Join(directory, hex), -1)
	if err != nil {
		return false
	}
	value, err := canon.Parse(body)
	if err != nil || !contractRoster(value,
		"schema_version", "kind", "receipt_version", "cause", "clear_state_digest",
		"previous_held_digest", "previous_target_digest", "previous_attempt_digest",
		"previous_boot_digest", "previous_generation", "previous_revision", "boot_digest", "terminal_run_digest",
	) {
		return false
	}
	readText := func(name string) (string, bool) {
		member, ok := value.LookupMember(name)
		if !ok {
			return "", false
		}
		result, ok := member.Text()
		return result, ok
	}
	schema, schemaOK := readText("schema_version")
	kind, kindOK := readText("kind")
	version, versionOK := readText("receipt_version")
	cause, causeOK := readText("cause")
	clearRaw, clearOK := readText("clear_state_digest")
	previousRaw, previousOK := readText("previous_held_digest")
	previousTargetRaw, previousTargetOK := readText("previous_target_digest")
	previousAttemptRaw, previousAttemptOK := readText("previous_attempt_digest")
	previousBootRaw, previousBootOK := readText("previous_boot_digest")
	previousGenerationRaw, previousGenerationOK := readText("previous_generation")
	bootRaw, bootOK := readText("boot_digest")
	runRaw, runOK := readText("terminal_run_digest")
	if !schemaOK || !kindOK || !versionOK || !causeOK || !clearOK || !previousOK || !bootOK || !runOK ||
		!previousTargetOK || !previousAttemptOK || !previousBootOK || !previousGenerationOK ||
		schema != domain.SchemaVersion || kind != clearReceiptKind || version != clearReceiptVersionV1 ||
		clearRaw != clear.digest.String() || bootRaw != clear.bootDigest.String() ||
		(cause != "FINALIZED_RUN" && cause != "CHANGED_BOOT" && cause != "KNOWN_NO_CLAIM") {
		return false
	}
	previous, previousErr := domain.ParseDigest(previousRaw)
	if previousErr != nil || !previous.Valid() {
		return false
	}
	previousTarget, targetErr := domain.ParseDigest(previousTargetRaw)
	previousAttempt, attemptErr := domain.ParseDigest(previousAttemptRaw)
	previousBoot, bootErr := domain.ParseDigest(previousBootRaw)
	previousGeneration, generationErr := domain.ParseDigest(previousGenerationRaw)
	revisionValue, revisionPresent := value.LookupMember("previous_revision")
	previousRevision, revisionOK := revisionValue.Int64()
	if targetErr != nil || attemptErr != nil || bootErr != nil || generationErr != nil || !revisionPresent || !revisionOK ||
		previousRevision < 1 || clear.revision != previousRevision+1 ||
		(cause == "CHANGED_BOOT" && clear.bootDigest == previousBoot) ||
		(cause != "CHANGED_BOOT" && clear.bootDigest != previousBoot) {
		return false
	}
	previousState, err := newExecutionInterlockState(
		interlockStateHeld, previousBoot, previousTarget, previousAttempt, previousGeneration, previousRevision,
	)
	if err != nil || previousState.digest != previous {
		return false
	}
	expectedClear, err := newExecutionInterlockState(
		interlockStateClear, clear.bootDigest, "", "", deriveInterlockGeneration(
			previousState.digest, previousState.targetDigest, previousState.attemptDigest, clear.bootDigest, clear.revision,
		), clear.revision,
	)
	if err != nil || !sameInterlockState(expectedClear, clear) {
		return false
	}
	var run domain.Digest
	if cause == "FINALIZED_RUN" {
		run, err = domain.ParseDigest(runRaw)
		if err != nil || !run.Valid() {
			return false
		}
	} else if runRaw != "" {
		return false
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return false
	}
	if cause == "FINALIZED_RUN" && !finalizedRunClearLinkExactLocked(store, clear, previousState, run, body) {
		return false
	}
	effect, err := createExactPrivateFile(directory, hex, body, nil)
	return err == nil && effect == contractExactConverged && store.assertReady() == nil
}

func finalizedRunClearLinkExactLocked(
	store *ObjectStore,
	clear executionInterlockState,
	previous executionInterlockState,
	runDigest domain.Digest,
	receiptBody []byte,
) bool {
	if store == nil || !clear.valid() || clear.state != interlockStateClear ||
		!previous.valid() || previous.state != interlockStateHeld || !runDigest.Valid() {
		return false
	}
	claimDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, startClaimDirectory)
	if err != nil {
		return false
	}
	claimPath := claimPathFor(claimDirectory, previous.targetDigest)
	claimBody, err := readExactPrivateFile(claimPath, -1)
	if err != nil {
		return false
	}
	claim, err := parseStartClaim(store, claimBody, claimPath)
	if err != nil || !claim.validForLocked(store) || claim.targetDigest != previous.targetDigest ||
		claim.attemptDigest != previous.attemptDigest || claim.bootDigest != previous.bootDigest ||
		claim.generation != previous.generation {
		return false
	}
	receiptDigestValue, err := canon.DigestBytes(clearReceiptKind, receiptBody)
	if err != nil {
		return false
	}
	receiptDigest, err := domain.ParseDigest(receiptDigestValue.String())
	if err != nil {
		return false
	}
	releaseDirectory, err := store.ensureContractDirectoryLocked(store.contractOps, finalizedReleaseDirectory)
	if err != nil {
		return false
	}
	runHex, err := strictDigestHex(runDigest)
	if err != nil {
		return false
	}
	releasePath := filepath.Join(releaseDirectory, runHex)
	if err := rejectPathCaseAlias(releasePath); err != nil {
		return false
	}
	releaseBody, err := readExactPrivateFile(releasePath, -1)
	if err != nil {
		return false
	}
	linkedClear, linkedReceipt, err := parseFinalizedRunRelease(releaseBody, claim, runDigest)
	return err == nil && linkedClear == clear.digest && linkedReceipt == receiptDigest
}

func deriveInterlockGeneration(
	previous, target, attempt, boot domain.Digest,
	revision int64,
) domain.Digest {
	digest, _, err := canon.DigestTyped("ExecutionInterlockGeneration", map[string]any{
		"schema_version": domain.SchemaVersion, "kind": "ExecutionInterlockGeneration",
		"previous_state_digest": previous.String(), "target_digest": target.String(),
		"attempt_digest": attempt.String(), "boot_digest": boot.String(), "revision": revision,
	})
	if err != nil {
		return ""
	}
	parsed, _ := domain.ParseDigest(digest.String())
	return parsed
}

func claimPathFor(directory string, target domain.Digest) string {
	hex, _ := strictDigestHex(target)
	return filepath.Join(directory, hex)
}

func optionalInterlockDigest(raw string) (domain.Digest, error) {
	if raw == "" {
		return "", nil
	}
	return domain.ParseDigest(raw)
}

func sameInterlockState(left, right executionInterlockState) bool {
	return left.valid() && right.valid() && left.digest == right.digest && bytes.Equal(left.canonical, right.canonical)
}
