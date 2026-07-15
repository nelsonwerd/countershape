package gitobj

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func materializationReceiptSeal(receipt MaterializationReceipt) (domain.Digest, error) {
	entries := append([]ManifestEntry(nil), receipt.Entries...)
	return digestIdentity("MaterializationReceiptSeal", struct {
		SchemaVersion         string          `json:"schema_version"`
		Kind                  string          `json:"kind"`
		ManifestDigest        string          `json:"manifest_digest"`
		PortableTreeDigest    string          `json:"portable_tree_digest"`
		PolicyDigest          string          `json:"policy_digest"`
		TargetFilesystem      string          `json:"target_filesystem"`
		Entries               []ManifestEntry `json:"entries"`
		PublishedRoot         string          `json:"published_root"`
		ManifestPath          string          `json:"manifest_path"`
		ObjectFormat          ObjectFormat    `json:"object_format"`
		RepositoryFingerprint string          `json:"repository_fingerprint"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "MaterializationReceiptSeal",
		ManifestDigest: receipt.ManifestDigest.String(), PortableTreeDigest: receipt.PortableTreeDigest.String(),
		PolicyDigest: receipt.PolicyDigest.String(), TargetFilesystem: receipt.TargetFilesystem,
		Entries: entries, PublishedRoot: receipt.PublishedRoot, ManifestPath: receipt.ManifestPath, ObjectFormat: receipt.ObjectFormat,
		RepositoryFingerprint: receipt.RepositoryFingerprint.String(),
	})
}

// ValidatePublishedMaterialization reopens the candidate after publication and
// checks its complete topology, modes, sizes, and portable hashes against the
// unforgeable receipt and opaque bound source. It narrows the trusted interval
// immediately before spawn; same-user filesystem racing remains explicit.
func ValidatePublishedMaterialization(ctx context.Context, bound BoundCandidate, receipt MaterializationReceipt) error {
	if !bound.Valid() || !receipt.Valid() || receipt.PolicyDigest != bound.source.policy.digest ||
		receipt.PortableTreeDigest != bound.source.portableTreeDigest ||
		receipt.ObjectFormat != bound.source.pinned.repository.objectFormat ||
		receipt.RepositoryFingerprint != bound.source.pinned.repository.fingerprint ||
		len(receipt.Entries) != len(bound.source.entries) {
		return refuse(CodePublicationFailed, "published receipt is not authority for the bound source", nil)
	}
	root, err := canonicalDirectory(receipt.PublishedRoot)
	if err != nil || root != receipt.PublishedRoot {
		return refuse(CodePublicationFailed, "published root is not canonical and symlink-free", err)
	}
	expectedManifestPath := filepath.Join(filepath.Dir(root), materializationManifestFilename)
	if receipt.ManifestPath != expectedManifestPath {
		return refuse(CodePublicationFailed, "manifest path is not adjacent to the published candidate", nil)
	}
	manifestDigest, manifestBytes, err := buildMaterializationManifest(bound.source)
	if err != nil || manifestDigest != receipt.ManifestDigest {
		return refuse(CodePublicationFailed, "manifest digest differs from the bound source", err)
	}
	manifestInfo, err := os.Lstat(receipt.ManifestPath)
	if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Mode().Perm() != 0o600 ||
		manifestInfo.Size() != int64(len(manifestBytes)) {
		return refuse(CodePublicationFailed, "durable manifest facts are invalid", err)
	}
	actualManifest, err := os.ReadFile(receipt.ManifestPath)
	if err != nil || !sameCanonicalBytes(actualManifest, manifestBytes) {
		return refuse(CodePublicationFailed, "durable manifest bytes differ from the bound source", err)
	}
	expectedFiles := make(map[string]entryRecord, len(bound.source.entries))
	expectedDirectories := map[string]struct{}{root: {}}
	for index, entry := range bound.source.entries {
		manifest := receipt.Entries[index]
		if manifest.Path != entry.path || manifest.Mode != entry.mode || manifest.Bytes != entry.size ||
			manifest.GitOID != entry.oid || manifest.PortableBlobDigest != entry.portableDigest {
			return refuse(CodePublicationFailed, "receipt entries differ from the bound source", nil)
		}
		full := filepath.Join(root, filepath.FromSlash(entry.path))
		expectedFiles[full] = entry
		for directory := filepath.Dir(full); strings.HasPrefix(directory, root); directory = filepath.Dir(directory) {
			expectedDirectories[directory] = struct{}{}
			if directory == root {
				break
			}
		}
	}
	seenFiles := make(map[string]struct{}, len(expectedFiles))
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("published topology contains a symbolic link")
		}
		if entry.IsDir() {
			if _, allowed := expectedDirectories[path]; !allowed {
				return errors.New("published topology contains an extra directory")
			}
			return nil
		}
		expected, allowed := expectedFiles[path]
		if !allowed || !info.Mode().IsRegular() {
			return errors.New("published topology contains an extra or nonregular entry")
		}
		digest, err := stagedPortableDigest(root, expected)
		if err != nil {
			return err
		}
		if digest != expected.portableDigest {
			return errors.New("published file digest differs from the bound source")
		}
		seenFiles[path] = struct{}{}
		return nil
	})
	if err != nil {
		return refuse(CodePublicationFailed, "published candidate validation failed", err)
	}
	if len(seenFiles) != len(expectedFiles) {
		missing := make([]string, 0)
		for path := range expectedFiles {
			if _, seen := seenFiles[path]; !seen {
				missing = append(missing, path)
			}
		}
		sort.Strings(missing)
		return refuse(CodePublicationFailed, "published candidate is missing bound files", errors.New(strings.Join(missing, ",")))
	}
	return nil
}
