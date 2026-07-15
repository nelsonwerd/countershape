// Package store owns durable, edge-authority persistence. Inward semantic
// packages never import it.
package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

const (
	reductionSweepObjectsDirectory = "objects"
	reductionSweepObjectSuffix     = ".completed-sweep.json"
	reductionSweepTempPrefix       = ".completed-sweep-"

	codeInvalidReductionSweepStore = "INVALID_REDUCTION_SWEEP_STORE"
	codeInvalidCompletedSweepDraft = "INVALID_COMPLETED_SWEEP_DRAFT"
	codeSweepPublicationFailed     = "SWEEP_PUBLICATION_FAILED"
	codeSweepObjectRefused         = "SWEEP_OBJECT_REFUSED"
	codeSweepAuthorityRefused      = "SWEEP_AUTHORITY_REFUSED"
)

// Error is a stable store refusal. Detail and Cause explain the local failure;
// neither is semantic identity.
type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

// reductionSweepStoreInstance is deliberately process-local. A store reopened
// after restart must reopen the exact object and mint a new authority; copying
// a digest or serialized value cannot recreate this identity.
type reductionSweepStoreInstance struct {
	marker byte
	mu     sync.Mutex
}

// ReductionSweepStore is the minimal content-addressed store needed for the
// U5 final-sweep authority boundary. It owns one private root and its objects
// directory. Directory identities are retained so a path replacement fails
// closed after the store has been opened.
type ReductionSweepStore struct {
	root        string
	objects     string
	rootInfo    os.FileInfo
	objectsInfo os.FileInfo
	instance    *reductionSweepStoreInstance
}

// SweepCompletionAuthority is intentionally opaque. Its only contents bind a
// successfully published and reopened draft to the exact in-memory store
// instance that issued it. There is no wire representation or public
// constructor.
type SweepCompletionAuthority struct {
	storeInstance *reductionSweepStoreInstance
	draft         reduce.CompletedSweepDraft
}

// OpenReductionSweepStore creates, or defensively opens, a private store root
// and its private objects directory. The final root and objects path must be
// real directories rather than symlinks. Existing directories are never
// chmodded into compliance: an unexpectedly broad root is refused.
func OpenReductionSweepStore(root string) (*ReductionSweepStore, error) {
	if root == "" {
		return nil, refuse(codeInvalidReductionSweepStore, "store root is empty", nil)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, refuse(codeInvalidReductionSweepStore, "store root cannot be made absolute", err)
	}
	absolute = filepath.Clean(absolute)

	rootInfo, rootCreated, err := ensurePrivateDirectory(absolute, true)
	if err != nil {
		return nil, refuse(codeInvalidReductionSweepStore, "store root is not a private directory", err)
	}
	if rootCreated {
		if err := syncDirectory(filepath.Dir(absolute)); err != nil {
			return nil, refuse(codeInvalidReductionSweepStore, "store-root parent sync failed", err)
		}
	}

	objects := filepath.Join(absolute, reductionSweepObjectsDirectory)
	objectsInfo, objectsCreated, err := ensurePrivateDirectory(objects, false)
	if err != nil {
		return nil, refuse(codeInvalidReductionSweepStore, "objects path is not a private directory", err)
	}
	if objectsCreated {
		if err := syncDirectory(absolute); err != nil {
			return nil, refuse(codeInvalidReductionSweepStore, "store-root sync failed", err)
		}
	}

	return &ReductionSweepStore{
		root: absolute, objects: objects, rootInfo: rootInfo, objectsInfo: objectsInfo,
		instance: &reductionSweepStoreInstance{marker: 1},
	}, nil
}

// Publish atomically publishes the exact canonical draft, fsyncs the object
// directory, reopens and reparses the final object, and only then returns an
// authority. A preexisting exact object is reopened instead of overwritten;
// a partial, corrupt, wrong-mode, or symlink object is refused.
func (s *ReductionSweepStore) Publish(ctx context.Context, draft reduce.CompletedSweepDraft) (SweepCompletionAuthority, error) {
	if s == nil || s.instance == nil {
		return SweepCompletionAuthority{}, refuse(codeInvalidReductionSweepStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()

	if err := contextRefusal(ctx); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := validateCompletedSweepDraft(draft); err != nil {
		return SweepCompletionAuthority{}, err
	}

	destination := s.objectPath(draft)
	if _, err := os.Lstat(destination); err == nil {
		return s.openLocked(ctx, draft)
	} else if !errors.Is(err, os.ErrNotExist) {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "destination classification failed", err)
	}

	temporary, err := writeSyncedTemporary(s.objects, draft.CanonicalBytes())
	if err != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "temporary object write failed", err)
	}
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()

	// The temp path is reopened before publication. This catches short writes, mode
	// drift, path replacement, and noncanonical bytes without exposing a final
	// object name.
	if _, err := reopenExactDraft(temporary, draft); err != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "temporary object verification failed", err)
	}
	if err := contextRefusal(ctx); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SweepCompletionAuthority{}, err
	}

	// Never intentionally replace an existing content address. If another
	// conforming publisher won before this check, reopen its exact object.
	if _, err := os.Lstat(destination); err == nil {
		if removeErr := os.Remove(temporary); removeErr != nil {
			return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "redundant temporary object cleanup failed", removeErr)
		}
		removeTemporary = false
		return s.openLocked(ctx, draft)
	} else if !errors.Is(err, os.ErrNotExist) {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "destination reclassification failed", err)
	}
	if err := contextRefusal(ctx); err != nil {
		return SweepCompletionAuthority{}, err
	}

	// A same-directory hard link is an atomic create-if-absent publication:
	// unlike rename, it cannot overwrite a destination that appears after the
	// lstat above. The temporary name is removed only after the final name is
	// linked to the exact fsynced inode.
	if err := os.Link(temporary, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			if removeErr := os.Remove(temporary); removeErr != nil {
				return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "losing temporary object cleanup failed", removeErr)
			}
			removeTemporary = false
			return s.openLocked(ctx, draft)
		}
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "atomic no-replace object link failed", err)
	}
	if err := os.Remove(temporary); err != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "published temporary name cleanup failed", err)
	}
	removeTemporary = false

	// Once publication succeeds, finish the durability and verification sequence
	// even if cancellation races with it. Cancellation still prevents returning
	// the capability, but it never leaves a deliberately half-checked object.
	syncErr := syncDirectory(s.objects)
	reopened, reopenErr := reopenExactDraft(destination, draft)
	readyErr := s.assertReady()
	contextErr := contextRefusal(ctx)
	if syncErr != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "objects-directory sync failed", syncErr)
	}
	if reopenErr != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepPublicationFailed, "published object did not reopen exactly", reopenErr)
	}
	if readyErr != nil {
		return SweepCompletionAuthority{}, readyErr
	}
	if contextErr != nil {
		return SweepCompletionAuthority{}, contextErr
	}
	return s.authority(reopened), nil
}

// Open reopens an already-published exact draft and mints an authority for this
// store instance. This is the restart path: an authority from an earlier
// process is neither serialized nor reused.
func (s *ReductionSweepStore) Open(ctx context.Context, draft reduce.CompletedSweepDraft) (SweepCompletionAuthority, error) {
	if s == nil || s.instance == nil {
		return SweepCompletionAuthority{}, refuse(codeInvalidReductionSweepStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	return s.openLocked(ctx, draft)
}

// Validate checks store-instance identity and exact draft identity, then
// reopens and reparses the durable object again. Successful validation is not
// memoized; corruption or a symlink substitution after issuance fails closed.
func (s *ReductionSweepStore) Validate(ctx context.Context, draft reduce.CompletedSweepDraft, authority SweepCompletionAuthority) error {
	if s == nil || s.instance == nil {
		return refuse(codeInvalidReductionSweepStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()

	if err := contextRefusal(ctx); err != nil {
		return err
	}
	if authority.storeInstance == nil || authority.storeInstance != s.instance {
		return refuse(codeSweepAuthorityRefused, "authority was not issued by this store instance", nil)
	}
	if !sameCompletedSweepDraft(authority.draft, draft) {
		return refuse(codeSweepAuthorityRefused, "authority is not bound to the exact draft", nil)
	}
	if err := s.assertReady(); err != nil {
		return err
	}
	if err := syncDirectory(s.objects); err != nil {
		return refuse(codeSweepAuthorityRefused, "objects-directory sync failed during validation", err)
	}
	reopened, err := reopenExactDraft(s.objectPath(draft), draft)
	if err != nil {
		return refuse(codeSweepAuthorityRefused, "authority object did not reopen exactly", err)
	}
	if err := s.assertReady(); err != nil {
		return err
	}
	if !sameCompletedSweepDraft(reopened, authority.draft) {
		return refuse(codeSweepAuthorityRefused, "reopened object disagrees with issued authority", nil)
	}
	return contextRefusal(ctx)
}

func (s *ReductionSweepStore) openLocked(ctx context.Context, draft reduce.CompletedSweepDraft) (SweepCompletionAuthority, error) {
	if err := contextRefusal(ctx); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := validateCompletedSweepDraft(draft); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := syncDirectory(s.objects); err != nil {
		return SweepCompletionAuthority{}, refuse(codeSweepObjectRefused, "objects-directory sync failed before reopen", err)
	}
	reopened, err := reopenExactDraft(s.objectPath(draft), draft)
	if err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SweepCompletionAuthority{}, err
	}
	if err := contextRefusal(ctx); err != nil {
		return SweepCompletionAuthority{}, err
	}
	return s.authority(reopened), nil
}

func (s *ReductionSweepStore) authority(draft reduce.CompletedSweepDraft) SweepCompletionAuthority {
	return SweepCompletionAuthority{storeInstance: s.instance, draft: draft}
}

func (s *ReductionSweepStore) objectPath(draft reduce.CompletedSweepDraft) string {
	digest := strings.TrimPrefix(draft.Digest().String(), "sha256:")
	return filepath.Join(s.objects, digest+reductionSweepObjectSuffix)
}

func (s *ReductionSweepStore) assertReady() error {
	if s == nil || s.instance == nil || s.root == "" || s.objects == "" || s.rootInfo == nil || s.objectsInfo == nil {
		return refuse(codeInvalidReductionSweepStore, "store is zero or incomplete", nil)
	}
	rootInfo, err := exactPrivateDirectoryInfo(s.root)
	if err != nil || !os.SameFile(s.rootInfo, rootInfo) {
		return refuse(codeInvalidReductionSweepStore, "store root changed identity or facts", err)
	}
	objectsInfo, err := exactPrivateDirectoryInfo(s.objects)
	if err != nil || !os.SameFile(s.objectsInfo, objectsInfo) {
		return refuse(codeInvalidReductionSweepStore, "objects directory changed identity or facts", err)
	}
	return nil
}

func ensurePrivateDirectory(path string, parents bool) (os.FileInfo, bool, error) {
	info, err := os.Lstat(path)
	created := false
	if errors.Is(err, os.ErrNotExist) {
		if parents {
			err = os.MkdirAll(path, 0o700)
		} else {
			err = os.Mkdir(path, 0o700)
		}
		if err != nil {
			return nil, false, err
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return nil, false, err
		}
		created = true
		info, err = os.Lstat(path)
	}
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return nil, false, errors.New("path is a symlink, non-directory, or not mode 0700")
	}
	return info, created, nil
}

func exactPrivateDirectoryInfo(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return nil, errors.New("path is a symlink, non-directory, or not mode 0700")
	}
	return info, nil
}

func validateCompletedSweepDraft(draft reduce.CompletedSweepDraft) error {
	body := draft.CanonicalBytes()
	if !draft.Valid() || len(body) == 0 || len(body) > canon.MaxInputBytes {
		return refuse(codeInvalidCompletedSweepDraft, "draft is zero, invalid, or exceeds the canonical byte bound", nil)
	}
	parsed, err := reduce.ParseCompletedSweepDraft(body)
	if err != nil || !sameCompletedSweepDraft(parsed, draft) {
		return refuse(codeInvalidCompletedSweepDraft, "draft does not round-trip through its owning parser", err)
	}
	return nil
}

func sameCompletedSweepDraft(left, right reduce.CompletedSweepDraft) bool {
	if !left.Valid() || !right.Valid() || left.Digest() != right.Digest() {
		return false
	}
	return bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes())
}

func writeSyncedTemporary(directory string, body []byte) (string, error) {
	if len(body) == 0 || len(body) > canon.MaxInputBytes {
		return "", errors.New("temporary body is empty or exceeds the canonical byte bound")
	}
	handle, err := os.CreateTemp(directory, reductionSweepTempPrefix)
	if err != nil {
		return "", err
	}
	name := handle.Name()
	fail := func(primary error) (string, error) {
		closeErr := handle.Close()
		_ = os.Remove(name)
		return "", errors.Join(primary, closeErr)
	}
	if err := handle.Chmod(0o600); err != nil {
		return fail(err)
	}
	if err := writeAll(handle, body); err != nil {
		return fail(err)
	}
	if err := handle.Sync(); err != nil {
		return fail(err)
	}
	if err := handle.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func writeAll(handle *os.File, body []byte) error {
	written := 0
	for written < len(body) {
		count, err := handle.Write(body[written:])
		written += count
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

// reopenExactDraft uses a portable no-follow defense: classify with Lstat,
// open, compare the opened file identity via fstat, classify the path again,
// perform a bounded read, then classify a final time. Platforms with native
// no-follow opens may strengthen this later without weakening this sequence.
func reopenExactDraft(path string, expected reduce.CompletedSweepDraft) (reduce.CompletedSweepDraft, error) {
	expectedBytes := expected.CanonicalBytes()
	if len(expectedBytes) == 0 || len(expectedBytes) > canon.MaxInputBytes {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "expected draft bytes are invalid or oversized", nil)
	}

	before, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object path facts are invalid", err)
	}
	handle, err := os.Open(path)
	if err != nil {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object open failed", err)
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 ||
		opened.Size() != int64(len(expectedBytes)) || !os.SameFile(before, opened) {
		_ = handle.Close()
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "opened object disagrees with lstat facts", statErr)
	}
	afterOpen, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil || !os.SameFile(opened, afterOpen) {
		_ = handle.Close()
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object path changed during open", err)
	}

	body, readErr := io.ReadAll(io.LimitReader(handle, int64(canon.MaxInputBytes)+1))
	closeErr := handle.Close()
	if readErr != nil || closeErr != nil || len(body) > canon.MaxInputBytes || len(body) != len(expectedBytes) {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object bounded read was incomplete", errors.Join(readErr, closeErr))
	}
	finalInfo, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil || !os.SameFile(opened, finalInfo) {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object path changed during read", err)
	}
	if !bytes.Equal(body, expectedBytes) {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object bytes differ from the exact draft", nil)
	}
	reopened, err := reduce.ParseCompletedSweepDraft(body)
	if err != nil || !sameCompletedSweepDraft(reopened, expected) {
		return reduce.CompletedSweepDraft{}, refuse(codeSweepObjectRefused, "object failed owning-package reconstruction", err)
	}
	return reopened, nil
}

func exactObjectInfo(path string, expectedBytes int64) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() < 0 || info.Size() > int64(canon.MaxInputBytes) || info.Size() != expectedBytes {
		return nil, errors.New("object is a symlink, non-regular, wrong-mode, oversized, or wrong-sized")
	}
	return info, nil
}

func syncDirectory(path string) error {
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(handle.Sync(), handle.Close())
}

func contextRefusal(ctx context.Context) error {
	if ctx == nil {
		return refuse(codeSweepAuthorityRefused, "context is nil", nil)
	}
	if err := ctx.Err(); err != nil {
		return refuse(codeSweepAuthorityRefused, "operation context is not live", err)
	}
	return nil
}
