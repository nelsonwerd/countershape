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
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	privateManifestDirectory = "manifests"
	privatePackDirectory     = "packs"
	privatePurgeDirectory    = "purge-intents"
	privatePackHeader        = "COUNTERSHAPE_PRIVATE_EVIDENCE_PACK_V1\n"
	privateManifestKind      = "PrivateEvidenceManifest"
	privateManifestVersion   = "private-evidence-manifest/v1"
	privatePurgeKind         = "PrivateEvidencePurgeIntent"
	privatePurgeVersion      = "private-evidence-purge-intent/v1"
	privateDefaultExport     = "OMITTED"
	privateRetentionFact     = "RETAINED_AT_FINALIZATION"
	maxPrivateBlobs          = 16
	maxPrivateEvidenceBytes  = 64 * 1024 * 1024

	privateStateRetained          = "RETAINED"
	privateStatePurged            = "PURGED"
	privateStateMissingUnexpected = "MISSING_UNEXPECTED"

	codePrivateEvidenceRefused   = "PRIVATE_EVIDENCE_REFUSED"
	codePrivateEvidenceAmbiguous = "PRIVATE_EVIDENCE_AMBIGUOUS"
)

const (
	faultBeforePackLink       contractFaultPhase = "before-private-pack-link"
	faultAfterPackSync        contractFaultPhase = "after-private-pack-sync"
	faultBeforeManifestLink   contractFaultPhase = "before-private-manifest-link"
	faultAfterManifestSync    contractFaultPhase = "after-private-manifest-sync"
	faultBeforePurgeIntent    contractFaultPhase = "before-private-purge-intent"
	faultAfterPurgeIntentSync contractFaultPhase = "after-private-purge-intent-sync"
	faultAfterPackUnlink      contractFaultPhase = "after-private-pack-unlink"
	faultAfterPurgeSync       contractFaultPhase = "after-private-purge-directory-sync"
)

var privateEvidenceKindOrder = [...]string{
	"MATERIALIZATION_REVALIDATION",
	"RUNTIME_REVALIDATION",
	"PROCESS_RESULT",
	"WAIT_RESULT",
	"DRAIN_RESULT",
	"TEARDOWN_RESULT",
	"ORPHAN_CHECK",
	"FINALIZATION_MARKER",
	"CAPTURED_OBSERVATION",
	"PROJECTION_RESULT",
	"TARGET_INVENTORY",
	"CHILD_BINDINGS",
	"IMPORT_RESOLUTION",
	"SERVICE_BINDINGS",
	"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
}

type privateEvidenceReference struct {
	kind   string
	digest domain.Digest
}

func (reference privateEvidenceReference) valid() bool {
	if !reference.digest.Valid() {
		return false
	}
	for _, kind := range privateEvidenceKindOrder {
		if reference.kind == kind {
			return true
		}
	}
	return false
}

func (reference privateEvidenceReference) key() string {
	return reference.kind + "\x00" + reference.digest.String()
}

type privateBlobInput struct {
	body       []byte
	references []privateEvidenceReference
}

type privateManifestEntry struct {
	blobIndex  int64
	digest     domain.Digest
	count      int64
	offset     int64
	references []privateEvidenceReference
}

type privateManifestRecord struct {
	storeInstance    *objectStoreInstance
	targetDigest     domain.Digest
	attemptDigest    domain.Digest
	startClaimDigest domain.Digest
	digest           domain.Digest
	canonical        []byte
	packDigest       domain.Digest
	packBytes        int64
	blobCount        int64
	aggregateBytes   int64
	entries          []privateManifestEntry
	manifestPath     string
	packPath         string
	purgePath        string
	seal             *privateManifestSeal
}

type privateManifestSeal struct{ marker byte }

type privateEvidenceAvailability struct {
	state string
}

func openPrivateManifest(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
	claim startClaimRecord,
	manifestDigest domain.Digest,
) (privateManifestRecord, error) {
	if store == nil || !target.validFor(store) || !claim.validFor(store) || !manifestDigest.Valid() ||
		!startClaimJoinsTarget(target, claim) {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "manifest reopen parents are invalid or mismatched", nil)
	}
	if err := storeContextRefusal(ctx, codePrivateEvidenceRefused); err != nil {
		return privateManifestRecord{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return privateManifestRecord{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "private namespace lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) || !claim.validForLocked(store) {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "manifest parents changed before reopen", nil)
	}
	manifestDirectory, packDirectory, purgeDirectory, err := store.privateRunDirectoriesLocked()
	if err != nil {
		return privateManifestRecord{}, err
	}
	hex, _ := strictDigestHex(manifestDigest)
	manifestPath := filepath.Join(manifestDirectory, hex)
	packPath := filepath.Join(packDirectory, hex)
	purgePath := filepath.Join(purgeDirectory, hex)
	if aliasErr := errors.Join(
		rejectPathCaseAlias(manifestPath), rejectPathCaseAlias(packPath), rejectPathCaseAlias(purgePath),
	); aliasErr != nil {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "private evidence path aliases an existing name", aliasErr)
	}
	body, err := readExactPrivateFile(manifestPath, -1)
	if err != nil {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "private manifest is absent or invalid", err)
	}
	record, err := parsePrivateManifest(body)
	if err != nil || record.digest != manifestDigest || record.targetDigest != target.record.object.Digest() ||
		record.attemptDigest != target.record.relation.attemptDigest || record.startClaimDigest != claim.digest {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "private manifest does not join its exact parents", err)
	}
	if effect, convergeErr := createExactPrivateFile(
		manifestDirectory, filepath.Base(manifestPath), body, nil,
	); convergeErr != nil || effect != contractExactConverged {
		return privateManifestRecord{}, refuse(
			codePrivateEvidenceAmbiguous, "visible private manifest did not converge durably", convergeErr,
		)
	}
	record.storeInstance = store.instance
	record.manifestPath = manifestPath
	record.packPath = packPath
	record.purgePath = purgePath
	record.seal = &privateManifestSeal{marker: 1}
	if !record.validFor(store) {
		return privateManifestRecord{}, refuse(codePrivateEvidenceRefused, "reopened private manifest is invalid", nil)
	}
	return record, nil
}

func createPrivateManifest(
	ctx context.Context,
	store *ObjectStore,
	target targetStorageRecord,
	claim startClaimRecord,
	blobs []privateBlobInput,
	fault contractFault,
) (privateManifestRecord, contractEffect, error) {
	if store == nil || !target.validFor(store) || !claim.validFor(store) ||
		!startClaimJoinsTarget(target, claim) {
		return privateManifestRecord{}, "", refuse(codePrivateEvidenceRefused, "manifest parents are invalid, foreign, or mismatched", nil)
	}
	normalized, aggregate, err := normalizePrivateBlobs(blobs)
	if err != nil {
		return privateManifestRecord{}, "", err
	}
	pack := make([]byte, len(privatePackHeader), len(privatePackHeader)+int(aggregate))
	copy(pack, privatePackHeader)
	entries := make([]privateManifestEntry, len(normalized))
	manifestEntries := make([]any, len(normalized))
	logicalReferences := make(map[string]struct{}, len(privateEvidenceKindOrder))
	for index, blob := range normalized {
		offset := int64(len(pack))
		pack = append(pack, blob.body...)
		digest := privateBytesDigest(blob.body)
		references := make([]any, len(blob.references))
		for refIndex, reference := range blob.references {
			references[refIndex] = map[string]any{"kind": reference.kind, "digest": reference.digest.String()}
		}
		for _, reference := range blob.references {
			logicalReferences[reference.key()] = struct{}{}
		}
		entries[index] = privateManifestEntry{
			blobIndex: int64(index), digest: digest, count: int64(len(blob.body)), offset: offset,
			references: append([]privateEvidenceReference(nil), blob.references...),
		}
		manifestEntries[index] = map[string]any{
			"blob_index": int64(index), "blob_digest": digest.String(), "byte_count": int64(len(blob.body)),
			"pack_offset": offset, "evidence_refs": references,
		}
	}
	packDigest := privateBytesDigest(pack)
	manifestBytes, err := canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": privateManifestKind,
		"manifest_version":   privateManifestVersion,
		"target_digest":      target.record.object.Digest().String(),
		"attempt_digest":     target.record.relation.attemptDigest.String(),
		"start_claim_digest": claim.digest.String(),
		"entries":            manifestEntries, "logical_reference_count": int64(len(logicalReferences)),
		"blob_count": int64(len(normalized)), "aggregate_byte_count": aggregate,
		"pack_digest": packDigest.String(), "pack_byte_count": int64(len(pack)),
		"retention_at_finalization": privateRetentionFact, "default_export": privateDefaultExport,
	})
	if err != nil {
		return privateManifestRecord{}, "", refuse(codePrivateEvidenceRefused, "manifest cannot be canonicalized", err)
	}
	manifestDigestValue, err := canon.DigestBytes(privateManifestKind, manifestBytes)
	if err != nil {
		return privateManifestRecord{}, "", err
	}
	manifestDigest, err := domain.ParseDigest(manifestDigestValue.String())
	if err != nil {
		return privateManifestRecord{}, "", err
	}
	record := privateManifestRecord{
		storeInstance: store.instance, targetDigest: target.record.object.Digest(),
		attemptDigest: target.record.relation.attemptDigest, startClaimDigest: claim.digest,
		digest: manifestDigest, canonical: manifestBytes, packDigest: packDigest,
		packBytes: int64(len(pack)), blobCount: int64(len(normalized)), aggregateBytes: aggregate,
		entries: entries, seal: &privateManifestSeal{marker: 1},
	}
	hex, _ := strictDigestHex(manifestDigest)
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return privateManifestRecord{}, "", err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return privateManifestRecord{}, "", refuse(codePrivateEvidenceRefused, "private namespace lock failed", err)
	}
	defer lock.release()
	if !target.validForLocked(store) || !claim.validForLocked(store) {
		return privateManifestRecord{}, contractKnownNoEffect,
			refuse(codePrivateEvidenceRefused, "manifest parents changed before creation", nil)
	}
	manifestDirectory, packDirectory, purgeDirectory, err := store.privateRunDirectoriesLocked()
	if err != nil {
		return privateManifestRecord{}, "", err
	}
	record.manifestPath = filepath.Join(manifestDirectory, hex)
	record.packPath = filepath.Join(packDirectory, hex)
	record.purgePath = filepath.Join(purgeDirectory, hex)
	if aliasErr := errors.Join(
		rejectPathCaseAlias(record.manifestPath), rejectPathCaseAlias(record.packPath), rejectPathCaseAlias(record.purgePath),
	); aliasErr != nil {
		return privateManifestRecord{}, contractKnownNoEffect, mutationFailure(
			codePrivateEvidenceRefused, contractKnownNoEffect, faultBeforePackLink, aliasErr,
		)
	}
	if err := injectContractFault(fault, faultBeforePackLink); err != nil {
		return privateManifestRecord{}, contractKnownNoEffect, mutationFailure(codePrivateEvidenceRefused, contractKnownNoEffect, faultBeforePackLink, err)
	}
	packEffect, err := createExactPrivatePack(packDirectory, hex, pack, fault)
	if err != nil {
		return privateManifestRecord{}, packEffect, err
	}
	if err := injectContractFault(fault, faultBeforeManifestLink); err != nil {
		return privateManifestRecord{}, contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeManifestLink, err)
	}
	if _, err := createExactPrivateFile(manifestDirectory, hex, manifestBytes, fault); err != nil {
		return privateManifestRecord{}, contractAmbiguous,
			mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	if err := injectContractFault(fault, faultAfterManifestSync); err != nil {
		return privateManifestRecord{}, contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterManifestSync, err)
	}
	if err := validatePrivateManifestLocked(store, record); err != nil {
		return privateManifestRecord{}, contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeReopen, err)
	}
	return record, contractExactConverged, nil
}

func normalizePrivateBlobs(input []privateBlobInput) ([]privateBlobInput, int64, error) {
	if len(input) > maxPrivateBlobs {
		return nil, 0, refuse(codePrivateEvidenceRefused, "private blob count exceeds 16", nil)
	}
	normalized := make([]privateBlobInput, len(input))
	seenBlobs := make(map[domain.Digest]struct{}, len(input))
	seenReferenceKinds := make(map[string]domain.Digest, len(privateEvidenceKindOrder))
	var aggregate int64
	for index, blob := range input {
		if len(blob.body) == 0 || len(blob.references) == 0 {
			return nil, 0, refuse(codePrivateEvidenceRefused, "private blob or reference roster is empty", nil)
		}
		digest := privateBytesDigest(blob.body)
		if _, duplicate := seenBlobs[digest]; duplicate {
			return nil, 0, refuse(codePrivateEvidenceRefused, "private blob digest repeats", nil)
		}
		seenBlobs[digest] = struct{}{}
		references := append([]privateEvidenceReference(nil), blob.references...)
		sort.Slice(references, func(left, right int) bool { return references[left].key() < references[right].key() })
		for refIndex, reference := range references {
			if !reference.valid() {
				return nil, 0, refuse(codePrivateEvidenceRefused, "private logical reference is invalid", nil)
			}
			if refIndex > 0 && references[refIndex-1].key() == reference.key() {
				return nil, 0, refuse(codePrivateEvidenceRefused, "private logical reference repeats within one blob", nil)
			}
			if previous, present := seenReferenceKinds[reference.kind]; present && previous != reference.digest {
				return nil, 0, refuse(codePrivateEvidenceRefused, "one private evidence kind maps to multiple digests", nil)
			}
			seenReferenceKinds[reference.kind] = reference.digest
		}
		aggregate += int64(len(blob.body))
		if aggregate > maxPrivateEvidenceBytes {
			return nil, 0, refuse(codePrivateEvidenceRefused, "private evidence exceeds 64 MiB", nil)
		}
		normalized[index] = privateBlobInput{body: append([]byte(nil), blob.body...), references: references}
	}
	if len(seenReferenceKinds) > len(privateEvidenceKindOrder) {
		return nil, 0, refuse(codePrivateEvidenceRefused, "private logical reference roster exceeds the closed profile", nil)
	}
	sort.Slice(normalized, func(left, right int) bool {
		return privateBytesDigest(normalized[left].body).String() < privateBytesDigest(normalized[right].body).String()
	})
	return normalized, aggregate, nil
}

func parsePrivateManifest(body []byte) (privateManifestRecord, error) {
	root, err := canon.Parse(body)
	if err != nil {
		return privateManifestRecord{}, err
	}
	exact, err := root.CanonicalChecked()
	if err != nil || !bytes.Equal(exact, body) {
		return privateManifestRecord{}, errors.Join(err, errors.New("private manifest is not exact canonical JSON"))
	}
	if !privateRoster(root,
		"schema_version", "kind", "manifest_version", "target_digest", "attempt_digest",
		"start_claim_digest", "entries", "logical_reference_count", "blob_count",
		"aggregate_byte_count", "pack_digest", "pack_byte_count", "retention_at_finalization", "default_export",
	) {
		return privateManifestRecord{}, errors.New("private manifest root roster differs")
	}
	schema, _ := privateText(root, "schema_version")
	kind, _ := privateText(root, "kind")
	version, _ := privateText(root, "manifest_version")
	retention, _ := privateText(root, "retention_at_finalization")
	defaultExport, _ := privateText(root, "default_export")
	if schema != domain.SchemaVersion || kind != privateManifestKind || version != privateManifestVersion ||
		retention != privateRetentionFact || defaultExport != privateDefaultExport {
		return privateManifestRecord{}, errors.New("private manifest constants differ")
	}
	target, err := privateDigest(root, "target_digest")
	if err != nil {
		return privateManifestRecord{}, err
	}
	attempt, err := privateDigest(root, "attempt_digest")
	if err != nil {
		return privateManifestRecord{}, err
	}
	claim, err := privateDigest(root, "start_claim_digest")
	if err != nil {
		return privateManifestRecord{}, err
	}
	packDigest, err := privateDigest(root, "pack_digest")
	if err != nil {
		return privateManifestRecord{}, err
	}
	logicalCount, err := privateInteger(root, "logical_reference_count")
	if err != nil {
		return privateManifestRecord{}, err
	}
	blobCount, err := privateInteger(root, "blob_count")
	if err != nil {
		return privateManifestRecord{}, err
	}
	aggregate, err := privateInteger(root, "aggregate_byte_count")
	if err != nil {
		return privateManifestRecord{}, err
	}
	packBytes, err := privateInteger(root, "pack_byte_count")
	if err != nil {
		return privateManifestRecord{}, err
	}
	entriesValue, ok := root.LookupMember("entries")
	if !ok {
		return privateManifestRecord{}, errors.New("private manifest entries are absent")
	}
	entryValues, ok := entriesValue.Elements()
	if !ok || int64(len(entryValues)) != blobCount || blobCount < 0 || blobCount > maxPrivateBlobs ||
		aggregate < 0 || aggregate > maxPrivateEvidenceBytes || logicalCount < 0 || logicalCount > int64(len(privateEvidenceKindOrder)) ||
		packBytes != int64(len(privatePackHeader))+aggregate {
		return privateManifestRecord{}, errors.New("private manifest aggregate bounds differ")
	}
	entries := make([]privateManifestEntry, len(entryValues))
	seenReferenceKinds := make(map[string]domain.Digest, len(privateEvidenceKindOrder))
	expectedOffset := int64(len(privatePackHeader))
	var countedBytes int64
	var countedRefs int64
	previousBlobDigest := ""
	for index, entryValue := range entryValues {
		if !privateRoster(entryValue, "blob_index", "blob_digest", "byte_count", "pack_offset", "evidence_refs") {
			return privateManifestRecord{}, errors.New("private manifest entry roster differs")
		}
		blobIndex, err := privateInteger(entryValue, "blob_index")
		if err != nil || blobIndex != int64(index) {
			return privateManifestRecord{}, errors.New("private manifest blob index differs")
		}
		digest, err := privateDigest(entryValue, "blob_digest")
		if err != nil {
			return privateManifestRecord{}, err
		}
		if index > 0 && previousBlobDigest >= digest.String() {
			return privateManifestRecord{}, errors.New("private manifest blob order or uniqueness differs")
		}
		previousBlobDigest = digest.String()
		count, err := privateInteger(entryValue, "byte_count")
		if err != nil || count < 1 || count > maxPrivateEvidenceBytes ||
			countedBytes > maxPrivateEvidenceBytes-count {
			return privateManifestRecord{}, errors.New("private manifest blob count differs")
		}
		offset, err := privateInteger(entryValue, "pack_offset")
		if err != nil || offset != expectedOffset || offset < int64(len(privatePackHeader)) ||
			offset > packBytes-count {
			return privateManifestRecord{}, errors.New("private manifest pack offsets are not contiguous")
		}
		refsValue, _ := entryValue.LookupMember("evidence_refs")
		refs, ok := refsValue.Elements()
		if !ok || len(refs) == 0 {
			return privateManifestRecord{}, errors.New("private manifest evidence references are empty")
		}
		references := make([]privateEvidenceReference, len(refs))
		for refIndex, refValue := range refs {
			if !privateRoster(refValue, "kind", "digest") {
				return privateManifestRecord{}, errors.New("private evidence reference roster differs")
			}
			refKind, err := privateText(refValue, "kind")
			if err != nil {
				return privateManifestRecord{}, err
			}
			refDigest, err := privateDigest(refValue, "digest")
			if err != nil {
				return privateManifestRecord{}, err
			}
			reference := privateEvidenceReference{kind: refKind, digest: refDigest}
			if !reference.valid() {
				return privateManifestRecord{}, errors.New("private evidence reference kind differs")
			}
			if refIndex > 0 && references[refIndex-1].key() >= reference.key() {
				return privateManifestRecord{}, errors.New("private evidence reference order or uniqueness differs")
			}
			if previous, present := seenReferenceKinds[reference.kind]; present && previous != reference.digest {
				return privateManifestRecord{}, errors.New("one private evidence kind maps to multiple digests")
			}
			seenReferenceKinds[reference.kind] = reference.digest
			references[refIndex] = reference
		}
		entries[index] = privateManifestEntry{
			blobIndex: blobIndex, digest: digest, count: count, offset: offset, references: references,
		}
		expectedOffset += count
		countedBytes += count
		countedRefs = int64(len(seenReferenceKinds))
	}
	if countedBytes != aggregate || countedRefs != logicalCount || expectedOffset != packBytes {
		return privateManifestRecord{}, errors.New("private manifest totals differ from its entries")
	}
	digestValue, err := canon.DigestBytes(privateManifestKind, body)
	if err != nil {
		return privateManifestRecord{}, err
	}
	digest, _ := domain.ParseDigest(digestValue.String())
	return privateManifestRecord{
		targetDigest: target, attemptDigest: attempt, startClaimDigest: claim,
		digest: digest, canonical: append([]byte(nil), body...), packDigest: packDigest,
		packBytes: packBytes, blobCount: blobCount, aggregateBytes: aggregate, entries: entries,
	}, nil
}

func privateRoster(value canon.Value, names ...string) bool {
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

func privateText(value canon.Value, name string) (string, error) {
	member, ok := value.LookupMember(name)
	if !ok {
		return "", fmt.Errorf("missing %s", name)
	}
	text, ok := member.Text()
	if !ok {
		return "", fmt.Errorf("%s is not text", name)
	}
	return text, nil
}

func privateDigest(value canon.Value, name string) (domain.Digest, error) {
	raw, err := privateText(value, name)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(raw)
}

func privateInteger(value canon.Value, name string) (int64, error) {
	member, ok := value.LookupMember(name)
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	integer, ok := member.Int64()
	if !ok {
		return 0, fmt.Errorf("%s is not an integer", name)
	}
	return integer, nil
}

func (record privateManifestRecord) validFor(store *ObjectStore) bool {
	return store != nil && record.storeInstance == store.instance && record.seal != nil && record.seal.marker == 1 &&
		record.targetDigest.Valid() && record.attemptDigest.Valid() && record.startClaimDigest.Valid() &&
		record.digest.Valid() && record.packDigest.Valid() && record.blobCount >= 0 &&
		record.blobCount <= maxPrivateBlobs && record.aggregateBytes >= 0 &&
		record.aggregateBytes <= maxPrivateEvidenceBytes && len(record.canonical) > 0
}

func validatePrivateManifestForFinalization(ctx context.Context, store *ObjectStore, record privateManifestRecord) error {
	if err := storeContextRefusal(ctx, codePrivateEvidenceRefused); err != nil {
		return err
	}
	if !record.validFor(store) {
		return refuse(codePrivateEvidenceRefused, "private manifest record is invalid or foreign", nil)
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return refuse(codePrivateEvidenceRefused, "private namespace lock failed", err)
	}
	defer lock.release()
	return validatePrivateManifestLocked(store, record)
}

func validatePrivateManifestLocked(store *ObjectStore, record privateManifestRecord) error {
	if err := reopenPrivateManifestLocked(store, record); err != nil {
		return err
	}
	if err := rejectPathCaseAlias(record.purgePath); err != nil {
		return refuse(codePrivateEvidenceRefused, "purge intent changed spelling", err)
	}
	if _, err := os.Lstat(record.purgePath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return refuse(codePrivateEvidenceRefused, "purged private evidence cannot finalize a run", err)
	}
	if err := validatePrivatePack(record.packPath, record); err != nil {
		return err
	}
	return store.assertReady()
}

func reopenPrivateManifestLocked(store *ObjectStore, record privateManifestRecord) error {
	manifestDirectory, packDirectory, purgeDirectory, err := store.privateRunDirectoriesLocked()
	hex, digestErr := strictDigestHex(record.digest)
	manifestAliasErr := rejectPathCaseAlias(record.manifestPath)
	packAliasErr := rejectPathCaseAlias(record.packPath)
	purgeAliasErr := rejectPathCaseAlias(record.purgePath)
	if err != nil || digestErr != nil || record.manifestPath != filepath.Join(manifestDirectory, hex) ||
		record.packPath != filepath.Join(packDirectory, hex) || record.purgePath != filepath.Join(purgeDirectory, hex) ||
		manifestAliasErr != nil || packAliasErr != nil || purgeAliasErr != nil {
		return refuse(codePrivateEvidenceRefused, "private evidence path is invalid or changed", errors.Join(
			err, digestErr, manifestAliasErr, packAliasErr, purgeAliasErr,
		))
	}
	manifest, err := readExactPrivateFile(record.manifestPath, int64(len(record.canonical)))
	if err != nil || !bytes.Equal(manifest, record.canonical) {
		return refuse(codePrivateEvidenceRefused, "private manifest bytes changed", err)
	}
	reopened, err := parsePrivateManifest(manifest)
	if err != nil || reopened.digest != record.digest || reopened.targetDigest != record.targetDigest ||
		reopened.attemptDigest != record.attemptDigest || reopened.startClaimDigest != record.startClaimDigest ||
		reopened.packDigest != record.packDigest || reopened.packBytes != record.packBytes ||
		reopened.blobCount != record.blobCount || reopened.aggregateBytes != record.aggregateBytes ||
		!samePrivateManifestEntries(reopened.entries, record.entries) {
		return refuse(codePrivateEvidenceRefused, "private manifest did not reconstruct exactly", err)
	}
	return store.assertReady()
}

func samePrivateManifestEntries(left, right []privateManifestEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].blobIndex != right[index].blobIndex || left[index].digest != right[index].digest ||
			left[index].count != right[index].count || left[index].offset != right[index].offset ||
			len(left[index].references) != len(right[index].references) {
			return false
		}
		for refIndex := range left[index].references {
			if left[index].references[refIndex] != right[index].references[refIndex] {
				return false
			}
		}
	}
	return true
}

func validatePrivatePack(path string, record privateManifestRecord) error {
	aliasErr := rejectPathCaseAlias(path)
	info, err := os.Lstat(path)
	if aliasErr != nil || err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() != record.packBytes || record.packBytes < int64(len(privatePackHeader)) ||
		record.packBytes > int64(len(privatePackHeader))+maxPrivateEvidenceBytes {
		return refuse(codePrivateEvidenceRefused, "private pack facts differ", errors.Join(aliasErr, err))
	}
	handle, err := os.Open(path)
	if err != nil {
		return refuse(codePrivateEvidenceRefused, "private pack open failed", err)
	}
	defer handle.Close()
	opened, err := handle.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return refuse(codePrivateEvidenceRefused, "private pack changed during open", err)
	}
	header := make([]byte, len(privatePackHeader))
	if _, err := io.ReadFull(handle, header); err != nil || string(header) != privatePackHeader {
		return refuse(codePrivateEvidenceRefused, "private pack header differs", err)
	}
	for _, entry := range record.entries {
		if entry.count < 1 || entry.count > maxPrivateEvidenceBytes ||
			entry.offset < int64(len(privatePackHeader)) || entry.offset > record.packBytes ||
			entry.count > record.packBytes-entry.offset {
			return refuse(codePrivateEvidenceRefused, "private pack entry bounds differ", nil)
		}
		body := make([]byte, int(entry.count))
		if _, err := handle.ReadAt(body, entry.offset); err != nil || privateBytesDigest(body) != entry.digest {
			return refuse(codePrivateEvidenceRefused, "private pack entry differs", err)
		}
	}
	if digest, err := digestPrivatePack(handle, record.packBytes); err != nil || digest != record.packDigest {
		return refuse(codePrivateEvidenceRefused, "private pack aggregate digest differs", err)
	}
	after, err := os.Lstat(path)
	aliasErr = rejectPathCaseAlias(path)
	if err != nil || !os.SameFile(opened, after) || aliasErr != nil {
		return refuse(codePrivateEvidenceRefused, "private pack changed during validation", errors.Join(err, aliasErr))
	}
	return nil
}

func validatePrivateManifestRoster(run SemanticObject, manifest privateManifestRecord) error {
	references, blobCount, aggregate, manifestDigest, err := finalizedRunPrivateFacts(run)
	if err != nil || manifestDigest != manifest.digest || blobCount != manifest.blobCount || aggregate != manifest.aggregateBytes {
		return refuse(codePrivateEvidenceRefused, "run private summary differs from exact manifest", err)
	}
	expected := make([]string, 0, len(references))
	for _, reference := range references {
		expected = append(expected, reference.key())
	}
	actualSet := make(map[string]struct{}, len(expected))
	for _, entry := range manifest.entries {
		for _, reference := range entry.references {
			actualSet[reference.key()] = struct{}{}
		}
	}
	actual := make([]string, 0, len(actualSet))
	for key := range actualSet {
		actual = append(actual, key)
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if strings.Join(expected, "\n") != strings.Join(actual, "\n") {
		return refuse(codePrivateEvidenceRefused, "run logical evidence roster differs from the private manifest", nil)
	}
	return nil
}

func privateAvailability(
	ctx context.Context,
	store *ObjectStore,
	run finalizedRunStorageRecord,
	manifest privateManifestRecord,
) (privateEvidenceAvailability, error) {
	if store == nil || !run.validFor(store) || !manifest.validFor(store) ||
		run.record.relation.targetDigest != manifest.targetDigest ||
		run.record.relation.attemptDigest != manifest.attemptDigest ||
		run.record.relation.startClaimDigest != manifest.startClaimDigest {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "availability parents differ", nil)
	}
	if err := storeContextRefusal(ctx, codePrivateEvidenceRefused); err != nil {
		return privateEvidenceAvailability{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return privateEvidenceAvailability{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "private namespace lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return privateEvidenceAvailability{}, err
	}
	if !run.validForLocked(store) || validatePrivateManifestRoster(run.record.object, manifest) != nil ||
		reopenPrivateManifestLocked(store, manifest) != nil {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "run and durable private manifest do not rejoin exactly", nil)
	}
	if tombstone, err := readExactPrivateFile(manifest.purgePath, -1); err == nil {
		if purgeIntentExact(tombstone, run, manifest) {
			if effect, convergeErr := createExactPrivateFile(
				filepath.Dir(manifest.purgePath), filepath.Base(manifest.purgePath), tombstone, nil,
			); convergeErr != nil || effect != contractExactConverged {
				return privateEvidenceAvailability{}, mutationFailure(
					codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeReopen,
					errors.Join(convergeErr, errors.New("visible purge intent did not converge before classification")),
				)
			}
			return privateEvidenceAvailability{state: privateStatePurged}, nil
		}
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "purge intent is corrupt or mismatched", nil)
	} else if !errors.Is(err, os.ErrNotExist) {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "purge intent path is ambiguous", err)
	}
	if err := validatePrivatePack(manifest.packPath, manifest); err != nil {
		return privateEvidenceAvailability{state: privateStateMissingUnexpected}, nil
	}
	return privateEvidenceAvailability{state: privateStateRetained}, nil
}

func purgePrivateEvidence(
	ctx context.Context,
	store *ObjectStore,
	run finalizedRunStorageRecord,
	manifest privateManifestRecord,
	fault contractFault,
) (privateEvidenceAvailability, error) {
	if store == nil || !run.validFor(store) || !manifest.validFor(store) ||
		run.record.relation.targetDigest != manifest.targetDigest ||
		run.record.relation.attemptDigest != manifest.attemptDigest ||
		run.record.relation.startClaimDigest != manifest.startClaimDigest {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "purge parents differ", nil)
	}
	if err := storeContextRefusal(ctx, codePrivateEvidenceRefused); err != nil {
		return privateEvidenceAvailability{}, err
	}
	store.instance.mu.Lock()
	defer store.instance.mu.Unlock()
	if err := store.assertReady(); err != nil {
		return privateEvidenceAvailability{}, err
	}
	lock, err := openAndLockStudy(ctx, filepath.Join(store.contractRoot, contractNamespaceLock), true)
	if err != nil {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "private namespace lock failed", err)
	}
	defer lock.release()
	if err := store.assertReady(); err != nil {
		return privateEvidenceAvailability{}, err
	}
	if !run.validForLocked(store) || validatePrivateManifestRoster(run.record.object, manifest) != nil ||
		reopenPrivateManifestLocked(store, manifest) != nil {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "run and private manifest do not rejoin before purge", nil)
	}
	purgeBytes, err := privatePurgeIntent(run, manifest)
	if err != nil {
		return privateEvidenceAvailability{}, err
	}
	purgeDurable := false
	if existing, readErr := readExactPrivateFile(manifest.purgePath, int64(len(purgeBytes))); readErr == nil {
		if !bytes.Equal(existing, purgeBytes) {
			return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "existing purge intent is mismatched", nil)
		}
		if effect, convergeErr := createExactPrivateFile(
			filepath.Dir(manifest.purgePath), filepath.Base(manifest.purgePath), purgeBytes, nil,
		); convergeErr != nil || effect != contractExactConverged {
			return privateEvidenceAvailability{}, mutationFailure(
				codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeReopen,
				errors.Join(convergeErr, errors.New("visible purge intent did not converge durably")),
			)
		}
		purgeDurable = true
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return privateEvidenceAvailability{}, refuse(codePrivateEvidenceRefused, "purge intent path is ambiguous", readErr)
	} else if err := validatePrivatePack(manifest.packPath, manifest); err != nil {
		return privateEvidenceAvailability{state: privateStateMissingUnexpected}, err
	}
	if !purgeDurable {
		if err := injectContractFault(fault, faultBeforePurgeIntent); err != nil {
			return privateEvidenceAvailability{state: privateStateRetained}, mutationFailure(codePrivateEvidenceRefused, contractKnownNoEffect, faultBeforePurgeIntent, err)
		}
		purgeDirectory := filepath.Dir(manifest.purgePath)
		if _, err := createExactPrivateFile(purgeDirectory, filepath.Base(manifest.purgePath), purgeBytes, fault); err != nil {
			return privateEvidenceAvailability{}, err
		}
		if err := injectContractFault(fault, faultAfterPurgeIntentSync); err != nil {
			return privateEvidenceAvailability{state: privateStatePurged}, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPurgeIntentSync, err)
		}
		if tombstone, err := readExactPrivateFile(manifest.purgePath, int64(len(purgeBytes))); err != nil || !bytes.Equal(tombstone, purgeBytes) {
			return privateEvidenceAvailability{}, refuse(codePrivateEvidenceAmbiguous, "purge intent did not reopen exactly", err)
		}
	}
	if err := rejectPathCaseAlias(manifest.packPath); err != nil {
		return privateEvidenceAvailability{state: privateStatePurged}, refuse(codePrivateEvidenceRefused, "private pack changed spelling before purge", err)
	}
	if err := os.Remove(manifest.packPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return privateEvidenceAvailability{state: privateStatePurged}, nil
	}
	if err := injectContractFault(fault, faultAfterPackUnlink); err != nil {
		return privateEvidenceAvailability{state: privateStatePurged}, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPackUnlink, err)
	}
	if err := syncDirectory(filepath.Dir(manifest.packPath)); err != nil {
		return privateEvidenceAvailability{state: privateStatePurged}, nil
	}
	if err := injectContractFault(fault, faultAfterPurgeSync); err != nil {
		return privateEvidenceAvailability{state: privateStatePurged}, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPurgeSync, err)
	}
	return privateEvidenceAvailability{state: privateStatePurged}, nil
}

func (store *ObjectStore) privateRunDirectoriesLocked() (string, string, string, error) {
	manifest, err := store.ensureContractDirectoryLocked(store.contractRuns, privateManifestDirectory)
	if err != nil {
		return "", "", "", err
	}
	pack, err := store.ensureContractDirectoryLocked(store.contractRuns, privatePackDirectory)
	if err != nil {
		return "", "", "", err
	}
	purge, err := store.ensureContractDirectoryLocked(store.contractRuns, privatePurgeDirectory)
	if err != nil {
		return "", "", "", err
	}
	return manifest, pack, purge, nil
}

func createExactPrivatePack(directory, name string, body []byte, fault contractFault) (contractEffect, error) {
	if name == "" || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return contractKnownNoEffect, mutationFailure(codePrivateEvidenceRefused, contractKnownNoEffect, faultBeforePackLink, errors.New("invalid private pack component"))
	}
	if err := rejectCaseAlias(directory, name); err != nil {
		return contractKnownNoEffect, mutationFailure(codePrivateEvidenceRefused, contractKnownNoEffect, faultBeforePackLink, err)
	}
	destination := filepath.Join(directory, name)
	if info, err := os.Lstat(destination); err == nil {
		if info.Mode().IsRegular() && info.Mode().Perm() == 0o600 && info.Size() == int64(len(body)) {
			existing, readErr := os.ReadFile(destination)
			if readErr == nil && bytes.Equal(existing, body) {
				if syncErr := syncDirectory(directory); syncErr != nil {
					return contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPackSync, syncErr)
				}
				reopened, reopenErr := os.ReadFile(destination)
				if reopenErr != nil || !bytes.Equal(reopened, body) {
					return contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforeReopen, reopenErr)
				}
				return contractExactConverged, nil
			}
		}
		return contractAmbiguous, mutationFailure(codePrivateEvidenceRefused, contractAmbiguous, faultBeforePackLink, errors.New("alternate pack exists"))
	} else if !errors.Is(err, os.ErrNotExist) {
		return contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultBeforePackLink, err)
	}
	temporary, err := os.CreateTemp(directory, ".private-pack-")
	if err != nil {
		return contractKnownNoEffect, err
	}
	nameTemporary := temporary.Name()
	defer os.Remove(nameTemporary)
	if err := errors.Join(temporary.Chmod(0o600), writeAll(temporary, body), temporary.Sync(), temporary.Close()); err != nil {
		return contractKnownNoEffect, err
	}
	if err := os.Link(nameTemporary, destination); err != nil {
		return contractKnownNoEffect, err
	}
	if err := syncDirectory(directory); err != nil {
		return contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPackSync, err)
	}
	if err := injectContractFault(fault, faultAfterPackSync); err != nil {
		return contractAmbiguous, mutationFailure(codePrivateEvidenceAmbiguous, contractAmbiguous, faultAfterPackSync, err)
	}
	return contractExactConverged, nil
}

func privatePurgeIntent(run finalizedRunStorageRecord, manifest privateManifestRecord) ([]byte, error) {
	return canon.CanonicalizeTyped(map[string]any{
		"schema_version": domain.SchemaVersion, "kind": privatePurgeKind, "purge_version": privatePurgeVersion,
		"target_digest": manifest.targetDigest.String(), "attempt_digest": manifest.attemptDigest.String(),
		"start_claim_digest":   manifest.startClaimDigest.String(),
		"finalized_run_digest": run.record.object.Digest().String(), "manifest_digest": manifest.digest.String(),
		"disposition": "DO_NOT_SERVE", "secure_erasure_claim": false,
	})
}

func purgeIntentExact(body []byte, run finalizedRunStorageRecord, manifest privateManifestRecord) bool {
	expected, err := privatePurgeIntent(run, manifest)
	return err == nil && bytes.Equal(body, expected)
}

func privateBytesDigest(body []byte) domain.Digest {
	digest, err := canon.DigestBytes("PrivateEvidenceBytes", body)
	if err != nil {
		return ""
	}
	parsed, _ := domain.ParseDigest(digest.String())
	return parsed
}

func digestPrivatePack(handle *os.File, count int64) (domain.Digest, error) {
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	body, err := io.ReadAll(io.LimitReader(handle, count+1))
	if err != nil || int64(len(body)) != count {
		return "", errors.Join(err, errors.New("private pack bounded read differs"))
	}
	return privateBytesDigest(body), nil
}

func finalizedRunPrivateFacts(object SemanticObject) ([]privateEvidenceReference, int64, int64, domain.Digest, error) {
	root, err := canon.Parse(object.CanonicalBytes())
	if err != nil {
		return nil, 0, 0, "", err
	}
	witness, ok := root.LookupMember("closed_run_witness")
	if !ok {
		return nil, 0, 0, "", errors.New("run witness absent")
	}
	private, ok := witness.LookupMember("private_evidence")
	if !ok {
		return nil, 0, 0, "", errors.New("private summary absent")
	}
	manifestRef, ok := private.LookupMember("manifest_ref")
	if !ok {
		return nil, 0, 0, "", errors.New("manifest ref absent")
	}
	manifestDigest, err := typedReferenceDigest(manifestRef, "PRIVATE_EVIDENCE_MANIFEST")
	if err != nil {
		return nil, 0, 0, "", err
	}
	blobCount, err := integerValue(private, "blob_count")
	if err != nil {
		return nil, 0, 0, "", err
	}
	aggregate, err := integerValue(private, "aggregate_byte_count")
	if err != nil {
		return nil, 0, 0, "", err
	}
	references := make([]privateEvidenceReference, 0, len(privateEvidenceKindOrder))
	collectEvidenceReferences(witness, &references)
	filtered := references[:0]
	for _, reference := range references {
		if reference.kind != "PRIVATE_EVIDENCE_MANIFEST" {
			filtered = append(filtered, reference)
		}
	}
	return filtered, blobCount, aggregate, manifestDigest, nil
}

func collectEvidenceReferences(value canon.Value, output *[]privateEvidenceReference) {
	if members, ok := value.Members(); ok {
		kindValue, hasKind := value.LookupMember("kind")
		digestValue, hasDigest := value.LookupMember("digest")
		kind, kindText := kindValue.Text()
		raw, digestText := digestValue.Text()
		if hasKind && hasDigest && kindText && digestText {
			if digest, err := domain.ParseDigest(raw); err == nil {
				*output = append(*output, privateEvidenceReference{kind: kind, digest: digest})
			}
		}
		for _, member := range members {
			collectEvidenceReferences(member.Value, output)
		}
		return
	}
	if elements, ok := value.Elements(); ok {
		for _, element := range elements {
			collectEvidenceReferences(element, output)
		}
	}
}

func typedReferenceDigest(value canon.Value, wantKind string) (domain.Digest, error) {
	kindValue, kindOK := value.LookupMember("kind")
	digestValue, digestOK := value.LookupMember("digest")
	kind, kindText := kindValue.Text()
	raw, digestText := digestValue.Text()
	if !kindOK || !digestOK || !kindText || !digestText || kind != wantKind {
		return "", errors.New("typed reference differs")
	}
	return domain.ParseDigest(raw)
}

func integerValue(value canon.Value, name string) (int64, error) {
	member, ok := value.LookupMember(name)
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	integer, ok := member.Int64()
	if !ok {
		return 0, fmt.Errorf("%s is not an integer", name)
	}
	return integer, nil
}

func validPrivateLabel(value string) bool {
	return value != "" && len(value) <= 128 && utf8.ValidString(value) && !strings.ContainsRune(value, '\x00')
}
