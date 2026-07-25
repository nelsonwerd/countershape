package store

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
)

const (
	spawnObservationDirectory = "spawn-observations"
	spawnObservationKind      = "ParentSpawnObservation"
	spawnObservationVersion   = "parent-spawn-observation/v1"
)

type contractRunOwnerPhase uint8

const (
	contractRunOwnerAcquired contractRunOwnerPhase = iota + 1
	contractRunOwnerStartConsumed
	contractRunOwnerSpawnObserved
	contractRunOwnerManifestPersisted
	contractRunOwnerFinalized
)

type spawnObservationRecord struct {
	storeInstance *objectStoreInstance
	targetDigest  domain.Digest
	attemptDigest domain.Digest
	claimDigest   domain.Digest
	bindingDigest domain.Digest
	observation   contractmodel.SpawnObservation
	digest        domain.Digest
	canonical     []byte
	path          string
}

type contractRunOwnerState struct {
	mu       sync.Mutex
	store    *ObjectStore
	target   targetStorageRecord
	claim    startClaimRecord
	winner   startClaimWinner
	phase    contractRunOwnerPhase
	binding  domain.Digest
	spawn    spawnObservationRecord
	manifest privateManifestRecord
	closure  *terminalClosureState
}

// ContractRunOwner is the one-shot private authority created by exact target
// admission. Copies share one consumed state and cannot multiply start rights.
type ContractRunOwner struct {
	state *contractRunOwnerState
}

// PrivateRunManifest is an inert handle to exact retained private evidence.
type PrivateRunManifest struct {
	store  *ObjectStore
	record privateManifestRecord
}

// FinalizedRunRecord is an inert, exactly reopened finalized-run value.
type FinalizedRunRecord struct {
	store   *ObjectStore
	target  ContractTargetRecord
	record  finalizedRunStorageRecord
	model   contractmodel.FinalizedContractRun
	release *finalizedRunReleaseSeal
}

type finalizedRunReleaseSeal struct {
	mu       sync.Mutex
	marker   byte
	released bool
}

type terminalClosureState struct {
	store    *ObjectStore
	target   ContractTargetRecord
	claim    startClaimRecord
	spawn    spawnObservationRecord
	manifest PrivateRunManifest
	run      FinalizedRunRecord
	record   terminalClosureRecord
}

// TerminalClosure is the exact durable proof required to clear the execution
// interlock. It cannot be synthesized from public digests or semantic bytes.
type TerminalClosure struct {
	state *terminalClosureState
}

// ContractExecutionRecord is an inert, exactly converged classification
// record. It carries no authority to start or retry a process.
type ContractExecutionRecord struct {
	store  *ObjectStore
	run    FinalizedRunRecord
	record executionStorageRecord
	model  contractmodel.ContractExecution
}

// AcquireContractRunOwner admits the exact store-bound target under the
// measured boot epoch. There is deliberately no owner reopen or digest-only
// acquisition route.
func AcquireContractRunOwner(
	ctx context.Context,
	target ContractTargetRecord,
	epoch hostepoch.Epoch,
) (ContractRunOwner, error) {
	if !target.Valid() || !epoch.Valid() {
		return ContractRunOwner{}, refuse(codeInterlockRefused, "exact target record and measured host epoch are required", nil)
	}
	store := target.store
	_, claim, winner, err := acquireInterlockAndStartClaim(
		ctx, store, target.record, epoch.Digest(), nil,
	)
	if err != nil {
		return ContractRunOwner{}, err
	}
	return ContractRunOwner{state: &contractRunOwnerState{
		store: store, target: target.record, claim: claim, winner: winner,
		phase: contractRunOwnerAcquired,
	}}, nil
}

// ConsumeForStart burns the private start authority. The runner calls it only
// after preparing the immutable invocation and immediately before Start.
func (owner ContractRunOwner) ConsumeForStart(ctx context.Context, bindingDigest domain.Digest) error {
	state := owner.state
	if state == nil {
		return refuse(codeInterlockRefused, "contract-run owner is absent", nil)
	}
	if !bindingDigest.Valid() {
		return refuse(codeInterlockRefused, "prepared invocation binding is invalid", nil)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.phase != contractRunOwnerAcquired {
		return refuse(codeInterlockRefused, "start authority is unavailable or already consumed", nil)
	}
	if err := consumeStartClaimWinner(ctx, state.store, &state.winner); err != nil {
		return err
	}
	state.binding = bindingDigest
	state.phase = contractRunOwnerStartConsumed
	return nil
}

func (owner ContractRunOwner) StartClaimDigest() domain.Digest {
	state := owner.state
	if state == nil {
		return ""
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.phase < contractRunOwnerAcquired || !state.claim.digest.Valid() {
		return ""
	}
	return state.claim.digest
}

// PersistSpawnObservation durably binds the parent-observed start outcome to
// the exact prepared invocation before any finalized run can be published.
func (owner ContractRunOwner) PersistSpawnObservation(
	ctx context.Context,
	observation contractmodel.SpawnObservation,
) error {
	state := owner.state
	if state == nil || !observation.Valid() {
		return refuse(codeContractRecordRefused, "spawn observation inputs are invalid", nil)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.phase < contractRunOwnerStartConsumed {
		return refuse(codeContractRecordRefused, "start authority has not been consumed", nil)
	}
	if state.phase > contractRunOwnerStartConsumed {
		if sameSpawnObservation(state.spawn.observation, observation) && state.spawn.bindingDigest == state.binding {
			durable, err := openSpawnObservation(ctx, state.store, state.target, state.claim)
			if err == nil && durable.bindingDigest == state.binding &&
				sameSpawnObservation(durable.observation, observation) {
				state.spawn = durable
				return nil
			}
			return refuse(codeContractRecordRefused, "durable spawn observation changed after persistence", err)
		}
		return refuse(codeContractRecordRefused, "owner is already bound to another spawn observation", nil)
	}
	record, err := persistSpawnObservation(
		ctx, state.store, state.target, state.claim, observation, state.binding,
	)
	if err != nil {
		return err
	}
	state.spawn = record
	state.phase = contractRunOwnerSpawnObserved
	return nil
}

// PersistPrivateRunManifest writes a bounded set of private evidence bodies,
// deriving every typed reference digest inside the store. Identical bodies are
// grouped deterministically. The returned value exposes identity, not bytes.
func (owner ContractRunOwner) PersistPrivateRunManifest(
	ctx context.Context,
	evidence map[contractmodel.EvidenceKind][]byte,
) (PrivateRunManifest, error) {
	blobs, err := privateBlobsFromEvidence(evidence)
	if err != nil {
		return PrivateRunManifest{}, err
	}
	state := owner.state
	if state == nil {
		return PrivateRunManifest{}, refuse(codePrivateEvidenceRefused, "contract-run owner is absent", nil)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.phase < contractRunOwnerSpawnObserved {
		return PrivateRunManifest{}, refuse(codePrivateEvidenceRefused, "durable spawn observation is required before private evidence", nil)
	}
	if state.phase > contractRunOwnerSpawnObserved {
		if !privateManifestMatchesBlobs(state.manifest, blobs) {
			return PrivateRunManifest{}, refuse(codePrivateEvidenceRefused, "owner is already bound to another private manifest", nil)
		}
		if err := validatePrivateManifestForFinalization(ctx, state.store, state.manifest); err != nil {
			return PrivateRunManifest{}, err
		}
		return PrivateRunManifest{store: state.store, record: state.manifest}, nil
	}
	record, _, createErr := createPrivateManifest(
		ctx, state.store, state.target, state.claim, blobs, nil,
	)
	if createErr != nil {
		return PrivateRunManifest{}, createErr
	}
	state.manifest = record
	state.phase = contractRunOwnerManifestPersisted
	return PrivateRunManifest{store: state.store, record: record}, nil
}

func privateBlobsFromEvidence(evidence map[contractmodel.EvidenceKind][]byte) ([]privateBlobInput, error) {
	if len(evidence) == 0 || len(evidence) > len(privateEvidenceKindOrder) {
		return nil, refuse(codePrivateEvidenceRefused, "private evidence roster is empty or exceeds the closed profile", nil)
	}
	type groupedBlob struct {
		body       []byte
		references []privateEvidenceReference
	}
	groups := make(map[string]groupedBlob, len(evidence))
	for kind, source := range evidence {
		if len(source) == 0 || kind == contractmodel.EvidencePrivateManifest {
			return nil, refuse(codePrivateEvidenceRefused, "private evidence body is empty or recursively typed", nil)
		}
		digest := privateBytesDigest(source)
		reference := privateEvidenceReference{kind: string(kind), digest: digest}
		if !reference.valid() {
			return nil, refuse(codePrivateEvidenceRefused, "private evidence kind is outside the closed store roster", nil)
		}
		key := digest.String()
		group, present := groups[key]
		if present && !bytes.Equal(group.body, source) {
			return nil, refuse(codePrivateEvidenceRefused, "private evidence digest collision is ambiguous", nil)
		}
		if !present {
			group.body = append([]byte(nil), source...)
		}
		group.references = append(group.references, reference)
		groups[key] = group
	}
	blobs := make([]privateBlobInput, 0, len(groups))
	for _, group := range groups {
		blobs = append(blobs, privateBlobInput{body: group.body, references: group.references})
	}
	normalized, _, err := normalizePrivateBlobs(blobs)
	return normalized, err
}

func privateManifestMatchesBlobs(record privateManifestRecord, blobs []privateBlobInput) bool {
	if len(record.entries) != len(blobs) {
		return false
	}
	for index, blob := range blobs {
		entry := record.entries[index]
		if entry.digest != privateBytesDigest(blob.body) || entry.count != int64(len(blob.body)) ||
			len(entry.references) != len(blob.references) {
			return false
		}
		for referenceIndex := range blob.references {
			if entry.references[referenceIndex] != blob.references[referenceIndex] {
				return false
			}
		}
	}
	return true
}

func (manifest PrivateRunManifest) Valid() bool {
	return manifest.store != nil && manifest.record.validFor(manifest.store)
}

func (manifest PrivateRunManifest) Summary() (contractmodel.PrivateManifestSummary, error) {
	if !manifest.Valid() {
		return contractmodel.PrivateManifestSummary{}, refuse(codePrivateEvidenceRefused, "private manifest is invalid", nil)
	}
	reference, err := contractmodel.NewPrivateEvidenceManifestRef(manifest.record.digest)
	if err != nil {
		return contractmodel.PrivateManifestSummary{}, err
	}
	return contractmodel.NewPrivateManifestSummary(
		reference, manifest.record.blobCount, manifest.record.aggregateBytes,
	)
}

// EvidenceRef returns the store-derived logical reference for one retained
// evidence body. Callers cannot choose or substitute its digest.
func (manifest PrivateRunManifest) EvidenceRef(kind contractmodel.EvidenceKind) (contractmodel.EvidenceRef, error) {
	if !manifest.Valid() || kind == contractmodel.EvidencePrivateManifest {
		return contractmodel.EvidenceRef{}, refuse(codePrivateEvidenceRefused, "private manifest or evidence kind is invalid", nil)
	}
	var digest domain.Digest
	for _, entry := range manifest.record.entries {
		for _, reference := range entry.references {
			if reference.kind == string(kind) {
				if digest.Valid() && digest != reference.digest {
					return contractmodel.EvidenceRef{}, refuse(codePrivateEvidenceRefused, "private evidence kind is ambiguous", nil)
				}
				digest = reference.digest
			}
		}
	}
	if !digest.Valid() {
		return contractmodel.EvidenceRef{}, refuse(codePrivateEvidenceRefused, "private evidence kind is absent", nil)
	}
	return modelEvidenceReference(kind, digest)
}

func modelEvidenceReference(kind contractmodel.EvidenceKind, digest domain.Digest) (contractmodel.EvidenceRef, error) {
	switch kind {
	case contractmodel.EvidenceMaterializationRevalidation:
		return contractmodel.NewMaterializationRevalidationRef(digest)
	case contractmodel.EvidenceRuntimeRevalidation:
		return contractmodel.NewRuntimeRevalidationRef(digest)
	case contractmodel.EvidenceProcessResult:
		return contractmodel.NewProcessResultRef(digest)
	case contractmodel.EvidenceWaitResult:
		return contractmodel.NewWaitResultRef(digest)
	case contractmodel.EvidenceDrainResult:
		return contractmodel.NewDrainResultRef(digest)
	case contractmodel.EvidenceTeardownResult:
		return contractmodel.NewTeardownResultRef(digest)
	case contractmodel.EvidenceOrphanCheck:
		return contractmodel.NewOrphanCheckRef(digest)
	case contractmodel.EvidenceFinalizationMarker:
		return contractmodel.NewFinalizationMarkerRef(digest)
	case contractmodel.EvidenceCapturedObservation:
		return contractmodel.NewCapturedObservationRef(digest)
	case contractmodel.EvidenceProjectionResult:
		return contractmodel.NewProjectionResultRef(digest)
	case contractmodel.EvidenceTargetInventory:
		return contractmodel.NewTargetInventoryRef(digest)
	case contractmodel.EvidenceChildBindings:
		return contractmodel.NewChildBindingsRef(digest)
	case contractmodel.EvidenceImportResolution:
		return contractmodel.NewImportResolutionRef(digest)
	case contractmodel.EvidenceServiceBindings:
		return contractmodel.NewServiceBindingsRef(digest)
	case contractmodel.EvidenceSentinelInheritance:
		return contractmodel.NewSentinelInheritanceRef(digest)
	default:
		return contractmodel.EvidenceRef{}, refuse(codePrivateEvidenceRefused, "private evidence kind is outside the model roster", nil)
	}
}

// PersistFinalizedRun publishes only a model value whose exact spawn witness,
// target, claim, and private manifest all join this consumed owner.
func (owner ContractRunOwner) PersistFinalizedRun(
	ctx context.Context,
	manifest PrivateRunManifest,
	run contractmodel.FinalizedContractRun,
) (TerminalClosure, error) {
	state := owner.state
	if state == nil {
		return TerminalClosure{}, refuse(codeContractRecordRefused, "contract-run owner is absent", nil)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.phase == contractRunOwnerFinalized {
		if state.closure != nil && state.closure.manifest.record.digest == manifest.record.digest && state.closure.run.model.Equal(run) {
			durableSpawn, spawnErr := openSpawnObservation(ctx, state.store, state.target, state.claim)
			durableRun, runErr := openFinalizedRunByTarget(ctx, state.store, state.target)
			manifestErr := validatePrivateManifestForFinalization(ctx, state.store, state.manifest)
			if spawnErr == nil && runErr == nil && manifestErr == nil &&
				durableSpawn.bindingDigest == state.binding &&
				sameSpawnObservation(durableSpawn.observation, state.spawn.observation) &&
				durableRun.record.object.Digest() == run.Digest() &&
				bytes.Equal(durableRun.record.object.CanonicalBytes(), run.CanonicalBytes()) {
				state.spawn = durableSpawn
				return TerminalClosure{state: state.closure}, nil
			}
			return TerminalClosure{}, refuse(
				codeContractRecordRefused, "durable finalized-run graph changed after persistence",
				errors.Join(spawnErr, runErr, manifestErr),
			)
		}
		return TerminalClosure{}, refuse(codeContractRecordRefused, "owner is already bound to another finalized run", nil)
	}
	if state.phase != contractRunOwnerManifestPersisted || manifest.store != state.store || !manifest.Valid() ||
		manifest.record.digest != state.manifest.digest || !run.Valid() ||
		run.TargetDigest() != state.target.record.object.Digest() ||
		run.AttemptArtifactDigest() != state.target.record.relation.attemptDigest ||
		run.StartClaimDigest() != state.claim.digest ||
		run.PrivateManifest().ManifestRef().Digest() != manifest.record.digest ||
		!sameSpawnObservation(run.Witness().SpawnObservation(), state.spawn.observation) {
		return TerminalClosure{}, refuse(codeContractRecordRefused, "owner, spawn observation, manifest, and finalized run do not join", nil)
	}
	durableSpawn, err := openSpawnObservation(ctx, state.store, state.target, state.claim)
	if err != nil || durableSpawn.bindingDigest != state.binding ||
		!sameSpawnObservation(durableSpawn.observation, state.spawn.observation) {
		return TerminalClosure{}, refuse(codeContractRecordRefused, "durable spawn observation changed before finalized publication", err)
	}
	object, err := finalizedRunSemanticObject(run)
	if err != nil {
		return TerminalClosure{}, err
	}
	_, _, persistErr := persistFinalizedRunRecord(
		ctx, state.store,
		runStorageInput{target: state.target, claim: state.claim, object: object, manifest: manifest.record},
		nil,
	)
	reopened, openErr := openFinalizedRunByTarget(ctx, state.store, state.target)
	if openErr != nil || reopened.record.object.Digest() != run.Digest() ||
		!bytes.Equal(reopened.record.object.CanonicalBytes(), run.CanonicalBytes()) {
		return TerminalClosure{}, refuse(
			codeContractRecordAmbiguous, "finalized-run publication did not reconcile exactly", errors.Join(persistErr, openErr),
		)
	}
	if persistErr != nil && openErr == nil {
		// An exact typed reopen is the recovery authority for an ambiguous
		// post-publication failure. The original error remains in didrun logs,
		// but does not make the durable relationship indeterminate.
	}
	targetModel, err := parseContractTargetStorageRecord(state.target)
	if err != nil {
		return TerminalClosure{}, err
	}
	parsedRun, err := parseFinalizedRunStorageRecord(reopened, targetModel)
	if err != nil || !parsedRun.Equal(run) {
		return TerminalClosure{}, refuse(codeContractRecordAmbiguous, "finalized run did not reconstruct from durable parents", err)
	}
	publicTarget := ContractTargetRecord{store: state.store, record: state.target}
	publicRun := FinalizedRunRecord{
		store: state.store, target: publicTarget, record: reopened, model: parsedRun,
		release: &finalizedRunReleaseSeal{marker: 1},
	}
	closureRecord := terminalClosureRecord{
		run: reopened, manifest: manifest.record, seal: &terminalClosureSeal{marker: 1},
	}
	closureState := &terminalClosureState{
		store: state.store, target: publicTarget, claim: state.claim, spawn: state.spawn,
		manifest: manifest, run: publicRun, record: closureRecord,
	}
	state.phase = contractRunOwnerFinalized
	state.closure = closureState
	return TerminalClosure{state: closureState}, nil
}

func (run FinalizedRunRecord) Valid() bool {
	return run.store != nil && run.target.store == run.store && run.target.Valid() &&
		run.record.validFor(run.store) && run.model.Valid() &&
		run.release != nil && run.release.marker == 1 &&
		run.model.Digest() == run.record.record.object.Digest() &&
		bytes.Equal(run.model.CanonicalBytes(), run.record.record.object.CanonicalBytes())
}

func (run FinalizedRunRecord) classificationReady() bool {
	if !run.Valid() {
		return false
	}
	run.release.mu.Lock()
	defer run.release.mu.Unlock()
	return run.release.marker == 1 && run.release.released
}

func (run FinalizedRunRecord) markReleased() {
	if run.release == nil {
		return
	}
	run.release.mu.Lock()
	defer run.release.mu.Unlock()
	if run.release.marker == 1 {
		run.release.released = true
	}
}

func (run FinalizedRunRecord) Digest() domain.Digest {
	if !run.Valid() {
		return ""
	}
	return run.model.Digest()
}

func (run FinalizedRunRecord) Model() contractmodel.FinalizedContractRun {
	if !run.Valid() {
		return contractmodel.FinalizedContractRun{}
	}
	return run.model
}

// OpenFinalizedRunRecord reconstructs the public run without opening private
// evidence. This is the classification-only restart path after durable FCR.
func OpenFinalizedRunRecord(
	ctx context.Context,
	target ContractTargetRecord,
	targetModel contractmodel.ContractExecutionTarget,
) (FinalizedRunRecord, error) {
	result, err := openFinalizedRunRecord(ctx, target, targetModel)
	if err != nil {
		return FinalizedRunRecord{}, err
	}
	claim, err := openStartClaim(ctx, target.store, target.record)
	if err != nil || result.model.StartClaimDigest() != claim.digest {
		return FinalizedRunRecord{}, refuse(codeContractRecordRefused, "finalized run and durable start claim differ", err)
	}
	if err := validateFinalizedRunRelease(ctx, target.store, claim, result.model.Digest()); err != nil {
		return FinalizedRunRecord{}, err
	}
	result.markReleased()
	return result, nil
}

func openFinalizedRunRecord(
	ctx context.Context,
	target ContractTargetRecord,
	targetModel contractmodel.ContractExecutionTarget,
) (FinalizedRunRecord, error) {
	if !target.Valid() || !targetModel.Valid() || targetModel.Digest() != target.Digest() ||
		!bytes.Equal(targetModel.CanonicalBytes(), target.record.record.object.CanonicalBytes()) {
		return FinalizedRunRecord{}, refuse(codeContractRecordRefused, "exact target record and target model are required", nil)
	}
	record, err := openFinalizedRunByTarget(ctx, target.store, target.record)
	if err != nil {
		return FinalizedRunRecord{}, err
	}
	run, err := parseFinalizedRunStorageRecord(record, targetModel)
	if err != nil {
		return FinalizedRunRecord{}, err
	}
	result := FinalizedRunRecord{
		store: target.store, target: target, record: record, model: run,
		release: &finalizedRunReleaseSeal{marker: 1},
	}
	if !result.Valid() {
		return FinalizedRunRecord{}, refuse(codeContractRecordRefused, "reopened finalized run is invalid", nil)
	}
	return result, nil
}

// OpenTerminalClosure reconstructs only an exact still-retained closure. It is
// for release recovery, not discovery or classification.
func OpenTerminalClosure(
	ctx context.Context,
	target ContractTargetRecord,
	targetModel contractmodel.ContractExecutionTarget,
) (TerminalClosure, error) {
	run, err := openFinalizedRunRecord(ctx, target, targetModel)
	if err != nil {
		return TerminalClosure{}, err
	}
	claim, err := openStartClaim(ctx, target.store, target.record)
	if err != nil || run.model.StartClaimDigest() != claim.digest {
		return TerminalClosure{}, refuse(codeContractRecordRefused, "finalized run and durable start claim differ", err)
	}
	spawn, err := openSpawnObservation(ctx, target.store, target.record, claim)
	if err != nil || !sameSpawnObservation(run.model.Witness().SpawnObservation(), spawn.observation) {
		return TerminalClosure{}, refuse(codeContractRecordRefused, "finalized run and durable spawn observation differ", err)
	}
	manifestDigest := run.model.PrivateManifest().ManifestRef().Digest()
	manifestRecord, err := openPrivateManifest(ctx, target.store, target.record, claim, manifestDigest)
	if err != nil {
		return TerminalClosure{}, err
	}
	if err := validatePrivateManifestForFinalization(ctx, target.store, manifestRecord); err != nil {
		return TerminalClosure{}, err
	}
	if err := validatePrivateManifestRoster(run.record.record.object, manifestRecord); err != nil {
		return TerminalClosure{}, err
	}
	manifest := PrivateRunManifest{store: target.store, record: manifestRecord}
	closureRecord := terminalClosureRecord{
		run: run.record, manifest: manifestRecord, seal: &terminalClosureSeal{marker: 1},
	}
	state := &terminalClosureState{
		store: target.store, target: target, claim: claim, spawn: spawn,
		manifest: manifest, run: run, record: closureRecord,
	}
	return TerminalClosure{state: state}, nil
}

func (closure TerminalClosure) FinalizedRun() FinalizedRunRecord {
	if closure.state == nil {
		return FinalizedRunRecord{}
	}
	return closure.state.run
}

// Release converges both CLEAR state and its deterministic FINALIZED_RUN
// receipt. A durable CLEAR with an interrupted receipt write is recoverable.
func (closure TerminalClosure) Release(ctx context.Context) error {
	state := closure.state
	if state == nil || !state.run.Valid() || !state.manifest.Valid() ||
		state.store == nil || !sameSpawnObservation(state.run.model.Witness().SpawnObservation(), state.spawn.observation) {
		return refuse(codeInterlockRefused, "exact terminal closure is required", nil)
	}
	durableSpawn, err := openSpawnObservation(ctx, state.store, state.target.record, state.claim)
	if err != nil || durableSpawn.bindingDigest != state.spawn.bindingDigest ||
		!sameSpawnObservation(durableSpawn.observation, state.spawn.observation) {
		return refuse(codeInterlockRefused, "durable spawn observation changed before release", err)
	}
	if err := releaseInterlockAfterFinalizedRun(ctx, state.store, state.record, state.claim, nil); err != nil {
		return err
	}
	if err := validateFinalizedRunRelease(ctx, state.store, state.claim, state.run.model.Digest()); err != nil {
		return err
	}
	state.run.markReleased()
	return nil
}

// PersistContractExecutionRecord exactly converges the classifier output. It
// accepts no requested result, tuple, reason, or process authority.
func PersistContractExecutionRecord(
	ctx context.Context,
	run FinalizedRunRecord,
	execution contractmodel.ContractExecution,
) (ContractExecutionRecord, error) {
	if !run.classificationReady() || !execution.Valid() || execution.TargetDigest() != run.model.TargetDigest() ||
		execution.FinalizedRunDigest() != run.model.Digest() {
		return ContractExecutionRecord{}, refuse(codeContractRecordRefused, "execution and exact finalized run do not join", nil)
	}
	claim, err := openStartClaim(ctx, run.store, run.target.record)
	if err != nil || claim.digest != run.model.StartClaimDigest() {
		return ContractExecutionRecord{}, refuse(codeContractRecordRefused, "durable start claim differs before classification publication", err)
	}
	if err := validateFinalizedRunRelease(ctx, run.store, claim, run.model.Digest()); err != nil {
		return ContractExecutionRecord{}, refuse(codeContractRecordRefused, "durable finalized-run release differs before classification publication", err)
	}
	object, err := contractExecutionSemanticObject(execution)
	if err != nil {
		return ContractExecutionRecord{}, err
	}
	_, _, persistErr := persistExecutionRecord(
		ctx, run.store, executionStorageInput{run: run.record, object: object}, nil,
	)
	reopened, openErr := openExecutionByRunProfile(ctx, run.store, run.record)
	if openErr != nil || reopened.record.object.Digest() != execution.Digest() ||
		!bytes.Equal(reopened.record.object.CanonicalBytes(), execution.CanonicalBytes()) {
		return ContractExecutionRecord{}, refuse(
			codeContractRecordAmbiguous, "execution publication did not reconcile exactly", errors.Join(persistErr, openErr),
		)
	}
	result := ContractExecutionRecord{
		store: run.store, run: run, record: reopened, model: execution,
	}
	if !result.Valid() {
		return ContractExecutionRecord{}, refuse(codeContractRecordRefused, "reopened execution record is invalid", nil)
	}
	return result, nil
}

func (record ContractExecutionRecord) Valid() bool {
	return record.store != nil && record.run.store == record.store && record.run.classificationReady() &&
		record.record.validFor(record.store) && record.model.Valid() &&
		record.model.Digest() == record.record.record.object.Digest() &&
		bytes.Equal(record.model.CanonicalBytes(), record.record.record.object.CanonicalBytes())
}

func (record ContractExecutionRecord) Digest() domain.Digest {
	if !record.Valid() {
		return ""
	}
	return record.model.Digest()
}

func (record ContractExecutionRecord) Model() contractmodel.ContractExecution {
	if !record.Valid() {
		return contractmodel.ContractExecution{}
	}
	return record.model
}

func persistSpawnObservation(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
	claim startClaimRecord,
	observation contractmodel.SpawnObservation,
	bindingDigest domain.Digest,
) (spawnObservationRecord, error) {
	if store == nil || !target.validFor(store) || !claim.validFor(store) ||
		!startClaimJoinsTarget(target, claim) {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation parents are invalid or foreign", nil)
	}
	record, err := buildSpawnObservationRecord(store, target, claim, observation, bindingDigest)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return spawnObservationRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return spawnObservationRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation namespace lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) || !claim.validForLocked(store) || !startClaimJoinsTarget(target, claim) {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation parents changed before persistence", nil)
	}
	directory, err := store.ensureContractDirectoryLocked(store.contractOps, spawnObservationDirectory)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	hex, err := strictDigestHex(target.record.object.Digest())
	if err != nil {
		return spawnObservationRecord{}, err
	}
	record.path = filepath.Join(directory, hex)
	if err := rejectPathCaseAlias(record.path); err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation path aliases an existing name", err)
	}
	if _, err := createExactPrivateFile(directory, hex, record.canonical, nil); err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordAmbiguous, "spawn observation did not persist exactly", err)
	}
	if !record.validForLocked(store) {
		return spawnObservationRecord{}, refuse(codeContractRecordAmbiguous, "spawn observation did not reopen exactly", nil)
	}
	return record, nil
}

func openSpawnObservation(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
	claim startClaimRecord,
) (spawnObservationRecord, error) {
	if store == nil || !target.validFor(store) || !claim.validFor(store) || !startClaimJoinsTarget(target, claim) {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation reopen parents are invalid", nil)
	}
	if err := storeContextRefusal(ctx, codeContractRecordRefused); err != nil {
		return spawnObservationRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return spawnObservationRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation namespace lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) || !claim.validForLocked(store) {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation parents changed before reopen", nil)
	}
	directory, err := store.ensureContractDirectoryLocked(store.contractOps, spawnObservationDirectory)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	hex, err := strictDigestHex(target.record.object.Digest())
	if err != nil {
		return spawnObservationRecord{}, err
	}
	path := filepath.Join(directory, hex)
	if err := rejectPathCaseAlias(path); err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn-observation path changed exact spelling", err)
	}
	body, err := readExactPrivateFile(path, -1)
	if err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation is absent or invalid", err)
	}
	record, err := parseSpawnObservationRecord(store, body, path)
	if err != nil || record.targetDigest != target.record.object.Digest() ||
		record.attemptDigest != target.record.relation.attemptDigest || record.claimDigest != claim.digest {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation does not join its exact parents", err)
	}
	if effect, convergeErr := createExactPrivateFile(directory, hex, body, nil); convergeErr != nil || effect != contractExactConverged {
		return spawnObservationRecord{}, refuse(codeContractRecordAmbiguous, "visible spawn observation did not converge durably", convergeErr)
	}
	if !record.validForLocked(store) {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "reopened spawn observation changed", nil)
	}
	return record, nil
}

func buildSpawnObservationRecord(
	store *ObjectStore,
	target targetStorageRecord,
	claim startClaimRecord,
	observation contractmodel.SpawnObservation,
	bindingDigest domain.Digest,
) (spawnObservationRecord, error) {
	if store == nil || !observation.Valid() || !bindingDigest.Valid() {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation or invocation binding is invalid", nil)
	}
	wire := map[string]any{
		"schema_version":            domain.SchemaVersion,
		"kind":                      spawnObservationKind,
		"observation_version":       spawnObservationVersion,
		"target_digest":             target.record.object.Digest().String(),
		"attempt_digest":            target.record.relation.attemptDigest.String(),
		"start_claim_digest":        claim.digest.String(),
		"invocation_binding_digest": bindingDigest.String(),
		"status":                    string(observation.State()),
	}
	if code, ok := observation.ErrorCode(); ok {
		wire["error_code"] = code
	} else if pid, ok := observation.PID(); ok {
		wire["pid"] = pid
	} else {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation has no exact outcome", nil)
	}
	exact, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return spawnObservationRecord{}, refuse(codeContractRecordRefused, "spawn observation could not be canonicalized", err)
	}
	digestValue, err := canon.DigestBytes(spawnObservationKind, exact)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	digest, err := domain.ParseDigest(digestValue.String())
	if err != nil {
		return spawnObservationRecord{}, err
	}
	return spawnObservationRecord{
		storeInstance: store.instance, targetDigest: target.record.object.Digest(),
		attemptDigest: target.record.relation.attemptDigest, claimDigest: claim.digest,
		bindingDigest: bindingDigest, observation: observation, digest: digest,
		canonical: append([]byte(nil), exact...),
	}, nil
}

func parseSpawnObservationRecord(
	store *ObjectStore,
	body []byte,
	path string,
) (spawnObservationRecord, error) {
	value, err := canon.Parse(body)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	exact, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return spawnObservationRecord{}, errors.Join(err, errors.New("spawn observation is not exact canonical JSON"))
	}
	status, err := privateText(value, "status")
	if err != nil {
		return spawnObservationRecord{}, err
	}
	var observation contractmodel.SpawnObservation
	switch contractmodel.SpawnObservationState(status) {
	case contractmodel.SpawnStartError:
		if !privateRoster(value, "schema_version", "kind", "observation_version", "target_digest", "attempt_digest", "start_claim_digest", "invocation_binding_digest", "status", "error_code") {
			return spawnObservationRecord{}, errors.New("start-error spawn observation roster differs")
		}
		code, codeErr := privateText(value, "error_code")
		if codeErr != nil {
			return spawnObservationRecord{}, codeErr
		}
		observation, err = contractmodel.NewStartErrorObservation(code)
	case contractmodel.SpawnChildPIDObserved:
		if !privateRoster(value, "schema_version", "kind", "observation_version", "target_digest", "attempt_digest", "start_claim_digest", "invocation_binding_digest", "status", "pid") {
			return spawnObservationRecord{}, errors.New("child spawn observation roster differs")
		}
		pid, pidErr := privateInteger(value, "pid")
		if pidErr != nil {
			return spawnObservationRecord{}, pidErr
		}
		observation, err = contractmodel.NewChildPIDObservation(pid)
	default:
		return spawnObservationRecord{}, errors.New("spawn observation state differs")
	}
	if err != nil {
		return spawnObservationRecord{}, err
	}
	schema, _ := privateText(value, "schema_version")
	kind, _ := privateText(value, "kind")
	version, _ := privateText(value, "observation_version")
	if schema != domain.SchemaVersion || kind != spawnObservationKind || version != spawnObservationVersion {
		return spawnObservationRecord{}, errors.New("spawn observation constants differ")
	}
	targetDigest, targetErr := privateDigest(value, "target_digest")
	attemptDigest, attemptErr := privateDigest(value, "attempt_digest")
	claimDigest, claimErr := privateDigest(value, "start_claim_digest")
	bindingDigest, bindingErr := privateDigest(value, "invocation_binding_digest")
	if errors.Join(targetErr, attemptErr, claimErr, bindingErr) != nil {
		return spawnObservationRecord{}, errors.Join(targetErr, attemptErr, claimErr, bindingErr)
	}
	digestValue, err := canon.DigestBytes(spawnObservationKind, body)
	if err != nil {
		return spawnObservationRecord{}, err
	}
	digest, err := domain.ParseDigest(digestValue.String())
	if err != nil {
		return spawnObservationRecord{}, err
	}
	return spawnObservationRecord{
		storeInstance: store.instance, targetDigest: targetDigest, attemptDigest: attemptDigest,
		claimDigest: claimDigest, bindingDigest: bindingDigest, observation: observation,
		digest: digest, canonical: append([]byte(nil), body...), path: path,
	}, nil
}

func (record spawnObservationRecord) validForLocked(store *ObjectStore) bool {
	if store == nil || record.storeInstance != store.instance || !record.targetDigest.Valid() ||
		!record.attemptDigest.Valid() || !record.claimDigest.Valid() || !record.bindingDigest.Valid() ||
		!record.observation.Valid() || !record.digest.Valid() || len(record.canonical) == 0 ||
		filepath.Dir(record.path) != filepath.Join(store.contractOps, spawnObservationDirectory) {
		return false
	}
	body, err := readExactPrivateFile(record.path, int64(len(record.canonical)))
	if err != nil || !bytes.Equal(body, record.canonical) {
		return false
	}
	rebuilt, err := parseSpawnObservationRecord(store, body, record.path)
	return err == nil && rebuilt.targetDigest == record.targetDigest &&
		rebuilt.attemptDigest == record.attemptDigest && rebuilt.claimDigest == record.claimDigest &&
		rebuilt.bindingDigest == record.bindingDigest && rebuilt.digest == record.digest &&
		sameSpawnObservation(rebuilt.observation, record.observation)
}

func sameSpawnObservation(left, right contractmodel.SpawnObservation) bool {
	if !left.Valid() || !right.Valid() || left.State() != right.State() {
		return false
	}
	if leftCode, leftOK := left.ErrorCode(); leftOK {
		rightCode, rightOK := right.ErrorCode()
		return rightOK && leftCode == rightCode
	}
	leftPID, leftOK := left.PID()
	rightPID, rightOK := right.PID()
	return leftOK && rightOK && leftPID == rightPID
}
