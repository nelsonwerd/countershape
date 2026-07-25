package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	contractPublicationDirectory = "publications"
	targetLinkDirectory          = "target-by-attempt"
	runLinkDirectory             = "run-by-target"
	executionLinkDirectory       = "execution-by-run-profile"
	contractNamespaceLock        = ".lock"

	contractTargetKind    = "ContractExecutionTarget"
	contractRunKind       = "FinalizedContractRun"
	contractExecutionKind = "ContractExecution"
	contractAttemptKind   = "ConformanceAttempt"
	contractClaimKind     = "StartClaim"
	contractProfileKind   = "ContractExecutionClassifierProfile"
	contractProfileV1     = "CONTRACT_EXECUTION_EXACT_TUPLE_V1"
	contractAttemptV1     = "contract-conformance-attempt/v1"
	contractAttemptsDir   = "conformance-attempts"
	contractAttemptMarker = "attempt.marker.json"

	relationAttemptTarget = "CONFORMANCE_ATTEMPT_TO_TARGET"
	relationTargetRun     = "TARGET_TO_FINALIZED_RUN"
	relationRunExecution  = "RUN_PROFILE_TO_EXECUTION"
	relationVersionV1     = "contract-storage-relation/v1"

	codeContractRecordRefused   = "CONTRACT_RECORD_REFUSED"
	codeContractRecordConflict  = "CONTRACT_RECORD_CONFLICT"
	codeContractRecordAmbiguous = "CONTRACT_RECORD_AMBIGUOUS"
)

type contractEffect string

const (
	contractKnownNoEffect  contractEffect = "KNOWN_NO_EFFECT"
	contractExactConverged contractEffect = "EXACT_CONVERGED"
	contractAmbiguous      contractEffect = "AMBIGUOUS"
)

type contractFaultPhase string
type contractFault func(contractFaultPhase) error

const (
	faultBeforeTemporary contractFaultPhase = "before-temporary-create"
	faultAfterTempSync   contractFaultPhase = "after-temporary-sync"
	faultBeforeLink      contractFaultPhase = "before-create-once-link"
	faultAfterLink       contractFaultPhase = "after-create-once-link"
	faultAfterParentSync contractFaultPhase = "after-parent-sync"
	faultBeforeReopen    contractFaultPhase = "before-exact-reopen"
)

type contractMutationError struct {
	effect contractEffect
	phase  contractFaultPhase
	cause  error
}

func (e *contractMutationError) Error() string {
	return fmt.Sprintf("%s at %s: %v", e.effect, e.phase, e.cause)
}

func (e *contractMutationError) Unwrap() error { return e.cause }

func mutationFailure(code string, effect contractEffect, phase contractFaultPhase, cause error) error {
	return refuse(code, string(effect), &contractMutationError{effect: effect, phase: phase, cause: cause})
}

func mutationEffect(err error) contractEffect {
	best := contractEffect("")
	var inspect func(error)
	inspect = func(current error) {
		if current == nil {
			return
		}
		if failure, ok := current.(*contractMutationError); ok {
			switch failure.effect {
			case contractAmbiguous:
				best = contractAmbiguous
			case contractExactConverged:
				if best != contractAmbiguous {
					best = contractExactConverged
				}
			case contractKnownNoEffect:
				if best == "" {
					best = contractKnownNoEffect
				}
			}
		}
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range wrapped.Unwrap() {
				inspect(child)
			}
		case interface{ Unwrap() error }:
			inspect(wrapped.Unwrap())
		}
	}
	inspect(err)
	return best
}

func injectContractFault(fault contractFault, phase contractFaultPhase) error {
	if fault == nil {
		return nil
	}
	return fault(phase)
}

type attemptStorageRecord struct {
	storeInstance *objectStoreInstance
	digest        domain.Digest
	seal          *attemptRecordSeal
	state         *conformanceAttemptState
}

type attemptRecordSeal struct{ marker byte }

func (record attemptStorageRecord) validFor(store *ObjectStore) bool {
	if store == nil || store.instance == nil {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return record.validForLocked(store)
}

func (record attemptStorageRecord) validForLocked(store *ObjectStore) bool {
	if store == nil || record.storeInstance != store.instance || record.seal == nil ||
		record.seal.marker != 1 || !record.digest.Valid() {
		return false
	}
	return record.state == nil || record.state.validForLocked(store, record.digest)
}

// ConformanceAttemptInput binds a fresh inert attempt root to the live facts
// C3 has already reopened. The store generates the nonce; callers cannot pick
// a durable attempt identity.
type ConformanceAttemptInput struct {
	ContractBundleDigest        domain.Digest
	ResidueHeadDigest           domain.Digest
	TreeIdentityDigest          domain.Digest
	MaterializationPolicyDigest domain.Digest
}

type conformanceAttemptRoots struct {
	attempt, candidateParent, fixture, home, temporary string
	xdgConfig, xdgCache, xdgData, xdgState, state      string
	evidence, marker                                   string
}

// ConformanceAttemptRoots exposes only fixed private paths derived by the
// store. Paths confer no attempt or target authority on their own.
type ConformanceAttemptRoots struct{ roots conformanceAttemptRoots }

func (roots ConformanceAttemptRoots) AttemptRoot() string     { return roots.roots.attempt }
func (roots ConformanceAttemptRoots) CandidateParent() string { return roots.roots.candidateParent }
func (roots ConformanceAttemptRoots) FixtureRoot() string     { return roots.roots.fixture }
func (roots ConformanceAttemptRoots) HomeRoot() string        { return roots.roots.home }
func (roots ConformanceAttemptRoots) TemporaryRoot() string   { return roots.roots.temporary }
func (roots ConformanceAttemptRoots) XDGConfigRoot() string   { return roots.roots.xdgConfig }
func (roots ConformanceAttemptRoots) XDGCacheRoot() string    { return roots.roots.xdgCache }
func (roots ConformanceAttemptRoots) XDGDataRoot() string     { return roots.roots.xdgData }
func (roots ConformanceAttemptRoots) XDGStateRoot() string    { return roots.roots.xdgState }
func (roots ConformanceAttemptRoots) StateRoot() string       { return roots.roots.state }
func (roots ConformanceAttemptRoots) EvidenceRoot() string    { return roots.roots.evidence }
func (roots ConformanceAttemptRoots) MarkerPath() string      { return roots.roots.marker }

type conformanceAttemptState struct {
	input     ConformanceAttemptInput
	nonce     string
	canonical []byte
	object    SemanticObject
	authority ObjectAuthority
	roots     conformanceAttemptRoots
	dirs      map[string]os.FileInfo
	marker    os.FileInfo
	seal      *attemptRecordSeal
}

// ConformanceAttemptRecord is a durable but inert store-bound attachment. It
// cannot issue OfficialTarget, process, interlock, or permit authority.
type ConformanceAttemptRecord struct {
	store  *ObjectStore
	record attemptStorageRecord
}

func (record ConformanceAttemptRecord) Valid() bool {
	return record.store != nil && record.record.validFor(record.store) && record.record.state != nil
}
func (record ConformanceAttemptRecord) Digest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.digest
}
func (record ConformanceAttemptRecord) ContractBundleDigest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.state.input.ContractBundleDigest
}
func (record ConformanceAttemptRecord) ResidueHeadDigest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.state.input.ResidueHeadDigest
}
func (record ConformanceAttemptRecord) TreeIdentityDigest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.state.input.TreeIdentityDigest
}
func (record ConformanceAttemptRecord) MaterializationPolicyDigest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.state.input.MaterializationPolicyDigest
}
func (record ConformanceAttemptRecord) InstanceNonce() string {
	if !record.Valid() {
		return ""
	}
	return record.record.state.nonce
}
func (record ConformanceAttemptRecord) Roots() ConformanceAttemptRoots {
	if !record.Valid() {
		return ConformanceAttemptRoots{}
	}
	return ConformanceAttemptRoots{roots: record.record.state.roots}
}

// ContractTargetRecord is the public inert wrapper over C2's exact typed
// relationship. It exposes identity only, never semantic bytes or issuance.
type ContractTargetRecord struct {
	store  *ObjectStore
	record targetStorageRecord
}

func (record ContractTargetRecord) Valid() bool {
	return record.store != nil && record.record.validFor(record.store)
}
func (record ContractTargetRecord) Digest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.record.object.Digest()
}
func (record ContractTargetRecord) AttemptDigest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.record.record.relation.attemptDigest
}

var conformanceAttemptRootNames = [...]string{
	"candidate-parent", "evidence", "fixture", "home", "state", "tmp",
	"xdg-cache", "xdg-config", "xdg-data", "xdg-state",
}

func buildConformanceAttempt(
	input ConformanceAttemptInput,
	nonce string,
) (SemanticObject, []byte, error) {
	if !input.ContractBundleDigest.Valid() || !input.ResidueHeadDigest.Valid() ||
		!input.TreeIdentityDigest.Valid() || !input.MaterializationPolicyDigest.Valid() ||
		len(nonce) != 64 || strings.ToLower(nonce) != nonce {
		return SemanticObject{}, nil, refuse(codeContractRecordRefused, "attempt input is incomplete", nil)
	}
	if decoded, err := hex.DecodeString(nonce); err != nil || len(decoded) != 32 {
		return SemanticObject{}, nil, refuse(codeContractRecordRefused, "attempt nonce is malformed", err)
	}
	digest, exact, err := canon.DigestTyped(contractAttemptKind, map[string]any{
		"schema_version": domain.SchemaVersion, "kind": contractAttemptKind, "attempt_version": contractAttemptV1,
		"attempt_purpose": "CONFORMANCE", "contract_bundle_digest": input.ContractBundleDigest.String(),
		"residue_head_digest": input.ResidueHeadDigest.String(), "tree_identity_digest": input.TreeIdentityDigest.String(),
		"materialization_policy_digest": input.MaterializationPolicyDigest.String(), "instance_nonce": nonce,
		"allocation_profile": "PRIVATE_FRESH_ROOT_V1", "root_names": append([]string(nil), conformanceAttemptRootNames[:]...),
		"marker_ordering": "DURABLE_BEFORE_SPAWN",
	})
	if err != nil {
		return SemanticObject{}, nil, refuse(codeContractRecordRefused, "attempt marker could not be encoded", err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return SemanticObject{}, nil, err
	}
	object, err := NewSemanticObject(contractAttemptKind, parsed, exact)
	return object, append([]byte(nil), exact...), err
}

// AllocateConformanceAttempt creates a new private root exactly once. Any
// partial failure leaves only inert orphan storage and returns no record.
func (store *ObjectStore) AllocateConformanceAttempt(ctx context.Context, input ConformanceAttemptInput) (ConformanceAttemptRecord, error) {
	if store == nil || store.instance == nil {
		return ConformanceAttemptRecord{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return ConformanceAttemptRecord{}, err
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordRefused, "fresh attempt nonce allocation failed", err)
	}
	nonce := hex.EncodeToString(random[:])
	object, exact, err := buildConformanceAttempt(input, nonce)
	if err != nil {
		return ConformanceAttemptRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return ConformanceAttemptRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordRefused, "attempt namespace lock failed", err)
	}
	defer lock.release()
	base, err := store.ensureContractDirectoryLocked(store.contractRuns, contractAttemptsDir)
	if err != nil {
		return ConformanceAttemptRecord{}, err
	}
	hexDigest, err := strictDigestHex(object.Digest())
	if err != nil {
		return ConformanceAttemptRecord{}, err
	}
	if err := rejectCaseAlias(base, hexDigest); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordConflict, "attempt root aliases an existing name", err)
	}
	attemptRoot := filepath.Join(base, hexDigest)
	if err := os.Mkdir(attemptRoot, 0o700); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordConflict, "fresh attempt root already exists or cannot be created", err)
	}
	if err := os.Chmod(attemptRoot, 0o700); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt root mode failed", err)
	}
	if err := syncDirectory(base); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt-root parent sync failed", err)
	}
	dirs := make(map[string]os.FileInfo, len(conformanceAttemptRootNames)+1)
	rootInfo, err := exactPrivateDirectoryInfo(attemptRoot)
	if err != nil {
		return ConformanceAttemptRecord{}, err
	}
	dirs[attemptRoot] = rootInfo
	for _, name := range conformanceAttemptRootNames {
		path := filepath.Join(attemptRoot, name)
		if err := os.Mkdir(path, 0o700); err != nil {
			return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt child allocation failed", err)
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt child mode failed", err)
		}
		if err := syncDirectory(attemptRoot); err != nil {
			return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt child parent sync failed", err)
		}
		info, err := exactPrivateDirectoryInfo(path)
		if err != nil {
			return ConformanceAttemptRecord{}, err
		}
		dirs[path] = info
	}
	authority, err := store.publishLocked(ctx, object)
	if err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "attempt object publication failed", err)
	}
	roots := rootsAt(attemptRoot)
	if _, err := createExactPrivateFile(roots.evidence, contractAttemptMarker, exact, nil); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "attempt marker publication failed", err)
	}
	marker, err := exactObjectInfo(roots.marker, int64(len(exact)))
	if err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "attempt marker did not reopen", err)
	}
	if err := syncDirectory(attemptRoot); err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "attempt root durability sync failed", err)
	}
	state := &conformanceAttemptState{
		input: input, nonce: nonce, canonical: exact, object: object, authority: authority,
		roots: roots, dirs: dirs, marker: marker, seal: &attemptRecordSeal{marker: 1},
	}
	internal := attemptStorageRecord{storeInstance: store.instance, digest: object.Digest(), seal: state.seal, state: state}
	if !state.validForLocked(store, object.Digest()) {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordAmbiguous, "fresh attempt failed exact reopen", nil)
	}
	return ConformanceAttemptRecord{store: store, record: internal}, nil
}

// OpenConformanceAttempt reconstructs inert attempt authority from its exact
// digest after a process restart. It does not discover or select attempts.
func (store *ObjectStore) OpenConformanceAttempt(ctx context.Context, digest domain.Digest) (ConformanceAttemptRecord, error) {
	if store == nil || store.instance == nil || !digest.Valid() {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordRefused, "store and exact attempt digest are required", nil)
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return ConformanceAttemptRecord{}, err
	}
	if err := store.assertReady(); err != nil {
		return ConformanceAttemptRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return ConformanceAttemptRecord{}, refuse(codeContractRecordRefused, "attempt namespace lock failed", err)
	}
	defer lock.release()
	record, err := store.openConformanceAttemptLocked(digest)
	if err != nil {
		return ConformanceAttemptRecord{}, err
	}
	return ConformanceAttemptRecord{store: store, record: record}, nil
}

func (store *ObjectStore) openConformanceAttemptLocked(digest domain.Digest) (attemptStorageRecord, error) {
	hexDigest, err := strictDigestHex(digest)
	if err != nil {
		return attemptStorageRecord{}, err
	}
	base := filepath.Join(store.contractRuns, contractAttemptsDir)
	baseInfo, err := exactPrivateDirectoryInfo(base)
	if err != nil {
		return attemptStorageRecord{}, refuse(codeContractRecordRefused, "attempt namespace is absent or changed", err)
	}
	store.contractInfos[base] = baseInfo
	if err := rejectCaseAlias(base, hexDigest); err != nil {
		return attemptStorageRecord{}, refuse(codeContractRecordRefused, "attempt root changed exact spelling", err)
	}
	attemptRoot := filepath.Join(base, hexDigest)
	roots := rootsAt(attemptRoot)
	exact, err := readExactPrivateFile(roots.marker, -1)
	if err != nil {
		return attemptStorageRecord{}, refuse(codeContractRecordRefused, "attempt marker is absent or invalid", err)
	}
	input, nonce, err := parseConformanceAttempt(exact, digest)
	if err != nil {
		return attemptStorageRecord{}, err
	}
	object, err := NewSemanticObject(contractAttemptKind, digest, exact)
	if err != nil {
		return attemptStorageRecord{}, err
	}
	objectPath, _, err := store.existingObjectPath(digest)
	if err != nil {
		return attemptStorageRecord{}, err
	}
	reopened, err := reopenExactObject(objectPath, object)
	if err != nil || !sameSemanticObject(reopened, object) {
		return attemptStorageRecord{}, refuse(codeContractRecordRefused, "attempt object did not reopen", err)
	}
	dirs, marker, err := inspectAttemptRoots(roots, exact)
	if err != nil {
		return attemptStorageRecord{}, err
	}
	seal := &attemptRecordSeal{marker: 1}
	state := &conformanceAttemptState{
		input: input, nonce: nonce, canonical: exact, object: object, authority: store.authority(object),
		roots: roots, dirs: dirs, marker: marker, seal: seal,
	}
	record := attemptStorageRecord{storeInstance: store.instance, digest: digest, seal: seal, state: state}
	if !state.validForLocked(store, digest) {
		return attemptStorageRecord{}, refuse(codeContractRecordRefused, "attempt graph failed exact reconstruction", nil)
	}
	return record, nil
}

// PersistContractTargetRecord preserves one valid inert model target through
// C2's object/witness/relation mechanics and returns only an opaque record. If
// cooperating callers race different valid children for one attempt, the first
// exact durable relationship wins; this layer does not arbitrate semantic intent.
func (store *ObjectStore) PersistContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord, target contractmodel.ContractExecutionTarget) (ContractTargetRecord, error) {
	if store == nil || attempt.store != store || !attempt.Valid() || !target.Valid() ||
		!attemptJoinsTarget(attempt.record.state, target) {
		return ContractTargetRecord{}, refuse(codeContractRecordRefused, "attempt and inert target do not join", nil)
	}
	object, err := NewSemanticObject(contractTargetKind, target.Digest(), target.CanonicalBytes())
	if err != nil {
		return ContractTargetRecord{}, err
	}
	_, _, persistErr := persistTargetRecord(ctx, store, targetStorageInput{attempt: attempt.record, object: object}, nil)
	reopened, openErr := openTargetByAttempt(ctx, store, attempt.record)
	if openErr != nil || reopened.record.object.Digest() != target.Digest() ||
		!bytes.Equal(reopened.record.object.CanonicalBytes(), target.CanonicalBytes()) {
		return ContractTargetRecord{}, refuse(codeContractRecordAmbiguous, "target publication did not reconcile exactly", errors.Join(persistErr, openErr))
	}
	return ContractTargetRecord{store: store, record: reopened}, nil
}

// OpenContractTargetRecord reopens only the typed child of one live attempt
// attachment; it performs no discovery and issues no OfficialTarget.
func (store *ObjectStore) OpenContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord) (ContractTargetRecord, error) {
	if store == nil || attempt.store != store || !attempt.Valid() {
		return ContractTargetRecord{}, refuse(codeContractRecordRefused, "live attempt attachment is required", nil)
	}
	record, err := openTargetByAttempt(ctx, store, attempt.record)
	if err != nil {
		return ContractTargetRecord{}, err
	}
	target, err := parseContractTargetStorageRecord(record)
	if err != nil || !attemptJoinsTarget(attempt.record.state, target) {
		return ContractTargetRecord{}, refuse(codeContractRecordRefused, "reopened target differs from its attempt graph", err)
	}
	return ContractTargetRecord{store: store, record: record}, nil
}

func parseContractTargetStorageRecord(record targetStorageRecord) (contractmodel.ContractExecutionTarget, error) {
	if !record.record.object.Valid() || record.record.object.Kind() != contractTargetKind {
		return contractmodel.ContractExecutionTarget{}, refuse(codeContractRecordRefused, "target storage record is invalid", nil)
	}
	target, err := contractmodel.ParseContractExecutionTarget(
		record.record.object.CanonicalBytes(), record.record.object.Digest(),
	)
	if err != nil {
		return contractmodel.ContractExecutionTarget{}, refuse(codeContractRecordRefused, "target model did not reconstruct exactly", err)
	}
	return target, nil
}

func attemptJoinsTarget(state *conformanceAttemptState, target contractmodel.ContractExecutionTarget) bool {
	if state == nil || !target.Valid() {
		return false
	}
	input := target.Input()
	return input.ContractBundleDigest == state.input.ContractBundleDigest &&
		input.TerminalResidue.HeadDigest == state.input.ResidueHeadDigest &&
		input.Tree.TreeIdentityDigest == state.input.TreeIdentityDigest &&
		input.Tree.MaterializationPolicyDigest == state.input.MaterializationPolicyDigest &&
		input.Attempt.ArtifactDigest == state.object.Digest() && input.Attempt.InstanceNonce == state.nonce
}

// finalizedRunSemanticObject is the store-side codec seam for an inert model
// value. Contract-run bridge code may join opaque records, but it does not own
// or reinterpret the semantic bytes.
func finalizedRunSemanticObject(run contractmodel.FinalizedContractRun) (SemanticObject, error) {
	if !run.Valid() {
		return SemanticObject{}, refuse(codeContractRecordRefused, "finalized run model is invalid", nil)
	}
	return NewSemanticObject(contractRunKind, run.Digest(), run.CanonicalBytes())
}

func parseFinalizedRunStorageRecord(
	record finalizedRunStorageRecord,
	target contractmodel.ContractExecutionTarget,
) (contractmodel.FinalizedContractRun, error) {
	if !record.record.object.Valid() || record.record.object.Kind() != contractRunKind || !target.Valid() {
		return contractmodel.FinalizedContractRun{}, refuse(codeContractRecordRefused, "finalized run record or target model is invalid", nil)
	}
	run, err := contractmodel.ParseFinalizedContractRun(
		record.record.object.CanonicalBytes(), record.record.object.Digest(), target,
	)
	if err != nil {
		return contractmodel.FinalizedContractRun{}, refuse(codeContractRecordRefused, "finalized run model did not reconstruct exactly", err)
	}
	return run, nil
}

func contractExecutionSemanticObject(execution contractmodel.ContractExecution) (SemanticObject, error) {
	if !execution.Valid() {
		return SemanticObject{}, refuse(codeContractRecordRefused, "contract execution model is invalid", nil)
	}
	return NewSemanticObject(contractExecutionKind, execution.Digest(), execution.CanonicalBytes())
}

func rootsAt(attemptRoot string) conformanceAttemptRoots {
	return conformanceAttemptRoots{
		attempt: attemptRoot, candidateParent: filepath.Join(attemptRoot, "candidate-parent"),
		fixture: filepath.Join(attemptRoot, "fixture"), home: filepath.Join(attemptRoot, "home"),
		temporary: filepath.Join(attemptRoot, "tmp"), xdgConfig: filepath.Join(attemptRoot, "xdg-config"),
		xdgCache: filepath.Join(attemptRoot, "xdg-cache"), xdgData: filepath.Join(attemptRoot, "xdg-data"),
		xdgState: filepath.Join(attemptRoot, "xdg-state"), state: filepath.Join(attemptRoot, "state"),
		evidence: filepath.Join(attemptRoot, "evidence"),
		marker:   filepath.Join(attemptRoot, "evidence", contractAttemptMarker),
	}
}

func attemptRootPaths(roots conformanceAttemptRoots) []string {
	return []string{
		roots.attempt, roots.candidateParent, roots.evidence, roots.fixture, roots.home,
		roots.state, roots.temporary, roots.xdgCache, roots.xdgConfig, roots.xdgData, roots.xdgState,
	}
}

func boundedExactDirectoryNames(path string, retained os.FileInfo, maximum int) ([]string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || retained == nil || !retained.IsDir() ||
		retained.Mode()&os.ModeSymlink != 0 || maximum < 0 || maximum > 32 {
		return nil, errors.New("bounded exact directory-roster inputs are invalid")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.IsDir() || before.Mode() != retained.Mode() || !os.SameFile(retained, before) {
		return nil, errors.Join(err, errors.New("bounded exact directory-roster root identity differs"))
	}
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.IsDir() || opened.Mode() != retained.Mode() || !os.SameFile(retained, opened) {
		return nil, errors.Join(openErr, handle.Close(), errors.New("bounded exact directory-roster descriptor differs"))
	}
	entries := make([]os.DirEntry, 0, maximum+1)
	var rosterErr error
	for {
		remaining := maximum + 1 - len(entries)
		if remaining <= 0 {
			rosterErr = errors.New("bounded exact directory-roster exceeds its direct-entry ceiling")
			break
		}
		batch, readErr := handle.ReadDir(remaining)
		entries = append(entries, batch...)
		if len(entries) > maximum {
			rosterErr = errors.New("bounded exact directory-roster exceeds its direct-entry ceiling")
			break
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				rosterErr = readErr
			}
			break
		}
		if len(batch) == 0 {
			rosterErr = errors.New("bounded exact directory-roster made no progress")
			break
		}
	}
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(path)
	if rosterErr != nil || descriptorErr != nil || closeErr != nil || pathErr != nil ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return nil, errors.Join(rosterErr, descriptorErr, closeErr, pathErr, errors.New("bounded exact directory-roster read did not close exactly"))
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	return names, nil
}

func inspectAttemptRoots(
	roots conformanceAttemptRoots,
	exact []byte,
) (map[string]os.FileInfo, os.FileInfo, error) {
	dirs := make(map[string]os.FileInfo, len(conformanceAttemptRootNames)+1)
	for _, path := range attemptRootPaths(roots) {
		info, err := exactPrivateDirectoryInfo(path)
		if err != nil {
			return nil, nil, refuse(codeContractRecordRefused, "attempt directory facts differ", err)
		}
		dirs[path] = info
	}
	names, err := boundedExactDirectoryNames(
		roots.attempt, dirs[roots.attempt], len(conformanceAttemptRootNames),
	)
	if err != nil || len(names) != len(conformanceAttemptRootNames) {
		return nil, nil, refuse(codeContractRecordRefused, "attempt root roster differs", err)
	}
	sort.Strings(names)
	if !stringRosterEqual(names, conformanceAttemptRootNames[:]) {
		return nil, nil, refuse(codeContractRecordRefused, "attempt root names differ", nil)
	}
	evidenceNames, err := boundedExactDirectoryNames(roots.evidence, dirs[roots.evidence], 1)
	if err != nil || len(evidenceNames) != 1 || evidenceNames[0] != contractAttemptMarker {
		return nil, nil, refuse(codeContractRecordRefused, "attempt evidence roster differs", err)
	}
	marker, err := exactObjectInfo(roots.marker, int64(len(exact)))
	if err != nil {
		return nil, nil, refuse(codeContractRecordRefused, "attempt marker facts differ", err)
	}
	body, err := readExactPrivateFile(roots.marker, int64(len(exact)))
	if err != nil || !bytes.Equal(body, exact) {
		return nil, nil, refuse(codeContractRecordRefused, "attempt marker bytes differ", err)
	}
	return dirs, marker, nil
}

func (state *conformanceAttemptState) validForLocked(store *ObjectStore, digest domain.Digest) bool {
	if state == nil || store == nil || state.seal == nil || state.seal.marker != 1 ||
		!state.object.Valid() || state.object.Kind() != contractAttemptKind || state.object.Digest() != digest ||
		state.authority.storeInstance != store.instance || !sameSemanticObject(state.authority.object, state.object) ||
		!bytes.Equal(state.canonical, state.object.CanonicalBytes()) || state.nonce == "" || store.assertReady() != nil {
		return false
	}
	hexDigest, err := strictDigestHex(digest)
	base := filepath.Join(store.contractRuns, contractAttemptsDir)
	if err != nil || rejectCaseAlias(base, hexDigest) != nil || state.roots.attempt != filepath.Join(base, hexDigest) ||
		state.roots != rootsAt(state.roots.attempt) {
		return false
	}
	input, nonce, err := parseConformanceAttempt(state.canonical, digest)
	if err != nil || input != state.input || nonce != state.nonce {
		return false
	}
	currentDirs, currentMarker, err := inspectAttemptRoots(state.roots, state.canonical)
	if err != nil || len(currentDirs) != len(state.dirs) || state.marker == nil || !os.SameFile(state.marker, currentMarker) {
		return false
	}
	for path, retained := range state.dirs {
		current, present := currentDirs[path]
		if !present || retained == nil || !os.SameFile(retained, current) {
			return false
		}
	}
	objectPath, _, err := store.existingObjectPath(digest)
	if err != nil {
		return false
	}
	reopened, err := reopenExactObject(objectPath, state.object)
	return err == nil && sameSemanticObject(reopened, state.object)
}

func parseConformanceAttempt(
	exact []byte,
	expected domain.Digest,
) (ConformanceAttemptInput, string, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt marker is not canonical JSON", err)
	}
	canonical, err := value.CanonicalChecked()
	members, object := value.Members()
	if err != nil || !object || len(members) != 12 || !bytes.Equal(canonical, exact) {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt marker roster or canonical bytes differ", err)
	}
	text := func(name string) (string, bool) {
		member, present := value.LookupMember(name)
		if !present {
			return "", false
		}
		return member.Text()
	}
	schema, schemaOK := text("schema_version")
	kind, kindOK := text("kind")
	version, versionOK := text("attempt_version")
	purpose, purposeOK := text("attempt_purpose")
	allocation, allocationOK := text("allocation_profile")
	ordering, orderingOK := text("marker_ordering")
	nonce, nonceOK := text("instance_nonce")
	if !schemaOK || !kindOK || !versionOK || !purposeOK || !allocationOK || !orderingOK || !nonceOK ||
		schema != domain.SchemaVersion || kind != contractAttemptKind || version != contractAttemptV1 ||
		purpose != "CONFORMANCE" || allocation != "PRIVATE_FRESH_ROOT_V1" || ordering != "DURABLE_BEFORE_SPAWN" {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt marker constants differ", nil)
	}
	parseDigest := func(name string) (domain.Digest, error) {
		raw, ok := text(name)
		if !ok {
			return "", errors.New("attempt digest member is absent")
		}
		return domain.ParseDigest(raw)
	}
	input := ConformanceAttemptInput{}
	input.ContractBundleDigest, err = parseDigest("contract_bundle_digest")
	if err != nil {
		return ConformanceAttemptInput{}, "", err
	}
	input.ResidueHeadDigest, err = parseDigest("residue_head_digest")
	if err != nil {
		return ConformanceAttemptInput{}, "", err
	}
	input.TreeIdentityDigest, err = parseDigest("tree_identity_digest")
	if err != nil {
		return ConformanceAttemptInput{}, "", err
	}
	input.MaterializationPolicyDigest, err = parseDigest("materialization_policy_digest")
	if err != nil {
		return ConformanceAttemptInput{}, "", err
	}
	rootsValue, present := value.LookupMember("root_names")
	elements, array := rootsValue.Elements()
	if !present || !array || len(elements) != len(conformanceAttemptRootNames) {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt root-name roster differs", nil)
	}
	rootNames := make([]string, len(elements))
	for index, element := range elements {
		rootNames[index], present = element.Text()
		if !present {
			return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt root name is not text", nil)
		}
	}
	if !stringRosterEqual(rootNames, conformanceAttemptRootNames[:]) {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt root-name order differs", nil)
	}
	rebuilt, _, err := buildConformanceAttempt(input, nonce)
	if err != nil || rebuilt.Digest() != expected || !bytes.Equal(rebuilt.CanonicalBytes(), exact) {
		return ConformanceAttemptInput{}, "", refuse(codeContractRecordRefused, "attempt marker digest differs", err)
	}
	return input, nonce, nil
}

func stringRosterEqual(left, right []string) bool {
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

type targetStorageInput struct {
	attempt attemptStorageRecord
	object  SemanticObject
}

type runStorageInput struct {
	target   targetStorageRecord
	claim    startClaimRecord
	object   SemanticObject
	manifest privateManifestRecord
}

type executionStorageInput struct {
	run    finalizedRunStorageRecord
	object SemanticObject
}

type contractRelation struct {
	relation                string
	parentKind              string
	parentDigest            domain.Digest
	secondaryParentKind     string
	secondaryParentDigest   domain.Digest
	childKind               string
	childDigest             domain.Digest
	targetDigest            domain.Digest
	attemptDigest           domain.Digest
	startClaimDigest        domain.Digest
	classifierProfileDigest domain.Digest
}

type contractStorageRecord struct {
	storeInstance *objectStoreInstance
	object        SemanticObject
	authority     ObjectAuthority
	witnessPath   string
	relationPath  string
	relation      contractRelation
	relationBytes []byte
}

type targetStorageRecord struct{ record contractStorageRecord }
type finalizedRunStorageRecord struct{ record contractStorageRecord }
type executionStorageRecord struct{ record contractStorageRecord }

func persistTargetRecord(
	ctx context.Context,
	store *ObjectStore,
	input targetStorageInput,
	fault contractFault,
) (targetStorageRecord, contractEffect, error) {
	if store == nil || !input.attempt.validFor(store) || input.object.Kind() != contractTargetKind {
		return targetStorageRecord{}, "", refuse(codeContractRecordRefused, "target input is not an inert typed store input", nil)
	}
	attemptDigest, err := targetAttemptDigest(input.object)
	if err != nil || attemptDigest != input.attempt.digest {
		return targetStorageRecord{}, "", refuse(codeContractRecordRefused, "target attempt join differs", err)
	}
	relation := contractRelation{
		relation: relationAttemptTarget, parentKind: contractAttemptKind, parentDigest: attemptDigest,
		childKind: contractTargetKind, childDigest: input.object.Digest(), targetDigest: input.object.Digest(),
		attemptDigest: attemptDigest,
	}
	record, effect, err := persistContractRecord(ctx, store, input.object, relation, fault)
	return targetStorageRecord{record: record}, effect, err
}

func persistFinalizedRunRecord(
	ctx context.Context,
	store *ObjectStore,
	input runStorageInput,
	fault contractFault,
) (finalizedRunStorageRecord, contractEffect, error) {
	if store == nil || !input.target.validFor(store) || !input.claim.validFor(store) ||
		!startClaimJoinsTarget(input.target, input.claim) || !input.manifest.validFor(store) ||
		input.object.Kind() != contractRunKind {
		return finalizedRunStorageRecord{}, "", refuse(codeContractRecordRefused, "run input lacks exact store-bound parents", nil)
	}
	targetDigest, attemptDigest, claimDigest, manifestDigest, err := finalizedRunJoins(input.object)
	if err != nil || targetDigest != input.target.record.object.Digest() ||
		attemptDigest != input.target.record.relation.attemptDigest || claimDigest != input.claim.digest ||
		manifestDigest != input.manifest.digest || input.manifest.targetDigest != targetDigest ||
		input.manifest.attemptDigest != attemptDigest || input.manifest.startClaimDigest != claimDigest {
		return finalizedRunStorageRecord{}, "", refuse(codeContractRecordRefused, "finalized-run parent or private-manifest join differs", err)
	}
	if err := validatePrivateManifestForFinalization(ctx, store, input.manifest); err != nil {
		return finalizedRunStorageRecord{}, "", err
	}
	if err := validatePrivateManifestRoster(input.object, input.manifest); err != nil {
		return finalizedRunStorageRecord{}, "", err
	}
	relation := contractRelation{
		relation: relationTargetRun, parentKind: contractTargetKind, parentDigest: targetDigest,
		childKind: contractRunKind, childDigest: input.object.Digest(), targetDigest: targetDigest,
		attemptDigest: attemptDigest, startClaimDigest: claimDigest,
	}
	record, effect, err := persistContractRecord(ctx, store, input.object, relation, fault)
	return finalizedRunStorageRecord{record: record}, effect, err
}

func persistExecutionRecord(
	ctx context.Context,
	store *ObjectStore,
	input executionStorageInput,
	fault contractFault,
) (executionStorageRecord, contractEffect, error) {
	if store == nil || !input.run.validFor(store) || input.object.Kind() != contractExecutionKind {
		return executionStorageRecord{}, "", refuse(codeContractRecordRefused, "execution input lacks an exact run parent", nil)
	}
	targetDigest, runDigest, profileDigest, err := executionJoins(input.object)
	if err != nil || runDigest != input.run.record.object.Digest() ||
		targetDigest != input.run.record.relation.targetDigest || profileDigest != contractClassifierProfileDigest() {
		return executionStorageRecord{}, "", refuse(codeContractRecordRefused, "execution target, run, or classifier-profile join differs", err)
	}
	relation := contractRelation{
		relation: relationRunExecution, parentKind: contractRunKind, parentDigest: runDigest,
		secondaryParentKind: contractProfileKind, secondaryParentDigest: profileDigest,
		childKind: contractExecutionKind, childDigest: input.object.Digest(), targetDigest: targetDigest,
		classifierProfileDigest: profileDigest,
	}
	record, effect, err := persistContractRecord(ctx, store, input.object, relation, fault)
	return executionStorageRecord{record: record}, effect, err
}

func persistContractRecord(
	ctx context.Context,
	store *ObjectStore,
	object SemanticObject,
	relation contractRelation,
	fault contractFault,
) (contractStorageRecord, contractEffect, error) {
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return contractStorageRecord{}, "", err
	}
	if !object.Valid() || relation.childKind != object.Kind() || relation.childDigest != object.Digest() {
		return contractStorageRecord{}, "", refuse(codeContractRecordRefused, "child object and relationship disagree", nil)
	}
	if err := validateContractRelation(relation, object); err != nil {
		return contractStorageRecord{}, "", err
	}
	relationBytes, relationPath, err := contractRelationMaterial(store, relation)
	if err != nil {
		return contractStorageRecord{}, "", err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, "", err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return contractStorageRecord{}, "", refuse(codeContractRecordRefused, "contract namespace lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, "", err
	}
	publicationRoot, err := store.ensureContractDirectoryLocked(store.contractRoot, contractPublicationDirectory)
	if err != nil {
		return contractStorageRecord{}, "", err
	}
	publicationDirectory, err := store.ensureContractDirectoryLocked(publicationRoot, object.Kind())
	if err != nil {
		return contractStorageRecord{}, "", err
	}
	authority, err := store.publishLocked(ctx, object)
	if err != nil {
		return contractStorageRecord{}, contractAmbiguous,
			mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	objectPath, _, err := store.existingObjectPath(object.Digest())
	if err != nil {
		return contractStorageRecord{}, contractAmbiguous,
			mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	hex, _ := strictDigestHex(object.Digest())
	witnessPath := filepath.Join(publicationDirectory, hex)
	_, err = createExactHardLink(objectPath, witnessPath, object.CanonicalBytes(), fault)
	if err != nil {
		return contractStorageRecord{}, contractAmbiguous,
			mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	linkDirectory, err := store.relationDirectoryLocked(relation.relation)
	if err != nil {
		return contractStorageRecord{}, contractAmbiguous,
			mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if filepath.Dir(relationPath) != linkDirectory {
		return contractStorageRecord{}, contractAmbiguous, mutationFailure(
			codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen,
			refuse(codeContractRecordRefused, "relationship path escaped its typed directory", nil),
		)
	}
	_, err = createExactPrivateFile(linkDirectory, filepath.Base(relationPath), relationBytes, fault)
	if err != nil {
		return contractStorageRecord{}, contractAmbiguous,
			mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if err := reopenContractWitness(objectPath, witnessPath, object.CanonicalBytes()); err != nil {
		return contractStorageRecord{}, contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if body, err := readExactPrivateFile(relationPath, int64(len(relationBytes))); err != nil || !bytes.Equal(body, relationBytes) {
		return contractStorageRecord{}, contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	return contractStorageRecord{
		storeInstance: store.instance, object: object, authority: authority, witnessPath: witnessPath,
		relationPath: relationPath, relation: relation, relationBytes: relationBytes,
	}, contractExactConverged, nil
}

func openTargetByAttempt(ctx context.Context, store *ObjectStore, attempt attemptStorageRecord) (targetStorageRecord, error) {
	if store == nil || !attempt.validFor(store) {
		return targetStorageRecord{}, refuse(codeContractRecordRefused, "attempt parent is invalid or foreign", nil)
	}
	key := contractRelation{relation: relationAttemptTarget, parentKind: contractAttemptKind, parentDigest: attempt.digest}
	record, err := openContractRecordByParent(ctx, store, key, contractTargetKind)
	if err != nil || record.relation.attemptDigest != attempt.digest {
		return targetStorageRecord{}, refuse(codeContractRecordRefused, "target relation does not join the attempt", err)
	}
	return targetStorageRecord{record: record}, nil
}

func openFinalizedRunByTarget(ctx context.Context, store *ObjectStore, target targetStorageRecord) (finalizedRunStorageRecord, error) {
	if store == nil || !target.validFor(store) {
		return finalizedRunStorageRecord{}, refuse(codeContractRecordRefused, "target parent is invalid or foreign", nil)
	}
	key := contractRelation{relation: relationTargetRun, parentKind: contractTargetKind, parentDigest: target.record.object.Digest()}
	record, err := openContractRecordByParent(ctx, store, key, contractRunKind)
	if err != nil || record.relation.targetDigest != target.record.object.Digest() ||
		record.relation.attemptDigest != target.record.relation.attemptDigest {
		return finalizedRunStorageRecord{}, refuse(codeContractRecordRefused, "run relation does not join the target", err)
	}
	return finalizedRunStorageRecord{record: record}, nil
}

func openExecutionByRunProfile(ctx context.Context, store *ObjectStore, run finalizedRunStorageRecord) (executionStorageRecord, error) {
	if store == nil || !run.validFor(store) {
		return executionStorageRecord{}, refuse(codeContractRecordRefused, "run parent is invalid or foreign", nil)
	}
	profile := contractClassifierProfileDigest()
	key := contractRelation{
		relation: relationRunExecution, parentKind: contractRunKind, parentDigest: run.record.object.Digest(),
		secondaryParentKind: contractProfileKind, secondaryParentDigest: profile,
	}
	record, err := openContractRecordByParent(ctx, store, key, contractExecutionKind)
	if err != nil || record.relation.classifierProfileDigest != profile ||
		record.relation.targetDigest != run.record.relation.targetDigest {
		return executionStorageRecord{}, refuse(codeContractRecordRefused, "execution relation does not join run and profile", err)
	}
	return executionStorageRecord{record: record}, nil
}

func openContractRecordByParent(
	ctx context.Context,
	store *ObjectStore,
	key contractRelation,
	wantKind string,
) (contractStorageRecord, error) {
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return contractStorageRecord{}, err
	}
	_, relationPath, err := contractRelationMaterial(store, key)
	if err != nil {
		return contractStorageRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return contractStorageRecord{}, refuse(codeContractRecordRefused, "contract namespace lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, err
	}
	relationDirectory, err := store.relationDirectoryLocked(key.relation)
	if err != nil || relationDirectory != filepath.Dir(relationPath) {
		return contractStorageRecord{}, refuse(
			codeContractRecordRefused, "typed relationship parent directory is invalid or changed", err,
		)
	}
	body, err := readExactPrivateFile(relationPath, -1)
	if err != nil {
		return contractStorageRecord{}, refuse(codeContractRecordRefused, "typed relationship is absent or invalid", err)
	}
	relation, err := parseContractRelation(body)
	if err != nil || relation.relation != key.relation || relation.parentKind != key.parentKind ||
		relation.parentDigest != key.parentDigest || relation.secondaryParentKind != key.secondaryParentKind ||
		relation.secondaryParentDigest != key.secondaryParentDigest || relation.childKind != wantKind {
		return contractStorageRecord{}, refuse(codeContractRecordRefused, "typed relationship key or child differs", err)
	}
	expected, expectedPath, err := contractRelationMaterial(store, relation)
	if err != nil || expectedPath != relationPath || !bytes.Equal(body, expected) {
		return contractStorageRecord{}, refuse(codeContractRecordRefused, "typed relationship is not exact canonical material", err)
	}
	if effect, convergeErr := createExactPrivateFile(
		relationDirectory, filepath.Base(relationPath), body, nil,
	); convergeErr != nil || effect != contractExactConverged {
		return contractStorageRecord{}, refuse(
			codeContractRecordAmbiguous, "visible typed relationship did not converge durably", convergeErr,
		)
	}
	object, err := store.readObjectReference(relation.childKind, relation.childDigest)
	if err != nil {
		return contractStorageRecord{}, err
	}
	if err := validateContractRelation(relation, object); err != nil {
		return contractStorageRecord{}, err
	}
	publicationRoot, err := store.ensureContractDirectoryLocked(store.contractRoot, contractPublicationDirectory)
	if err != nil {
		return contractStorageRecord{}, err
	}
	publicationDirectory, err := store.ensureContractDirectoryLocked(publicationRoot, relation.childKind)
	if err != nil {
		return contractStorageRecord{}, err
	}
	hex, _ := strictDigestHex(relation.childDigest)
	witnessPath := filepath.Join(publicationDirectory, hex)
	objectPath, _, err := store.existingObjectPath(relation.childDigest)
	if err != nil || reopenContractWitness(objectPath, witnessPath, object.CanonicalBytes()) != nil {
		return contractStorageRecord{}, refuse(codeContractRecordRefused, "typed publication witness is absent or changed", err)
	}
	if err := store.assertReady(); err != nil {
		return contractStorageRecord{}, err
	}
	return contractStorageRecord{
		storeInstance: store.instance, object: object, authority: store.authority(object), witnessPath: witnessPath,
		relationPath: relationPath, relation: relation, relationBytes: body,
	}, nil
}

func (record targetStorageRecord) validFor(store *ObjectStore) bool {
	if store == nil || store.instance == nil {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return record.validForLocked(store)
}

func (record finalizedRunStorageRecord) validFor(store *ObjectStore) bool {
	if store == nil || store.instance == nil {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return record.validForLocked(store)
}

func (record executionStorageRecord) validFor(store *ObjectStore) bool {
	if store == nil || store.instance == nil {
		return false
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	return record.validForLocked(store)
}

func (record targetStorageRecord) validForLocked(store *ObjectStore) bool {
	return record.record.validForLocked(store) && record.record.relation.relation == relationAttemptTarget
}

func (record finalizedRunStorageRecord) validForLocked(store *ObjectStore) bool {
	return record.record.validForLocked(store) && record.record.relation.relation == relationTargetRun
}

func (record executionStorageRecord) validForLocked(store *ObjectStore) bool {
	return record.record.validForLocked(store) && record.record.relation.relation == relationRunExecution
}

func (record contractStorageRecord) validForLocked(store *ObjectStore) bool {
	if store == nil || record.storeInstance != store.instance || !record.object.Valid() ||
		record.authority.storeInstance != store.instance || !sameSemanticObject(record.authority.object, record.object) {
		return false
	}
	if err := store.assertReady(); err != nil {
		return false
	}
	relationDirectory, err := store.relationDirectoryLocked(record.relation.relation)
	if err != nil || relationDirectory != filepath.Dir(record.relationPath) {
		return false
	}
	publicationRoot, err := store.ensureContractDirectoryLocked(store.contractRoot, contractPublicationDirectory)
	if err != nil {
		return false
	}
	publicationDirectory, err := store.ensureContractDirectoryLocked(publicationRoot, record.object.Kind())
	if err != nil || publicationDirectory != filepath.Dir(record.witnessPath) {
		return false
	}
	expectedRelation, expectedRelationPath, err := contractRelationMaterial(store, record.relation)
	if err != nil || expectedRelationPath != record.relationPath || !bytes.Equal(expectedRelation, record.relationBytes) {
		return false
	}
	hex, err := strictDigestHex(record.object.Digest())
	if err != nil || filepath.Join(publicationDirectory, hex) != record.witnessPath {
		return false
	}
	objectPath, _, err := store.existingObjectPath(record.object.Digest())
	if err != nil || reopenContractWitness(objectPath, record.witnessPath, record.object.CanonicalBytes()) != nil {
		return false
	}
	body, err := readExactPrivateFile(record.relationPath, int64(len(record.relationBytes)))
	return err == nil && bytes.Equal(body, record.relationBytes) &&
		validateContractRelation(record.relation, record.object) == nil && store.assertReady() == nil
}

func contractRelationMaterial(store *ObjectStore, relation contractRelation) ([]byte, string, error) {
	keyWire := map[string]any{
		"schema_version": domain.SchemaVersion, "kind": "ContractStorageRelationKey",
		"relation_version": relationVersionV1, "relation": relation.relation,
		"parent_kind": relation.parentKind, "parent_digest": relation.parentDigest.String(),
		"secondary_parent_kind":   relation.secondaryParentKind,
		"secondary_parent_digest": relation.secondaryParentDigest.String(),
	}
	keyDigest, _, err := canon.DigestTyped("ContractStorageRelationKey", keyWire)
	if err != nil {
		return nil, "", refuse(codeContractRecordRefused, "relationship key cannot be canonicalized", err)
	}
	parsedKey, err := domain.ParseDigest(keyDigest.String())
	if err != nil {
		return nil, "", err
	}
	hex, _ := strictDigestHex(parsedKey)
	var directory string
	switch relation.relation {
	case relationAttemptTarget:
		directory = filepath.Join(store.contractLinks, targetLinkDirectory)
	case relationTargetRun:
		directory = filepath.Join(store.contractLinks, runLinkDirectory)
	case relationRunExecution:
		directory = filepath.Join(store.contractLinks, executionLinkDirectory)
	default:
		return nil, "", refuse(codeContractRecordRefused, "unknown relationship", nil)
	}
	if !relation.childDigest.Valid() {
		return nil, filepath.Join(directory, hex), nil
	}
	body, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": "ContractStorageRelation",
		"relation_version": relationVersionV1, "relation": relation.relation,
		"parent_kind": relation.parentKind, "parent_digest": relation.parentDigest.String(),
		"secondary_parent_kind":   relation.secondaryParentKind,
		"secondary_parent_digest": relation.secondaryParentDigest.String(),
		"child_kind":              relation.childKind, "child_digest": relation.childDigest.String(),
		"target_digest": relation.targetDigest.String(), "attempt_digest": relation.attemptDigest.String(),
		"start_claim_digest":        relation.startClaimDigest.String(),
		"classifier_profile_digest": relation.classifierProfileDigest.String(),
	})
	if err != nil {
		return nil, "", refuse(codeContractRecordRefused, "relationship body cannot be canonicalized", err)
	}
	return body, filepath.Join(directory, hex), nil
}

func parseContractRelation(body []byte) (contractRelation, error) {
	value, err := canon.Parse(body)
	if err != nil {
		return contractRelation{}, err
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return contractRelation{}, errors.Join(err, errors.New("relationship is not exact canonical JSON"))
	}
	if !contractRoster(value,
		"schema_version", "kind", "relation_version", "relation", "parent_kind", "parent_digest",
		"secondary_parent_kind", "secondary_parent_digest", "child_kind", "child_digest",
		"target_digest", "attempt_digest", "start_claim_digest", "classifier_profile_digest",
	) {
		return contractRelation{}, errors.New("relationship root roster differs")
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
	parseOptional := func(name string) (domain.Digest, error) {
		raw, memberErr := text(name)
		if memberErr != nil {
			return "", memberErr
		}
		if raw == "" {
			return "", nil
		}
		return domain.ParseDigest(raw)
	}
	schema, err := text("schema_version")
	if err != nil {
		return contractRelation{}, err
	}
	kind, err := text("kind")
	if err != nil {
		return contractRelation{}, err
	}
	version, err := text("relation_version")
	if err != nil {
		return contractRelation{}, err
	}
	if schema != domain.SchemaVersion || kind != "ContractStorageRelation" || version != relationVersionV1 {
		return contractRelation{}, errors.New("relationship constants differ")
	}
	relation := contractRelation{}
	relation.relation, err = text("relation")
	if err != nil {
		return contractRelation{}, err
	}
	if relation.parentKind, err = text("parent_kind"); err != nil {
		return contractRelation{}, err
	}
	if relation.parentDigest, err = parseOptional("parent_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.secondaryParentKind, err = text("secondary_parent_kind"); err != nil {
		return contractRelation{}, err
	}
	if relation.secondaryParentDigest, err = parseOptional("secondary_parent_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.childKind, err = text("child_kind"); err != nil {
		return contractRelation{}, err
	}
	if relation.childDigest, err = parseOptional("child_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.targetDigest, err = parseOptional("target_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.attemptDigest, err = parseOptional("attempt_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.startClaimDigest, err = parseOptional("start_claim_digest"); err != nil {
		return contractRelation{}, err
	}
	if relation.classifierProfileDigest, err = parseOptional("classifier_profile_digest"); err != nil {
		return contractRelation{}, err
	}
	if !relation.parentDigest.Valid() || !relation.childDigest.Valid() || !contractRelationAlgebraValid(relation) {
		return contractRelation{}, errors.New("relationship typed algebra differs")
	}
	return relation, nil
}

func contractRoster(value canon.Value, names ...string) bool {
	members, ok := value.Members()
	if !ok || len(members) != len(names) {
		return false
	}
	expected := append([]string(nil), names...)
	sort.Strings(expected)
	for index, member := range members {
		if member.Name != expected[index] {
			return false
		}
	}
	return true
}

func contractRelationAlgebraValid(relation contractRelation) bool {
	switch relation.relation {
	case relationAttemptTarget:
		return relation.parentKind == contractAttemptKind && relation.parentDigest.Valid() &&
			relation.secondaryParentKind == "" && !relation.secondaryParentDigest.Valid() &&
			relation.childKind == contractTargetKind && relation.childDigest.Valid() &&
			relation.targetDigest == relation.childDigest && relation.attemptDigest == relation.parentDigest &&
			!relation.startClaimDigest.Valid() && !relation.classifierProfileDigest.Valid()
	case relationTargetRun:
		return relation.parentKind == contractTargetKind && relation.parentDigest.Valid() &&
			relation.secondaryParentKind == "" && !relation.secondaryParentDigest.Valid() &&
			relation.childKind == contractRunKind && relation.childDigest.Valid() &&
			relation.targetDigest == relation.parentDigest && relation.attemptDigest.Valid() &&
			relation.startClaimDigest.Valid() && !relation.classifierProfileDigest.Valid()
	case relationRunExecution:
		return relation.parentKind == contractRunKind && relation.parentDigest.Valid() &&
			relation.secondaryParentKind == contractProfileKind &&
			relation.secondaryParentDigest == contractClassifierProfileDigest() &&
			relation.childKind == contractExecutionKind && relation.childDigest.Valid() &&
			relation.targetDigest.Valid() && !relation.attemptDigest.Valid() &&
			!relation.startClaimDigest.Valid() &&
			relation.classifierProfileDigest == relation.secondaryParentDigest
	default:
		return false
	}
}

func validateContractRelation(relation contractRelation, object SemanticObject) error {
	if !object.Valid() || !contractRelationAlgebraValid(relation) || relation.childKind != object.Kind() ||
		relation.childDigest != object.Digest() {
		return refuse(codeContractRecordRefused, "relationship algebra or child identity differs", nil)
	}
	switch relation.relation {
	case relationAttemptTarget:
		attempt, err := targetAttemptDigest(object)
		if err != nil || attempt != relation.attemptDigest {
			return refuse(codeContractRecordRefused, "target object attempt join differs", err)
		}
	case relationTargetRun:
		target, attempt, claim, _, err := finalizedRunJoins(object)
		if err != nil || target != relation.targetDigest || attempt != relation.attemptDigest || claim != relation.startClaimDigest {
			return refuse(codeContractRecordRefused, "run object relationship joins differ", err)
		}
	case relationRunExecution:
		target, run, profile, err := executionJoins(object)
		if err != nil || target != relation.targetDigest || run != relation.parentDigest ||
			profile != relation.classifierProfileDigest {
			return refuse(codeContractRecordRefused, "execution object relationship joins differ", err)
		}
	}
	return nil
}

func (store *ObjectStore) ensureContractDirectoryLocked(parent, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return "", refuse(codeContractRecordRefused, "invalid contract directory component", nil)
	}
	if err := rejectCaseAlias(parent, name); err != nil {
		return "", refuse(codeContractRecordRefused, "contract directory aliases an existing name", err)
	}
	path := filepath.Join(parent, name)
	info, created, err := ensurePrivateDirectory(path)
	if err != nil {
		return "", refuse(codeContractRecordRefused, "contract directory is invalid", err)
	}
	if created {
		if err := syncDirectory(parent); err != nil {
			return "", refuse(codeContractRecordAmbiguous, "contract-directory parent sync failed", err)
		}
	}
	if err := rejectCaseAlias(parent, name); err != nil {
		return "", refuse(codeContractRecordRefused, "contract directory changed spelling", err)
	}
	if retained, ok := store.contractInfos[path]; ok && !os.SameFile(retained, info) {
		return "", refuse(codeContractRecordRefused, "contract directory changed identity", nil)
	}
	store.contractInfos[path] = info
	return path, nil
}

func (store *ObjectStore) relationDirectoryLocked(relation string) (string, error) {
	var name string
	switch relation {
	case relationAttemptTarget:
		name = targetLinkDirectory
	case relationTargetRun:
		name = runLinkDirectory
	case relationRunExecution:
		name = executionLinkDirectory
	default:
		return "", refuse(codeContractRecordRefused, "unknown relationship directory", nil)
	}
	return store.ensureContractDirectoryLocked(store.contractLinks, name)
}

func createExactHardLink(source, destination string, body []byte, fault contractFault) (contractEffect, error) {
	if err := rejectPathCaseAlias(destination); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, err)
	}
	if err := injectContractFault(fault, faultBeforeLink); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, err)
	}
	if err := os.Link(source, destination); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, err)
		}
		if reopenContractWitness(source, destination, body) != nil {
			return contractAmbiguous, mutationFailure(codeContractRecordConflict, contractAmbiguous, faultBeforeLink, err)
		}
		if syncErr := syncDirectory(filepath.Dir(destination)); syncErr != nil {
			return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, syncErr)
		}
		if reopenContractWitness(source, destination, body) != nil {
			return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, errors.New("existing witness did not reopen after parent sync"))
		}
		return contractExactConverged, nil
	}
	if err := injectContractFault(fault, faultAfterLink); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterLink, err)
	}
	if err := syncDirectory(filepath.Dir(destination)); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, err)
	}
	if err := injectContractFault(fault, faultAfterParentSync); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, err)
	}
	return contractExactConverged, nil
}

func createExactPrivateFile(directory, name string, body []byte, fault contractFault) (contractEffect, error) {
	if name == "" || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeTemporary, errors.New("invalid private file component"))
	}
	if err := rejectCaseAlias(directory, name); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeTemporary, err)
	}
	destination := filepath.Join(directory, name)
	if existing, err := readExactPrivateFile(destination, -1); err == nil {
		if bytes.Equal(existing, body) {
			if syncErr := syncDirectory(directory); syncErr != nil {
				return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, syncErr)
			}
			reopened, reopenErr := readExactPrivateFile(destination, int64(len(body)))
			if reopenErr != nil || !bytes.Equal(reopened, body) {
				return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, reopenErr)
			}
			return contractExactConverged, nil
		}
		return contractAmbiguous, mutationFailure(codeContractRecordConflict, contractAmbiguous, faultBeforeTemporary, errors.New("alternate exact-key bytes exist"))
	} else if !errors.Is(err, os.ErrNotExist) {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeTemporary, err)
	}
	if err := injectContractFault(fault, faultBeforeTemporary); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeTemporary, err)
	}
	temporary, err := writeSyncedTemporary(directory, ".contract-", body)
	if err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeTemporary, err)
	}
	defer os.Remove(temporary)
	if err := injectContractFault(fault, faultAfterTempSync); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultAfterTempSync, err)
	}
	if err := injectContractFault(fault, faultBeforeLink); err != nil {
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, err)
	}
	if err := os.Link(temporary, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			existing, readErr := readExactPrivateFile(destination, -1)
			if readErr == nil && bytes.Equal(existing, body) {
				if syncErr := syncDirectory(directory); syncErr != nil {
					return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, syncErr)
				}
				reopened, reopenErr := readExactPrivateFile(destination, int64(len(body)))
				if reopenErr != nil || !bytes.Equal(reopened, body) {
					return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, reopenErr)
				}
				return contractExactConverged, nil
			}
			return contractAmbiguous, mutationFailure(codeContractRecordConflict, contractAmbiguous, faultBeforeLink, errors.Join(err, readErr))
		}
		return contractKnownNoEffect, mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, err)
	}
	if err := injectContractFault(fault, faultAfterLink); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterLink, err)
	}
	if err := syncDirectory(directory); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, err)
	}
	if err := injectContractFault(fault, faultAfterParentSync); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultAfterParentSync, err)
	}
	if err := injectContractFault(fault, faultBeforeReopen); err != nil {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	reopened, err := readExactPrivateFile(destination, int64(len(body)))
	if err != nil || !bytes.Equal(reopened, body) {
		return contractAmbiguous, mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	return contractExactConverged, nil
}

func reopenContractWitness(source, witness string, body []byte) error {
	sourceInfo, err := exactObjectInfo(source, int64(len(body)))
	if err != nil {
		return err
	}
	witnessInfo, err := exactObjectInfo(witness, int64(len(body)))
	if err != nil || !os.SameFile(sourceInfo, witnessInfo) {
		return errors.Join(err, errors.New("typed witness is not the exact object inode"))
	}
	reopened, err := readExactPrivateFile(witness, int64(len(body)))
	if err != nil || !bytes.Equal(reopened, body) {
		return errors.Join(err, errors.New("typed witness bytes differ"))
	}
	return nil
}

func readExactPrivateFile(path string, expected int64) ([]byte, error) {
	if err := rejectPathCaseAlias(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() < 1 || info.Size() > int64(canon.MaxInputBytes) || (expected >= 0 && info.Size() != expected) {
		return nil, errors.New("private file facts are invalid")
	}
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := handle.Stat()
	body, readErr := io.ReadAll(io.LimitReader(handle, int64(canon.MaxInputBytes)+1))
	closeErr := handle.Close()
	after, afterErr := os.Lstat(path)
	aliasErr := rejectPathCaseAlias(path)
	if statErr != nil || readErr != nil || closeErr != nil || afterErr != nil || !os.SameFile(info, opened) ||
		!os.SameFile(opened, after) || int64(len(body)) != info.Size() || aliasErr != nil {
		return nil, errors.Join(statErr, readErr, closeErr, afterErr, aliasErr)
	}
	return body, nil
}

func targetAttemptDigest(object SemanticObject) (domain.Digest, error) {
	return nestedDigest(object, "attempt_binding", "attempt_artifact_digest")
}

func targetBootDigest(object SemanticObject) (domain.Digest, error) {
	return nestedDigest(object, "boot_session_binding", "identity_digest")
}

func finalizedRunJoins(object SemanticObject) (domain.Digest, domain.Digest, domain.Digest, domain.Digest, error) {
	target, err := topDigest(object, "contract_execution_target_digest")
	if err != nil {
		return "", "", "", "", err
	}
	attempt, err := topDigest(object, "attempt_artifact_digest")
	if err != nil {
		return "", "", "", "", err
	}
	claim, err := nestedTypedDigest(object, "start_claim_ref", contractClaimKind)
	if err != nil {
		return "", "", "", "", err
	}
	manifest, err := deepTypedDigest(object, []string{"closed_run_witness", "private_evidence", "manifest_ref"}, "PRIVATE_EVIDENCE_MANIFEST")
	return target, attempt, claim, manifest, err
}

func executionJoins(object SemanticObject) (domain.Digest, domain.Digest, domain.Digest, error) {
	target, err := topDigest(object, "contract_execution_target_digest")
	if err != nil {
		return "", "", "", err
	}
	run, err := topDigest(object, "finalized_contract_run_digest")
	if err != nil {
		return "", "", "", err
	}
	value, err := canon.Parse(object.CanonicalBytes())
	if err != nil {
		return "", "", "", err
	}
	profile, ok := value.LookupMember("classifier_profile")
	text, textOK := profile.Text()
	if !ok || !textOK || text != contractProfileV1 {
		return "", "", "", errors.New("classifier profile differs")
	}
	return target, run, contractClassifierProfileDigest(), nil
}

func contractClassifierProfileDigest() domain.Digest {
	digest, _, err := canon.DigestTyped(contractProfileKind, map[string]any{
		"schema_version": domain.SchemaVersion, "kind": contractProfileKind, "profile": contractProfileV1,
	})
	if err != nil {
		return ""
	}
	parsed, _ := domain.ParseDigest(digest.String())
	return parsed
}

func topDigest(object SemanticObject, name string) (domain.Digest, error) {
	value, err := canon.Parse(object.CanonicalBytes())
	if err != nil {
		return "", err
	}
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

func nestedDigest(object SemanticObject, parent, name string) (domain.Digest, error) {
	value, err := canon.Parse(object.CanonicalBytes())
	if err != nil {
		return "", err
	}
	nested, ok := value.LookupMember(parent)
	if !ok {
		return "", fmt.Errorf("missing %s", parent)
	}
	member, ok := nested.LookupMember(name)
	if !ok {
		return "", fmt.Errorf("missing %s.%s", parent, name)
	}
	raw, ok := member.Text()
	if !ok {
		return "", fmt.Errorf("%s.%s is not text", parent, name)
	}
	return domain.ParseDigest(raw)
}

func nestedTypedDigest(object SemanticObject, parent, kind string) (domain.Digest, error) {
	return deepTypedDigest(object, []string{parent}, kind)
}

func deepTypedDigest(object SemanticObject, parents []string, kind string) (domain.Digest, error) {
	value, err := canon.Parse(object.CanonicalBytes())
	if err != nil {
		return "", err
	}
	for _, parent := range parents {
		value, _ = value.LookupMember(parent)
	}
	kindValue, kindOK := value.LookupMember("kind")
	digestValue, digestOK := value.LookupMember("digest")
	actualKind, kindText := kindValue.Text()
	raw, digestText := digestValue.Text()
	if !kindOK || !digestOK || !kindText || !digestText || actualKind != kind {
		return "", errors.New("typed digest reference differs")
	}
	return domain.ParseDigest(raw)
}
