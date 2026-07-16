// Package store owns durable, edge-authority persistence. Semantic packages
// own their codecs; the store preserves their exact canonical bytes.
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
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	objectDirectory         = "objects"
	objectAlgorithm         = "sha256"
	studyDirectory          = "studies"
	privateCaptureDirectory = "private-captures"
	objectTempPrefix        = ".object-"

	codeInvalidObjectStore      = "INVALID_OBJECT_STORE"
	codeInvalidSemanticObject   = "INVALID_SEMANTIC_OBJECT"
	codeObjectPublicationFailed = "OBJECT_PUBLICATION_FAILED"
	codeObjectRefused           = "OBJECT_REFUSED"
	codeObjectAuthorityRefused  = "OBJECT_AUTHORITY_REFUSED"
)

// Error is a stable store refusal. Detail and Cause are diagnostics and never
// participate in semantic identity.
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

// SemanticObject is exact canonical content plus the domain-separated digest
// its owning package computed. It deliberately has no ambient JSON marshal
// path and no pathname supplied by a caller.
type SemanticObject struct {
	kind      string
	digest    domain.Digest
	canonical []byte
}

func NewSemanticObject(kind string, expected domain.Digest, exactCanonical []byte) (SemanticObject, error) {
	if kind == "" || len(kind) > 128 || !utf8.ValidString(kind) || !expected.Valid() ||
		len(exactCanonical) == 0 || len(exactCanonical) > canon.MaxInputBytes {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "object identity is incomplete or outside the bounded profile", nil)
	}
	value, err := canon.Parse(exactCanonical)
	if err != nil {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "object bytes are outside the strict canonical profile", err)
	}
	canonicalBytes, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonicalBytes, exactCanonical) {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "object bytes are not exact canonical bytes", err)
	}
	members, object := value.Members()
	if !object || len(members) < 2 {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "semantic object must be a typed canonical object", nil)
	}
	schemaValue, hasSchema := value.LookupMember("schema_version")
	kindValue, hasKind := value.LookupMember("kind")
	schema, schemaText := schemaValue.Text()
	actualKind, kindText := kindValue.Text()
	if !hasSchema || !hasKind || !schemaText || !kindText || schema != domain.SchemaVersion || actualKind != kind {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "object schema or kind disagrees with its owning type", nil)
	}
	digest, err := canon.DigestBytes(kind, exactCanonical)
	if err != nil || digest.String() != expected.String() {
		return SemanticObject{}, refuse(codeInvalidSemanticObject, "expected digest differs from exact canonical bytes", err)
	}
	return SemanticObject{kind: kind, digest: expected, canonical: append([]byte(nil), exactCanonical...)}, nil
}

func (o SemanticObject) Valid() bool {
	rebuilt, err := NewSemanticObject(o.kind, o.digest, o.canonical)
	return err == nil && rebuilt.kind == o.kind && rebuilt.digest == o.digest && bytes.Equal(rebuilt.canonical, o.canonical)
}

func (o SemanticObject) Kind() string           { return o.kind }
func (o SemanticObject) Digest() domain.Digest  { return o.digest }
func (o SemanticObject) CanonicalBytes() []byte { return append([]byte(nil), o.canonical...) }

type objectStoreInstance struct {
	marker byte
	mu     sync.Mutex
}

// ObjectStore owns one private create-once semantic namespace and the mutable
// study-selector namespace used by head CAS. Directory identities are retained
// so replacement after open fails closed.
type ObjectStore struct {
	root            string
	objects         string
	digestRoot      string
	studies         string
	privateCaptures string
	rootInfo        os.FileInfo
	objectsInfo     os.FileInfo
	digestRootInfo  os.FileInfo
	studiesInfo     os.FileInfo
	privateInfo     os.FileInfo
	shardInfos      map[string]os.FileInfo
	studyInfos      map[string]os.FileInfo
	instance        *objectStoreInstance
}

type ObjectAuthority struct {
	storeInstance *objectStoreInstance
	object        SemanticObject
}

func OpenObjectStore(root string) (*ObjectStore, error) {
	if root == "" {
		return nil, refuse(codeInvalidObjectStore, "store root is empty", nil)
	}
	absolute, err := filepath.Abs(root)
	if err != nil || filepath.Clean(absolute) != absolute {
		return nil, refuse(codeInvalidObjectStore, "store root cannot be made clean and absolute", err)
	}
	rootInfo, rootCreated, err := ensurePrivateRoot(absolute)
	if err != nil {
		return nil, refuse(codeInvalidObjectStore, "store root is not a private real directory", err)
	}
	if rootCreated {
		if err := syncDirectory(filepath.Dir(absolute)); err != nil {
			return nil, refuse(codeInvalidObjectStore, "store-root parent sync failed", err)
		}
	}

	paths := []string{
		filepath.Join(absolute, objectDirectory),
		filepath.Join(absolute, objectDirectory, objectAlgorithm),
		filepath.Join(absolute, studyDirectory),
		filepath.Join(absolute, privateCaptureDirectory),
	}
	infos := make([]os.FileInfo, len(paths))
	for index, path := range paths {
		info, created, directoryErr := ensurePrivateDirectory(path)
		if directoryErr != nil {
			return nil, refuse(codeInvalidObjectStore, "owned store directory is invalid", directoryErr)
		}
		infos[index] = info
		if created {
			if err := syncDirectory(filepath.Dir(path)); err != nil {
				return nil, refuse(codeInvalidObjectStore, "owned directory parent sync failed", err)
			}
		}
	}
	return &ObjectStore{
		root: absolute, objects: paths[0], digestRoot: paths[1], studies: paths[2], privateCaptures: paths[3],
		rootInfo: rootInfo, objectsInfo: infos[0], digestRootInfo: infos[1], studiesInfo: infos[2], privateInfo: infos[3],
		shardInfos: make(map[string]os.FileInfo), studyInfos: make(map[string]os.FileInfo),
		instance: &objectStoreInstance{marker: 1},
	}, nil
}

func (s *ObjectStore) Publish(ctx context.Context, object SemanticObject) (ObjectAuthority, error) {
	if s == nil || s.instance == nil {
		return ObjectAuthority{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	return s.publishLocked(ctx, object)
}

func (s *ObjectStore) publishLocked(ctx context.Context, object SemanticObject) (ObjectAuthority, error) {
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return ObjectAuthority{}, err
	}
	if !object.Valid() {
		return ObjectAuthority{}, refuse(codeInvalidSemanticObject, "object is zero or invalid", nil)
	}
	destination, shard, err := s.ensureObjectPath(object.digest)
	if err != nil {
		return ObjectAuthority{}, err
	}
	if _, err := os.Lstat(destination); err == nil {
		return s.openAtLocked(ctx, object, destination, shard)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "destination classification failed", err)
	}

	temporary, err := writeSyncedTemporary(shard, objectTempPrefix, object.canonical)
	if err != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "temporary object write failed", err)
	}
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := reopenExactObject(temporary, object); err != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "temporary object verification failed", err)
	}
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return ObjectAuthority{}, err
	}
	if err := assertExactDirectory(shard); err != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "object shard changed facts", err)
	}
	if err := rejectCaseAlias(shard, filepath.Base(destination)); err != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "object namespace has a case alias", err)
	}
	if err := os.Link(temporary, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			if removeErr := os.Remove(temporary); removeErr != nil {
				return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "losing temporary cleanup failed", removeErr)
			}
			removeTemporary = false
			return s.openAtLocked(ctx, object, destination, shard)
		}
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "atomic no-replace object link failed", err)
	}
	if err := os.Remove(temporary); err != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "published temporary cleanup failed", err)
	}
	removeTemporary = false

	// Once linked, finish durability/reopen even if cancellation races. A
	// cancelled caller never receives the capability.
	syncErr := syncDirectory(shard)
	reopened, reopenErr := reopenExactObject(destination, object)
	readyErr := s.assertReady()
	contextErr := storeContextRefusal(ctx, codeObjectAuthorityRefused)
	if syncErr != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "object-shard sync failed", syncErr)
	}
	if reopenErr != nil {
		return ObjectAuthority{}, refuse(codeObjectPublicationFailed, "published object did not reopen exactly", reopenErr)
	}
	if readyErr != nil {
		return ObjectAuthority{}, readyErr
	}
	if contextErr != nil {
		return ObjectAuthority{}, contextErr
	}
	return s.authority(reopened), nil
}

func (s *ObjectStore) Open(ctx context.Context, object SemanticObject) (ObjectAuthority, error) {
	if s == nil || s.instance == nil {
		return ObjectAuthority{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if !object.Valid() {
		return ObjectAuthority{}, refuse(codeInvalidSemanticObject, "object is zero or invalid", nil)
	}
	path, shard, err := s.existingObjectPath(object.digest)
	if err != nil {
		return ObjectAuthority{}, err
	}
	return s.openAtLocked(ctx, object, path, shard)
}

// Read reopens an immutable object by exact kind/digest. The returned authority
// proves durable bytes in this store instance, not that any study head currently
// selects the object. Head-sensitive callers must separately OpenHead and match
// its stage, kind, and digest. Read is the restart-safe counterpart to Open,
// which requires the caller to already possess exact bytes.
func (s *ObjectStore) Read(
	ctx context.Context,
	kind string,
	digest domain.Digest,
) (SemanticObject, ObjectAuthority, error) {
	if s == nil || s.instance == nil {
		return SemanticObject{}, ObjectAuthority{}, refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return SemanticObject{}, ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SemanticObject{}, ObjectAuthority{}, err
	}
	object, err := s.readObjectReference(kind, digest)
	if err != nil {
		return SemanticObject{}, ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return SemanticObject{}, ObjectAuthority{}, err
	}
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return SemanticObject{}, ObjectAuthority{}, err
	}
	return object, s.authority(object), nil
}

func (s *ObjectStore) Validate(ctx context.Context, object SemanticObject, authority ObjectAuthority) error {
	if s == nil || s.instance == nil {
		return refuse(codeInvalidObjectStore, "store is zero or uninitialized", nil)
	}
	s.instance.mu.Lock()
	defer s.instance.mu.Unlock()
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return err
	}
	if authority.storeInstance == nil || authority.storeInstance != s.instance || !sameSemanticObject(authority.object, object) {
		return refuse(codeObjectAuthorityRefused, "authority is not bound to this store and exact object", nil)
	}
	path, shard, err := s.existingObjectPath(object.digest)
	if err != nil {
		return refuse(codeObjectAuthorityRefused, "authority object path is unavailable", err)
	}
	if err := syncDirectory(shard); err != nil {
		return refuse(codeObjectAuthorityRefused, "object-shard sync failed during validation", err)
	}
	reopened, err := reopenExactObject(path, object)
	if err != nil || !sameSemanticObject(reopened, authority.object) {
		return refuse(codeObjectAuthorityRefused, "authority object did not reopen exactly", err)
	}
	if err := s.assertReady(); err != nil {
		return err
	}
	return storeContextRefusal(ctx, codeObjectAuthorityRefused)
}

func (s *ObjectStore) openAtLocked(ctx context.Context, object SemanticObject, path, shard string) (ObjectAuthority, error) {
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return ObjectAuthority{}, err
	}
	if err := syncDirectory(shard); err != nil {
		return ObjectAuthority{}, refuse(codeObjectRefused, "object-shard sync failed before reopen", err)
	}
	reopened, err := reopenExactObject(path, object)
	if err != nil {
		return ObjectAuthority{}, err
	}
	if err := s.assertReady(); err != nil {
		return ObjectAuthority{}, err
	}
	if err := storeContextRefusal(ctx, codeObjectAuthorityRefused); err != nil {
		return ObjectAuthority{}, err
	}
	return s.authority(reopened), nil
}

func (s *ObjectStore) authority(object SemanticObject) ObjectAuthority {
	return ObjectAuthority{storeInstance: s.instance, object: object}
}

func (s *ObjectStore) ensureObjectPath(digest domain.Digest) (string, string, error) {
	hex, err := strictDigestHex(digest)
	if err != nil {
		return "", "", err
	}
	if err := rejectCaseAlias(s.digestRoot, hex[:2]); err != nil {
		return "", "", refuse(codeObjectPublicationFailed, "object shard aliases an existing name", err)
	}
	shard := filepath.Join(s.digestRoot, hex[:2])
	info, created, err := ensurePrivateDirectory(shard)
	if err != nil {
		return "", "", refuse(codeObjectPublicationFailed, "object shard is invalid", err)
	}
	if created {
		if err := syncDirectory(s.digestRoot); err != nil {
			return "", "", refuse(codeObjectPublicationFailed, "digest-root sync failed", err)
		}
	}
	if retained, present := s.shardInfos[shard]; present && !os.SameFile(retained, info) {
		return "", "", refuse(codeObjectPublicationFailed, "object shard changed identity", nil)
	}
	s.shardInfos[shard] = info
	if err := rejectCaseAlias(shard, hex); err != nil {
		return "", "", refuse(codeObjectPublicationFailed, "object aliases an existing name", err)
	}
	return filepath.Join(shard, hex), shard, nil
}

func (s *ObjectStore) existingObjectPath(digest domain.Digest) (string, string, error) {
	hex, err := strictDigestHex(digest)
	if err != nil {
		return "", "", err
	}
	if err := rejectCaseAlias(s.digestRoot, hex[:2]); err != nil {
		return "", "", refuse(codeObjectRefused, "object shard aliases an existing name", err)
	}
	shard := filepath.Join(s.digestRoot, hex[:2])
	if err := assertExactDirectory(shard); err != nil {
		return "", "", refuse(codeObjectRefused, "object shard is absent or invalid", err)
	}
	info, err := exactPrivateDirectoryInfo(shard)
	if err != nil {
		return "", "", refuse(codeObjectRefused, "object shard facts are invalid", err)
	}
	if retained, present := s.shardInfos[shard]; present && !os.SameFile(retained, info) {
		return "", "", refuse(codeObjectRefused, "object shard changed identity", nil)
	}
	s.shardInfos[shard] = info
	if err := rejectCaseAlias(shard, hex); err != nil {
		return "", "", refuse(codeObjectRefused, "object aliases an existing name", err)
	}
	return filepath.Join(shard, hex), shard, nil
}

func strictDigestHex(digest domain.Digest) (string, error) {
	if !digest.Valid() {
		return "", refuse(codeInvalidSemanticObject, "digest is invalid", nil)
	}
	raw := digest.String()
	if len(raw) != len("sha256:")+64 || !strings.HasPrefix(raw, "sha256:") {
		return "", refuse(codeInvalidSemanticObject, "digest is not canonical sha256", nil)
	}
	hex := strings.TrimPrefix(raw, "sha256:")
	for _, character := range hex {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return "", refuse(codeInvalidSemanticObject, "digest contains a non-lowercase-hex character", nil)
		}
	}
	return hex, nil
}

func (s *ObjectStore) assertReady() error {
	if s == nil || s.instance == nil || s.root == "" || s.rootInfo == nil {
		return refuse(codeInvalidObjectStore, "store is zero or incomplete", nil)
	}
	checks := []struct {
		path string
		info os.FileInfo
	}{
		{s.root, s.rootInfo}, {s.objects, s.objectsInfo}, {s.digestRoot, s.digestRootInfo},
		{s.studies, s.studiesInfo}, {s.privateCaptures, s.privateInfo},
	}
	for _, check := range checks {
		current, err := exactPrivateDirectoryInfo(check.path)
		if err != nil || !os.SameFile(check.info, current) {
			return refuse(codeInvalidObjectStore, "owned store directory changed identity or facts", err)
		}
	}
	return nil
}

func ensurePrivateRoot(path string) (os.FileInfo, bool, error) {
	volume := filepath.VolumeName(path)
	remainder := strings.TrimPrefix(path, volume)
	parts := strings.FieldsFunc(remainder, func(r rune) bool { return r == filepath.Separator })
	current := volume + string(filepath.Separator)
	createdRoot := false
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return nil, false, err
			}
			if err := os.Chmod(current, 0o700); err != nil {
				return nil, false, err
			}
			createdRoot = index == len(parts)-1
			info, err = os.Lstat(current)
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, false, errors.New("root path traverses a symlink or non-directory")
		}
		if index == len(parts)-1 {
			if info.Mode().Perm() != 0o700 {
				return nil, false, errors.New("root mode is not 0700")
			}
			return info, createdRoot, nil
		}
	}
	return nil, false, errors.New("root path has no components")
}

func ensurePrivateDirectory(path string) (os.FileInfo, bool, error) {
	info, err := os.Lstat(path)
	created := false
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return nil, false, err
			}
		} else {
			created = true
		}
		if created {
			if err := os.Chmod(path, 0o700); err != nil {
				return nil, false, err
			}
		}
		info, err = os.Lstat(path)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
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

func assertExactDirectory(path string) error {
	_, err := exactPrivateDirectoryInfo(path)
	return err
}

func rejectCaseAlias(directory, expected string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), expected) && entry.Name() != expected {
			return fmt.Errorf("actual entry %q aliases expected %q", entry.Name(), expected)
		}
	}
	return nil
}

func sameSemanticObject(left, right SemanticObject) bool {
	return left.Valid() && right.Valid() && left.kind == right.kind && left.digest == right.digest &&
		bytes.Equal(left.canonical, right.canonical)
}

func writeSyncedTemporary(directory, prefix string, body []byte) (string, error) {
	if len(body) == 0 || len(body) > canon.MaxInputBytes {
		return "", errors.New("temporary body is empty or exceeds the canonical byte bound")
	}
	handle, err := os.CreateTemp(directory, prefix)
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

func reopenExactObject(path string, expected SemanticObject) (SemanticObject, error) {
	expectedBytes := expected.canonical
	if !expected.Valid() {
		return SemanticObject{}, refuse(codeObjectRefused, "expected object is invalid", nil)
	}
	before, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil {
		return SemanticObject{}, refuse(codeObjectRefused, "object path facts are invalid", err)
	}
	handle, err := os.Open(path)
	if err != nil {
		return SemanticObject{}, refuse(codeObjectRefused, "object open failed", err)
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 ||
		opened.Size() != int64(len(expectedBytes)) || !os.SameFile(before, opened) {
		_ = handle.Close()
		return SemanticObject{}, refuse(codeObjectRefused, "opened object disagrees with path facts", statErr)
	}
	afterOpen, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil || !os.SameFile(opened, afterOpen) {
		_ = handle.Close()
		return SemanticObject{}, refuse(codeObjectRefused, "object path changed during open", err)
	}
	body, readErr := io.ReadAll(io.LimitReader(handle, int64(canon.MaxInputBytes)+1))
	closeErr := handle.Close()
	if readErr != nil || closeErr != nil || len(body) != len(expectedBytes) || len(body) > canon.MaxInputBytes {
		return SemanticObject{}, refuse(codeObjectRefused, "object bounded read was incomplete", errors.Join(readErr, closeErr))
	}
	finalInfo, err := exactObjectInfo(path, int64(len(expectedBytes)))
	if err != nil || !os.SameFile(opened, finalInfo) {
		return SemanticObject{}, refuse(codeObjectRefused, "object path changed during read", err)
	}
	if !bytes.Equal(body, expectedBytes) {
		return SemanticObject{}, refuse(codeObjectRefused, "object bytes differ from the exact expected object", nil)
	}
	reopened, err := NewSemanticObject(expected.kind, expected.digest, body)
	if err != nil || !sameSemanticObject(reopened, expected) {
		return SemanticObject{}, refuse(codeObjectRefused, "object failed exact reconstruction", err)
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

func storeContextRefusal(ctx context.Context, code string) error {
	if ctx == nil {
		return refuse(code, "context is nil", nil)
	}
	if err := ctx.Err(); err != nil {
		return refuse(code, "operation context is not live", err)
	}
	return nil
}
