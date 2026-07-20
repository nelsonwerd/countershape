package gitobj

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaterializeSingleTarget publishes one inspected tree without manufacturing
// comparison-only WorldPlan or BoundCandidate authority. It shares the same
// private blob, topology, manifest, and durability mechanics as comparison
// materialization, then returns only a freshly reconstructed sealed receipt.
func MaterializeSingleTarget(
	ctx context.Context,
	source InspectedTree,
	rawParent string,
) (receipt MaterializationReceipt, resultErr error) {
	return materializeSingleTarget(ctx, source, rawParent, publicationOperations{
		rename: renameExclusive,
		sync:   syncDirectory,
	})
}

func materializeSingleTarget(
	ctx context.Context,
	source InspectedTree,
	rawParent string,
	operations publicationOperations,
) (receipt MaterializationReceipt, resultErr error) {
	if ctx == nil || !source.Valid() {
		return MaterializationReceipt{}, refuse(CodePlanBindingMismatch, "single-target source authority is invalid", nil)
	}
	if operations.rename == nil || operations.sync == nil {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target publication operations are incomplete", nil)
	}
	parent, err := requirePrivateEmptyParent(rawParent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	fresh, err := Inspect(ctx, source.pinned, source.policy, parent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	if !sameInspectedTree(source, fresh) {
		return MaterializationReceipt{}, refuse(CodeSourceChanged, "single-target reinspection disagrees with retained authority", nil)
	}
	if _, err := requirePrivateEmptyParent(parent); err != nil {
		return MaterializationReceipt{}, err
	}
	stage, err := os.MkdirTemp(parent, ".countershape-stage-")
	if err != nil {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target staging allocation failed", err)
	}
	published := false
	manifestPath := filepath.Join(parent, materializationManifestFilename)
	manifestCreated := false
	defer func() {
		if published {
			return
		}
		_ = os.RemoveAll(stage)
		if manifestCreated {
			absenceDurable, cleanupErr := cleanupDurableExclusiveFile(manifestPath, defaultDurableExclusiveFileOperations())
			if !absenceDurable {
				resultErr = refuse(CodePublicationAmbiguous, "single-target cleanup could not prove durable manifest absence", errors.Join(resultErr, cleanupErr))
			}
		}
	}()
	if err := os.Chmod(stage, 0o700); err != nil {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target staging mode failed", err)
	}
	if err := reserveTopology(stage, fresh.entries); err != nil {
		return MaterializationReceipt{}, err
	}
	fresh.pinned.repository.mu.Lock()
	if err := fresh.pinned.repository.revalidate(); err != nil {
		fresh.pinned.repository.mu.Unlock()
		return MaterializationReceipt{}, err
	}
	for _, entry := range fresh.entries {
		if err := writeVerifiedBlob(ctx, fresh.pinned.repository, stage, entry); err != nil {
			fresh.pinned.repository.mu.Unlock()
			return MaterializationReceipt{}, err
		}
	}
	fresh.pinned.repository.mu.Unlock()
	for _, entry := range fresh.entries {
		digest, err := stagedPortableDigest(stage, entry)
		if err != nil || digest != entry.portableDigest {
			return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target staged bytes disagree with inspected source", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(stage, ".git")); err == nil || !errors.Is(err, os.ErrNotExist) {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, ".git appeared in single-target staging", err)
	}
	_, manifestBytes, err := buildMaterializationManifest(fresh)
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
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target destination is not absent", err)
	}
	outcome, err := publishStagedDirectory(stage, finalRoot, parent, operations)
	if err != nil {
		if outcome == publicationAmbiguous {
			published = true
			return MaterializationReceipt{}, refuse(
				CodePublicationAmbiguous,
				"single-target post-rename durability is ambiguous; inspect the exact private parent before recovery",
				err,
			)
		}
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target publication failed with destination absent", err)
	}
	published = true
	reopened, err := ReopenSingleTarget(ctx, source, parent)
	if err != nil {
		return MaterializationReceipt{}, refuse(
			CodePublicationAmbiguous,
			"single-target publication is durable but exact reopen failed",
			err,
		)
	}
	return reopened, nil
}

// ReopenSingleTarget reconstructs authority only from an opaque inspected tree
// and its exact private parent. A copied receipt, OID, digest, or path bag is not
// an input to this operation.
func ReopenSingleTarget(ctx context.Context, source InspectedTree, rawParent string) (MaterializationReceipt, error) {
	if ctx == nil || !source.Valid() {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target reopen requires live inspected authority", nil)
	}
	parent, err := exactPublishedParent(rawParent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	if err := requireExactParentRoster(parent); err != nil {
		return MaterializationReceipt{}, err
	}
	fresh, err := Inspect(ctx, source.pinned, source.policy, parent)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	if !sameInspectedTree(source, fresh) {
		return MaterializationReceipt{}, refuse(CodeSourceChanged, "single-target source changed before reopen", nil)
	}
	if err := requireExactParentRoster(parent); err != nil {
		return MaterializationReceipt{}, err
	}
	manifestDigest, manifestBytes, err := buildMaterializationManifest(fresh)
	if err != nil {
		return MaterializationReceipt{}, err
	}
	root := filepath.Join(parent, "candidate")
	manifestPath := filepath.Join(parent, materializationManifestFilename)
	if err := validateSingleTargetManifest(manifestPath, manifestBytes); err != nil {
		return MaterializationReceipt{}, err
	}
	if err := validateSingleTargetTopology(ctx, root, fresh.entries); err != nil {
		return MaterializationReceipt{}, err
	}
	if err := syncTreeDirectories(root, fresh.entries); err != nil {
		return MaterializationReceipt{}, refuse(
			CodePublicationAmbiguous,
			"single-target tree durability could not be reconciled",
			err,
		)
	}
	if err := syncDirectory(parent); err != nil {
		return MaterializationReceipt{}, refuse(CodePublicationAmbiguous, "single-target parent durability could not be reconciled", err)
	}
	if err := requireExactParentRoster(parent); err != nil {
		return MaterializationReceipt{}, err
	}
	receipt := MaterializationReceipt{
		ManifestDigest: manifestDigest, PortableTreeDigest: fresh.portableTreeDigest, PolicyDigest: fresh.policy.digest,
		TargetFilesystem: "device=" + fresh.targetVolume.identity.Device, Entries: fresh.Entries(),
		PublishedRoot: root, ManifestPath: manifestPath, ObjectFormat: fresh.pinned.repository.objectFormat,
		RepositoryFingerprint: fresh.pinned.repository.fingerprint,
	}
	receipt.seal, err = materializationReceiptSeal(receipt)
	if err != nil || !receipt.Valid() {
		return MaterializationReceipt{}, refuse(CodePublicationFailed, "single-target receipt could not be reconstructed", err)
	}
	return receipt, nil
}

func sameInspectedTree(left, right InspectedTree) bool {
	return left.Valid() && right.Valid() && left.pinned.identity == right.pinned.identity &&
		left.pinned.repository == right.pinned.repository && left.policy.digest == right.policy.digest &&
		left.portableTreeDigest == right.portableTreeDigest && entriesEqual(left.entries, right.entries) &&
		left.targetVolume.identity.Equal(right.targetVolume.identity)
}

func exactPublishedParent(raw string) (string, error) {
	if !filepath.IsAbs(raw) || filepath.Clean(raw) != raw {
		return "", refuse(CodeInvalidConfig, "single-target parent must be clean and absolute", nil)
	}
	canonical, err := canonicalDirectory(raw)
	if err != nil || canonical != raw {
		return "", refuse(CodeInvalidConfig, "single-target parent must be symlink-free", err)
	}
	info, err := os.Lstat(raw)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return "", refuse(CodeInvalidConfig, "single-target parent is not one exact private directory", err)
	}
	return raw, nil
}

func requireExactParentRoster(parent string) error {
	handle, err := os.Open(parent)
	if err != nil {
		return refuse(CodePublicationFailed, "single-target parent cannot be opened", err)
	}
	entries, readErr := handle.ReadDir(3)
	closeErr := handle.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return refuse(CodePublicationFailed, "single-target parent roster cannot be read", errors.Join(readErr, closeErr))
	}
	if closeErr != nil {
		return refuse(CodePublicationFailed, "single-target parent cannot be closed", closeErr)
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	sort.Strings(names)
	want := []string{"candidate", materializationManifestFilename}
	if len(entries) != 2 || !equalStrings(names, want) {
		return refuse(CodePublicationFailed, "single-target parent roster differs", nil)
	}
	return nil
}

func validateSingleTargetManifest(path string, expected []byte) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Mode()&os.ModeSymlink != 0 ||
		info.Size() != int64(len(expected)) {
		return refuse(CodePublicationFailed, "single-target manifest facts differ", err)
	}
	body, err := os.ReadFile(path)
	if err != nil || !sameCanonicalBytes(body, expected) {
		return refuse(CodePublicationFailed, "single-target manifest bytes differ", err)
	}
	return nil
}

func validateSingleTargetTopology(ctx context.Context, root string, expected []entryRecord) error {
	canonical, err := canonicalDirectory(root)
	if err != nil || canonical != root {
		return refuse(CodePublicationFailed, "single-target root is not canonical", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm() != 0o700 || rootInfo.Mode()&os.ModeSymlink != 0 {
		return refuse(CodePublicationFailed, "single-target root facts differ", err)
	}
	expectedFiles := make(map[string]entryRecord, len(expected))
	expectedDirectories := map[string]struct{}{root: {}}
	for _, entry := range expected {
		full := filepath.Join(root, filepath.FromSlash(entry.path))
		expectedFiles[full] = entry
		for directory := filepath.Dir(full); strings.HasPrefix(directory, root); directory = filepath.Dir(directory) {
			expectedDirectories[directory] = struct{}{}
			if directory == root {
				break
			}
		}
	}
	seen := make(map[string]struct{}, len(expectedFiles))
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.Join(errors.New("single-target entry is unavailable or symbolic"), err)
		}
		if entry.IsDir() {
			if _, allowed := expectedDirectories[path]; !allowed || info.Mode().Perm() != 0o700 {
				return errors.New("single-target directory roster or mode differs")
			}
			return nil
		}
		record, allowed := expectedFiles[path]
		if !allowed || !info.Mode().IsRegular() {
			return errors.New("single-target file roster or type differs")
		}
		digest, err := stagedPortableDigest(root, record)
		if err != nil || digest != record.portableDigest {
			return errors.Join(errors.New("single-target file identity differs"), err)
		}
		seen[path] = struct{}{}
		return nil
	})
	if err != nil || len(seen) != len(expectedFiles) {
		return refuse(CodePublicationFailed, "single-target topology validation failed", err)
	}
	return nil
}

func equalStrings(left, right []string) bool {
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
