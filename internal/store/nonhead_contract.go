package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
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
}

type attemptRecordSeal struct{ marker byte }

func (record attemptStorageRecord) validFor(store *ObjectStore) bool {
	return store != nil && record.storeInstance == store.instance && record.seal != nil &&
		record.seal.marker == 1 && record.digest.Valid()
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
