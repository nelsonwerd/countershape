package world

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const cliFixtureOverlayAuthority = climodel.CLIFixtureOverlayAuthority

type CLIFixtureEntryReceipt struct {
	path       string
	mode       string
	bytes      int64
	byteDigest domain.Digest
}

func (e CLIFixtureEntryReceipt) Path() string              { return e.path }
func (e CLIFixtureEntryReceipt) Mode() string              { return e.mode }
func (e CLIFixtureEntryReceipt) Bytes() int64              { return e.bytes }
func (e CLIFixtureEntryReceipt) ByteDigest() domain.Digest { return e.byteDigest }

// CLIFixtureOverlayReceipt proves only that the immutable binding's bounded
// regular files were exclusively written beneath one new private fixture root,
// then reopened and rehashed before spawn. Candidate trees are never overlaid.
type CLIFixtureOverlayReceipt struct {
	digest                 domain.Digest
	canonicalBytes         []byte
	executionBindingDigest domain.Digest
	fixtureRecipeDigest    domain.Digest
	root                   string
	entries                []CLIFixtureEntryReceipt
	authority              string
}

type cliFixtureEntryIdentity struct {
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	Bytes      int64  `json:"bytes"`
	ByteDigest string `json:"byte_digest"`
}

type cliFixtureOverlayIdentity struct {
	SchemaVersion          string                    `json:"schema_version"`
	Kind                   string                    `json:"kind"`
	ExecutionBindingDigest string                    `json:"execution_binding_digest"`
	FixtureRecipeDigest    string                    `json:"fixture_recipe_digest"`
	FixtureRoot            string                    `json:"fixture_root"`
	Authority              string                    `json:"authority"`
	OverlayPolicy          string                    `json:"overlay_policy"`
	Entries                []cliFixtureEntryIdentity `json:"entries"`
}

func (r CLIFixtureOverlayReceipt) identity() cliFixtureOverlayIdentity {
	entries := make([]cliFixtureEntryIdentity, len(r.entries))
	for index, entry := range r.entries {
		entries[index] = cliFixtureEntryIdentity{
			Path: entry.path, Mode: entry.mode, Bytes: entry.bytes, ByteDigest: entry.byteDigest.String(),
		}
	}
	return cliFixtureOverlayIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIFixtureOverlayReceipt",
		ExecutionBindingDigest: r.executionBindingDigest.String(),
		FixtureRecipeDigest:    r.fixtureRecipeDigest.String(), FixtureRoot: r.root,
		Authority: r.authority, OverlayPolicy: "PRIVATE_ROOT_EXCLUSIVE_NO_OVERWRITE", Entries: entries,
	}
}

func (r CLIFixtureOverlayReceipt) Valid() bool {
	if !r.digest.Valid() || len(r.canonicalBytes) == 0 || !r.executionBindingDigest.Valid() ||
		!r.fixtureRecipeDigest.Valid() ||
		!filepath.IsAbs(r.root) || r.authority != cliFixtureOverlayAuthority {
		return false
	}
	for index, entry := range r.entries {
		if entry.path == "" || entry.bytes < 0 || !entry.byteDigest.Valid() ||
			entry.mode != string(climodel.FixtureMode0644) ||
			(index > 0 && r.entries[index-1].path >= entry.path) {
			return false
		}
	}
	digest, canonicalBytes, err := canon.DigestTyped("CLIFixtureOverlayReceipt", r.identity())
	return err == nil && digest.String() == r.digest.String() && bytes.Equal(canonicalBytes, r.canonicalBytes)
}

func (r CLIFixtureOverlayReceipt) Digest() domain.Digest { return r.digest }
func (r CLIFixtureOverlayReceipt) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r CLIFixtureOverlayReceipt) ExecutionBindingDigest() domain.Digest {
	return r.executionBindingDigest
}
func (r CLIFixtureOverlayReceipt) FixtureRecipeDigest() domain.Digest {
	return r.fixtureRecipeDigest
}
func (r CLIFixtureOverlayReceipt) Root() string      { return r.root }
func (r CLIFixtureOverlayReceipt) Authority() string { return r.authority }
func (r CLIFixtureOverlayReceipt) Entries() []CLIFixtureEntryReceipt {
	return append([]CLIFixtureEntryReceipt(nil), r.entries...)
}

func materializeCLIFixtures(binding climodel.CLIExecutionBinding, root string) (CLIFixtureOverlayReceipt, error) {
	if !binding.Valid() || !binding.FixtureRecipe().Valid() ||
		binding.FixtureRecipeDigest() != binding.FixtureRecipe().Digest() ||
		!filepath.IsAbs(root) || filepath.Clean(root) != root {
		return CLIFixtureOverlayReceipt{}, refuse(CodeCLIFixtureRejected, "fixture authority or root is invalid", nil)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return CLIFixtureOverlayReceipt{}, refuse(CodeCLIFixtureRejected, "fixture root is not canonical", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm()&0o077 != 0 {
		return CLIFixtureOverlayReceipt{}, refuse(CodeCLIFixtureRejected, "fixture root is not a private directory", err)
	}

	files := binding.Fixtures()
	entries := make([]CLIFixtureEntryReceipt, 0, len(files))
	for _, fixture := range files {
		entry, err := writeAndReopenCLIFixture(root, fixture)
		if err != nil {
			return CLIFixtureOverlayReceipt{}, err
		}
		entries = append(entries, entry)
	}
	if err := syncDirectory(root); err != nil {
		return CLIFixtureOverlayReceipt{}, refuse(CodeCLIFixtureRejected, "fixture-root sync failed", err)
	}

	receipt := CLIFixtureOverlayReceipt{
		executionBindingDigest: binding.Digest(), fixtureRecipeDigest: binding.FixtureRecipeDigest(), root: root,
		entries: append([]CLIFixtureEntryReceipt(nil), entries...), authority: cliFixtureOverlayAuthority,
	}
	digest, canonicalBytes, err := canon.DigestTyped("CLIFixtureOverlayReceipt", receipt.identity())
	if err != nil {
		return CLIFixtureOverlayReceipt{}, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return CLIFixtureOverlayReceipt{}, err
	}
	receipt.digest = parsed
	receipt.canonicalBytes = append([]byte(nil), canonicalBytes...)
	return receipt, nil
}

func writeAndReopenCLIFixture(root string, fixture climodel.CLIFixtureFile) (CLIFixtureEntryReceipt, error) {
	segments := strings.Split(fixture.Path(), "/")
	if len(segments) == 0 {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture path is empty", nil)
	}
	parent := root
	for _, segment := range segments[:len(segments)-1] {
		if strings.EqualFold(segment, ".git") {
			return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture path enters a reserved metadata name", nil)
		}
		next, err := ensureExactFixtureDirectory(parent, segment)
		if err != nil {
			return CLIFixtureEntryReceipt{}, err
		}
		parent = next
	}
	leaf := segments[len(segments)-1]
	if strings.EqualFold(leaf, ".git") {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture path uses a reserved metadata name", nil)
	}
	if present, _, err := exactOrAliasedChild(parent, leaf); err != nil {
		return CLIFixtureEntryReceipt{}, err
	} else if present {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture destination already exists or aliases an existing name", nil)
	}

	if fixture.Mode() != climodel.FixtureMode0644 {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture mode exceeds the fixed recipe authority", nil)
	}
	mode := os.FileMode(0o644)
	contents := fixture.Contents()
	expected, err := canon.DigestBytes("CLIFixtureBytes", contents)
	if err != nil {
		return CLIFixtureEntryReceipt{}, err
	}
	expectedDigest, err := domain.ParseDigest(expected.String())
	if err != nil {
		return CLIFixtureEntryReceipt{}, err
	}
	destination := filepath.Join(parent, leaf)
	if err := writeExclusiveSyncedFile(destination, contents, mode); err != nil {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture exclusive write failed", err)
	}
	if err := syncDirectory(parent); err != nil {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "fixture parent sync failed", err)
	}
	actualDigest, actualBytes, err := reopenAndHashFixture(destination, mode, int64(len(contents)))
	if err != nil {
		return CLIFixtureEntryReceipt{}, err
	}
	// MUTATION_ANCHOR: fixture-finalized-by-reopen-and-rehash
	if actualDigest != expectedDigest || !bytes.Equal(actualBytes, contents) {
		return CLIFixtureEntryReceipt{}, refuse(CodeCLIFixtureRejected, "reopened fixture bytes differ from the immutable binding", nil)
	}
	return CLIFixtureEntryReceipt{
		path: fixture.Path(), mode: string(fixture.Mode()), bytes: int64(len(contents)), byteDigest: actualDigest,
	}, nil
}

func ensureExactFixtureDirectory(parent, name string) (string, error) {
	present, info, err := exactOrAliasedChild(parent, name)
	if err != nil {
		return "", err
	}
	path := filepath.Join(parent, name)
	if present {
		if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", refuse(CodeCLIFixtureRejected, "fixture parent component is not an exact directory", nil)
		}
		return path, nil
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return "", refuse(CodeCLIFixtureRejected, "fixture parent directory creation failed", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", refuse(CodeCLIFixtureRejected, "fixture parent directory mode failed", err)
	}
	if err := syncDirectory(parent); err != nil {
		return "", refuse(CodeCLIFixtureRejected, "fixture parent publication sync failed", err)
	}
	info, err = os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", refuse(CodeCLIFixtureRejected, "created fixture parent did not reopen as a directory", err)
	}
	return path, nil
}

// exactOrAliasedChild refuses actual-filesystem aliases. If the requested name
// is absent from directory bytes but Lstat resolves it, the volume has mapped
// another spelling to the request and the bridge fails closed.
func exactOrAliasedChild(parent, name string) (bool, os.FileInfo, error) {
	directory, err := os.Open(parent)
	if err != nil {
		return false, nil, refuse(CodeCLIFixtureRejected, "fixture parent cannot be opened", err)
	}
	foundExact := false
	for {
		entries, readErr := directory.ReadDir(256)
		for _, entry := range entries {
			if entry.Name() == name {
				foundExact = true
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			_ = directory.Close()
			return false, nil, refuse(CodeCLIFixtureRejected, "fixture parent enumeration failed", readErr)
		}
		if len(entries) == 0 {
			_ = directory.Close()
			return false, nil, refuse(CodeCLIFixtureRejected, "fixture parent enumeration made no progress", nil)
		}
	}
	if err := directory.Close(); err != nil {
		return false, nil, refuse(CodeCLIFixtureRejected, "fixture parent close failed", err)
	}
	info, statErr := os.Lstat(filepath.Join(parent, name))
	if foundExact {
		if statErr != nil {
			return false, nil, refuse(CodeCLIFixtureRejected, "fixture child changed during classification", statErr)
		}
		return true, info, nil
	}
	if statErr == nil {
		return true, info, refuse(CodeCLIFixtureRejected, "fixture child aliases a different directory spelling", nil)
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return false, nil, refuse(CodeCLIFixtureRejected, "fixture child classification failed", statErr)
	}
	return false, nil, nil
}

func writeExclusiveSyncedFile(path string, contents []byte, mode os.FileMode) error {
	handle, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	written := 0
	for written < len(contents) {
		count, writeErr := handle.Write(contents[written:])
		written += count
		if writeErr != nil {
			_ = handle.Close()
			return writeErr
		}
		if count == 0 {
			_ = handle.Close()
			return io.ErrNoProgress
		}
	}
	if err := handle.Chmod(mode); err != nil {
		_ = handle.Close()
		return err
	}
	return errors.Join(handle.Sync(), handle.Close())
}

func reopenAndHashFixture(path string, mode os.FileMode, expectedBytes int64) (domain.Digest, []byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != mode.Perm() || info.Size() != expectedBytes {
		return "", nil, refuse(CodeCLIFixtureRejected, "fixture did not reopen with exact regular-file facts", err)
	}
	handle, err := os.Open(path)
	if err != nil {
		return "", nil, refuse(CodeCLIFixtureRejected, "fixture reopen failed", err)
	}
	bytes, readErr := io.ReadAll(io.LimitReader(handle, expectedBytes+1))
	closeErr := handle.Close()
	if readErr != nil || closeErr != nil || int64(len(bytes)) != expectedBytes {
		return "", nil, refuse(CodeCLIFixtureRejected, "fixture reopen read was incomplete", errors.Join(readErr, closeErr))
	}
	digest, err := canon.DigestBytes("CLIFixtureBytes", bytes)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, bytes, nil
}
