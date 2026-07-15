package gitobj

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const materializationManifestFilename = "materialization.manifest.json"

type materializationManifestIdentity struct {
	SchemaVersion      string       `json:"schema_version"`
	Kind               string       `json:"kind"`
	PinnedTreeIdentity string       `json:"pinned_tree_identity_digest"`
	ObjectFormat       ObjectFormat `json:"object_format"`
	PolicyDigest       string       `json:"materialization_policy_digest"`
	PortableTreeDigest string       `json:"portable_tree_digest"`
	Entries            interface{}  `json:"entries"`
}

func buildMaterializationManifest(source InspectedTree) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped("MaterializationManifest", materializationManifestIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "MaterializationManifest",
		PinnedTreeIdentity: source.pinned.identity.String(), ObjectFormat: source.pinned.repository.objectFormat,
		PolicyDigest: source.policy.digest.String(), PortableTreeDigest: source.portableTreeDigest.String(),
		Entries: canonicalEntryIdentity(source.entries),
	})
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, append([]byte(nil), canonicalBytes...), nil
}

func materializedFileMode(gitMode string) (os.FileMode, error) {
	if gitMode == "100644" {
		return 0o644, nil
	}
	if gitMode == "100755" {
		if !preserveExecutableMode {
			return 0o644, nil
		}
		return 0o755, nil
	}
	return 0, refuse(CodeUnsupportedMode, gitMode, nil)
}

func requirePrivateEmptyParent(raw string) (string, error) {
	if !filepath.IsAbs(raw) || filepath.Clean(raw) != raw {
		return "", refuse(CodeInvalidConfig, "materialization parent must be clean and absolute", nil)
	}
	parent, err := canonicalDirectory(raw)
	if err != nil {
		return "", refuse(CodeInvalidConfig, "materialization parent is unavailable", err)
	}
	if parent != raw {
		return "", refuse(CodeInvalidConfig, "materialization parent must already be symlink-resolved", nil)
	}
	info, err := os.Stat(parent)
	if err != nil || info.Mode().Perm() != 0o700 {
		return "", refuse(CodeInvalidConfig, "materialization parent must be mode 0700", err)
	}
	handle, err := os.Open(parent)
	if err != nil {
		return "", refuse(CodeInvalidConfig, "materialization parent cannot be opened for inspection", err)
	}
	if err := inspectEmptyMaterializationParent(handle); err != nil {
		return "", err
	}
	return parent, nil
}

type boundedDirectoryEntryReadCloser interface {
	ReadDir(int) ([]os.DirEntry, error)
	Close() error
}

func inspectEmptyMaterializationParent(reader boundedDirectoryEntryReadCloser) error {
	// One physical entry is enough to refuse the parent. Never allocate or sort
	// the entire caller-controlled directory merely to prove nonemptiness.
	entries, readErr := reader.ReadDir(1)
	closeErr := reader.Close()
	if len(entries) > 1 {
		return refuse(CodeInvalidConfig, "materialization parent reader exceeded its requested bound", errors.Join(readErr, closeErr))
	}
	if len(entries) == 1 {
		return refuse(CodeInvalidConfig, "materialization parent must be empty", errors.Join(readErr, closeErr))
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return refuse(CodeInvalidConfig, "materialization parent cannot be inspected", errors.Join(readErr, closeErr))
	}
	if closeErr != nil {
		return refuse(CodeInvalidConfig, "materialization parent cannot be closed after inspection", closeErr)
	}
	if !errors.Is(readErr, io.EOF) {
		return refuse(CodeInvalidConfig, "materialization parent inspection made no progress", nil)
	}
	return nil
}

func reserveTopology(root string, entries []entryRecord) error {
	seenDirectories := map[string]struct{}{}
	for _, entry := range entries {
		components := strings.Split(entry.path, "/")
		for depth := 1; depth < len(components); depth++ {
			logical := strings.Join(components[:depth], "/")
			if _, exact := seenDirectories[logical]; exact {
				continue
			}
			if err := os.Mkdir(filepath.Join(root, filepath.FromSlash(logical)), 0o700); err != nil {
				return refuse(CodePathCollision, "target topology aliases a directory", err)
			}
			seenDirectories[logical] = struct{}{}
		}
		file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(entry.path)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return refuse(CodePathCollision, "target topology aliases a file", err)
		}
		if err := file.Close(); err != nil {
			return refuse(CodePublicationFailed, "reserved file close failed", err)
		}
	}
	return nil
}

func verifyReservedTarget(root string, entry entryRecord) error {
	target := filepath.Join(root, filepath.FromSlash(entry.path))
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() {
		return refuse(CodePublicationFailed, "reserved target changed type", err)
	}
	return nil
}

func writeVerifiedBlob(ctx context.Context, state *repositoryState, root string, entry entryRecord) error {
	if err := verifyReservedTarget(root, entry); err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(entry.path))
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return refuse(CodePublicationFailed, "reserved target cannot be opened", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	digest, err := verifyBlobStream(ctx, state, entry, file)
	if err != nil {
		return err
	}
	if digest != entry.portableDigest {
		return refuse(CodeSourceChanged, "blob portable digest changed after inspection", nil)
	}
	mode, err := materializedFileMode(entry.mode)
	if err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		return refuse(CodePublicationFailed, "materialized mode application failed", err)
	}
	if err := file.Sync(); err != nil {
		return refuse(CodePublicationFailed, "materialized file sync failed", err)
	}
	if err := file.Close(); err != nil {
		return refuse(CodePublicationFailed, "materialized file close failed", err)
	}
	closed = true
	return nil
}

func stagedPortableDigest(root string, entry entryRecord) (domain.Digest, error) {
	target := filepath.Join(root, filepath.FromSlash(entry.path))
	file, err := os.Open(target)
	if err != nil {
		return "", refuse(CodePublicationFailed, "staged file cannot be reopened", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != entry.size {
		return "", refuse(CodePublicationFailed, "staged file facts disagree with manifest", err)
	}
	expectedMode, err := materializedFileMode(entry.mode)
	if err != nil {
		return "", err
	}
	if info.Mode().Perm() != expectedMode.Perm() {
		return "", refuse(CodePublicationFailed, "staged file mode disagrees with manifest", nil)
	}
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, "countershape/portable-blob/v1")
	_, _ = hasher.Write([]byte{0})
	portableMode := entry.mode
	if !preserveExecutableMode && portableMode == "100755" {
		portableMode = "100644"
	}
	_, _ = io.WriteString(hasher, portableMode)
	_, _ = hasher.Write([]byte{0})
	_, _ = io.WriteString(hasher, strconv.FormatInt(entry.size, 10))
	_, _ = hasher.Write([]byte{0})
	written, err := io.Copy(hasher, file)
	if err != nil || written != entry.size {
		return "", refuse(CodePublicationFailed, "staged file rehash was incomplete", err)
	}
	return domain.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
}

func syncTreeDirectories(root string, entries []entryRecord) error {
	directories := map[string]struct{}{root: {}}
	for _, entry := range entries {
		current := filepath.Dir(filepath.Join(root, filepath.FromSlash(entry.path)))
		for strings.HasPrefix(current, root) {
			directories[current] = struct{}{}
			if current == root {
				break
			}
			current = filepath.Dir(current)
		}
	}
	ordered := make([]string, 0, len(directories))
	for directory := range directories {
		ordered = append(ordered, directory)
	}
	sort.Slice(ordered, func(a, b int) bool { return len(ordered[a]) > len(ordered[b]) })
	for _, directory := range ordered {
		handle, err := os.Open(directory)
		if err != nil {
			return refuse(CodePublicationFailed, "materialized directory cannot be opened for sync", err)
		}
		syncErr := handle.Sync()
		closeErr := handle.Close()
		if syncErr != nil || closeErr != nil {
			return refuse(CodePublicationFailed, "materialized directory sync failed", errors.Join(syncErr, closeErr))
		}
	}
	return nil
}

func syncDirectory(path string) error {
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(handle.Sync(), handle.Close())
}

type durableExclusiveFile interface {
	Write([]byte) (int, error)
	Chmod(os.FileMode) error
	Sync() error
	Close() error
}

type durableExclusiveFileOperations struct {
	open       func(string, int, os.FileMode) (durableExclusiveFile, error)
	remove     func(string) error
	lstat      func(string) (os.FileInfo, error)
	syncParent func(string) error
}

func defaultDurableExclusiveFileOperations() durableExclusiveFileOperations {
	return durableExclusiveFileOperations{
		open: func(path string, flags int, mode os.FileMode) (durableExclusiveFile, error) {
			return os.OpenFile(path, flags, mode)
		},
		remove:     os.Remove,
		lstat:      os.Lstat,
		syncParent: syncDirectory,
	}
}

func cleanupDurableExclusiveFile(path string, operations durableExclusiveFileOperations) (bool, error) {
	removeErr := operations.remove(path)
	parentSyncErr := operations.syncParent(filepath.Dir(path))
	_, absenceErr := operations.lstat(path)
	absenceProven := errors.Is(absenceErr, os.ErrNotExist)
	cause := errors.Join(removeErr, parentSyncErr)
	if !absenceProven {
		cause = errors.Join(cause, absenceErr)
	}
	return absenceProven && parentSyncErr == nil, cause
}

func durableExclusiveFileFailure(
	path string,
	handle durableExclusiveFile,
	operations durableExclusiveFileOperations,
	phase string,
	operationErr error,
) error {
	var closeErr error
	if handle != nil {
		closeErr = handle.Close()
	}
	absenceDurable, cleanupErr := cleanupDurableExclusiveFile(path, operations)
	cause := errors.Join(operationErr, closeErr, cleanupErr)
	if !absenceDurable {
		return refuse(
			CodePublicationAmbiguous,
			"canonical materialization manifest "+phase+" failed and cleanup could not prove durable absence",
			cause,
		)
	}
	return refuse(
		CodePublicationFailed,
		"canonical materialization manifest "+phase+" failed; created file was removed and its parent synced",
		cause,
	)
}

func writeDurableExclusiveFileWithOperations(
	path string,
	data []byte,
	mode os.FileMode,
	operations durableExclusiveFileOperations,
) error {
	if operations.open == nil || operations.remove == nil || operations.lstat == nil || operations.syncParent == nil {
		return refuse(CodePublicationFailed, "canonical materialization manifest operations are incomplete", nil)
	}
	handle, err := operations.open(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return refuse(CodePublicationFailed, "canonical materialization manifest create failed", err)
	}
	written := 0
	for written < len(data) {
		count, writeErr := handle.Write(data[written:])
		if count < 0 || count > len(data)-written {
			return durableExclusiveFileFailure(path, handle, operations, "write", io.ErrShortWrite)
		}
		written += count
		if writeErr != nil {
			return durableExclusiveFileFailure(path, handle, operations, "write", writeErr)
		}
		if count == 0 {
			return durableExclusiveFileFailure(path, handle, operations, "write", io.ErrNoProgress)
		}
	}
	if err := handle.Chmod(mode); err != nil {
		return durableExclusiveFileFailure(path, handle, operations, "chmod", err)
	}
	if err := handle.Sync(); err != nil {
		return durableExclusiveFileFailure(path, handle, operations, "sync", err)
	}
	if err := handle.Close(); err != nil {
		return durableExclusiveFileFailure(path, handle, operations, "close", err)
	}
	if err := operations.syncParent(filepath.Dir(path)); err != nil {
		return durableExclusiveFileFailure(path, nil, operations, "parent directory sync", err)
	}
	return nil
}

func writeDurableExclusiveFile(path string, data []byte, mode os.FileMode) error {
	return writeDurableExclusiveFileWithOperations(path, data, mode, defaultDurableExclusiveFileOperations())
}

type publicationOutcome int

const (
	publicationAbsent publicationOutcome = iota
	publicationDurable
	publicationAmbiguous
)

type publicationOperations struct {
	rename func(string, string) error
	sync   func(string) error
}

func publishStagedDirectory(stage, finalRoot, parent string, operations publicationOperations) (publicationOutcome, error) {
	if operations.rename == nil || operations.sync == nil {
		return publicationAbsent, errors.New("publication operations are incomplete")
	}
	if err := operations.rename(stage, finalRoot); err != nil {
		return publicationAbsent, err
	}
	if err := operations.sync(parent); err == nil {
		return publicationDurable, nil
	} else {
		originalSyncErr := err
		if rollbackErr := operations.rename(finalRoot, stage); rollbackErr != nil {
			return publicationAmbiguous, errors.Join(originalSyncErr, rollbackErr)
		}
		if rollbackSyncErr := operations.sync(parent); rollbackSyncErr != nil {
			// The destination is no longer visible in this process, but without a
			// durable parent-directory sync we cannot prove that absence across a
			// crash. Never collapse that persistence uncertainty into "absent".
			return publicationAmbiguous, errors.Join(originalSyncErr, rollbackSyncErr)
		}
		return publicationAbsent, originalSyncErr
	}
}

func (DefaultMaterializer) Materialize(
	ctx context.Context,
	bound BoundCandidate,
	rawParent string,
) (receipt MaterializationReceipt, resultErr error) {
	if !bound.Valid() {
		return MaterializationReceipt{}, refuse(CodePlanBindingMismatch, "candidate binding is invalid", nil)
	}
	parent, err := requirePrivateEmptyParent(rawParent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	// Reinspection on the exact publication volume proves topology, budgets, and
	// object bytes before the first candidate byte is written.
	fresh, err := Inspect(ctx, bound.source.pinned, bound.source.policy, parent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	if fresh.pinned.identity != bound.source.pinned.identity || fresh.policy.digest != bound.source.policy.digest ||
		fresh.portableTreeDigest != bound.source.portableTreeDigest || !entriesEqual(fresh.entries, bound.source.entries) {
		return MaterializationReceipt{}, refuse(CodeSourceChanged, "reinspection disagrees with the bound candidate", nil)
	}
	stage, err := os.MkdirTemp(parent, ".countershape-stage-")
	if err != nil {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "staging allocation failed", err)
	}
	published := false
	manifestPath := filepath.Join(parent, materializationManifestFilename)
	manifestCreated := false
	defer func() {
		if !published {
			_ = os.RemoveAll(stage)
			if manifestCreated {
				absenceDurable, cleanupErr := cleanupDurableExclusiveFile(
					manifestPath,
					defaultDurableExclusiveFileOperations(),
				)
				if !absenceDurable {
					resultErr = refuse(
						CodePublicationAmbiguous,
						"prepublication failure cleanup could not prove durable manifest absence",
						errors.Join(resultErr, cleanupErr),
					)
				}
			}
		}
	}()
	if err := os.Chmod(stage, 0o700); err != nil {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "staging mode failed", err)
	}
	if err := reserveTopology(stage, fresh.entries); err != nil {
		return MaterializationReceipt{}, err
	}
	fresh.pinned.repository.mu.Lock()
	defer fresh.pinned.repository.mu.Unlock()
	if err := fresh.pinned.repository.revalidate(); err != nil {
		return MaterializationReceipt{}, err
	}
	for _, entry := range fresh.entries {
		if err := writeVerifiedBlob(ctx, fresh.pinned.repository, stage, entry); err != nil {
			return MaterializationReceipt{}, err
		}
	}
	for _, entry := range fresh.entries {
		digest, err := stagedPortableDigest(stage, entry)
		if err != nil {
			return MaterializationReceipt{}, err
		}
		if digest != entry.portableDigest {
			return MaterializationReceipt{}, refuse(CodePublicationFailed, "staged file digest disagrees with manifest", nil)
		}
	}
	if _, err := os.Lstat(filepath.Join(stage, ".git")); err == nil || !errors.Is(err, os.ErrNotExist) {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, ".git appeared in staging", err)
	}
	manifestDigest, manifestBytes, err := buildMaterializationManifest(fresh)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	if err := syncTreeDirectories(stage, fresh.entries); err != nil {
		return MaterializationReceipt{}, err
	}
	if err := writeDurableExclusiveFile(manifestPath, manifestBytes, 0o600); err != nil {
		return MaterializationReceipt{}, err
	}
	manifestCreated = true
	finalRoot := filepath.Join(parent, "candidate")
	if _, err := os.Lstat(finalRoot); err == nil || !errors.Is(err, os.ErrNotExist) {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "final materialization destination is not absent", err)
	}
	receipt = MaterializationReceipt{
		ManifestDigest: manifestDigest, PortableTreeDigest: fresh.portableTreeDigest, PolicyDigest: fresh.policy.digest,
		TargetFilesystem: "device=" + fresh.targetVolume.identity.Device, Entries: fresh.Entries(), PublishedRoot: finalRoot,
		ManifestPath: manifestPath,
		ObjectFormat: fresh.pinned.repository.objectFormat, RepositoryFingerprint: fresh.pinned.repository.fingerprint,
	}
	receipt.seal, err = materializationReceiptSeal(receipt)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	outcome, err := publishStagedDirectory(stage, finalRoot, parent, publicationOperations{
		rename: renameExclusive, sync: syncDirectory,
	})
	if err != nil {
		if outcome == publicationAmbiguous {
			published = true
			return MaterializationReceipt{}, refuse(
				CodePublicationAmbiguous, "post-rename durability failed and rollback could not prove destination absence", err,
			)
		}
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "exclusive durable publication failed and destination is absent", err)
	}
	published = true
	return receipt, nil
}

func (r MaterializationReceipt) String() string {
	return fmt.Sprintf("%s %d files %s", r.ManifestDigest, len(r.Entries), r.TargetFilesystem)
}
