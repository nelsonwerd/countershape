package store

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func semanticObjectForTest(t *testing.T, kind, payload string) SemanticObject {
	t.Helper()
	digest, body, err := canon.DigestTyped(kind, struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Payload       string `json:"payload"`
	}{domain.SchemaVersion, kind, payload})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	object, err := NewSemanticObject(kind, parsed, body)
	if err != nil {
		t.Fatal(err)
	}
	return object
}

// resolvedStoreTestTempDir preserves the store's refusal of symlinked
// ancestors while allowing tests to run under macOS's /var -> /private/var
// default temporary-directory alias. Production callers must still supply an
// already-canonical private root; OpenObjectStore never resolves aliases.
func resolvedStoreTestTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatalf("resolved temporary directory is not clean and absolute: %q", root)
	}
	return root
}

func newObjectStoreForTest(t *testing.T) (*ObjectStore, string) {
	t.Helper()
	root := filepath.Join(resolvedStoreTestTempDir(t), "semantic-store")
	value, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return value, root
}

func TestExternalPublicationPathCannotOverlapOwnedStoreNamespace(t *testing.T) {
	objectStore, root := newObjectStoreForTest(t)
	sibling := filepath.Join(filepath.Dir(root), "contract-output")
	if err := objectStore.ValidateExternalPublicationPath(context.Background(), sibling); err != nil {
		t.Fatalf("disjoint sibling output was refused: %v", err)
	}
	for name, path := range map[string]string{
		"root":       root,
		"descendant": filepath.Join(root, "output"),
		"ancestor":   filepath.Dir(root),
		"relative":   "contract-output",
		"unclean":    filepath.Join(filepath.Dir(root), ".", "contract-output") + string(filepath.Separator) + "..",
	} {
		t.Run(name, func(t *testing.T) {
			if err := objectStore.ValidateExternalPublicationPath(context.Background(), path); err == nil {
				t.Fatalf("overlapping or invalid external path %q was accepted", path)
			}
		})
	}
	caseAlias := strings.Replace(root, string(filepath.Separator)+"Users"+string(filepath.Separator),
		string(filepath.Separator)+"users"+string(filepath.Separator), 1)
	if caseAlias != root {
		aliasInfo, aliasErr := os.Stat(caseAlias)
		if aliasErr == nil && os.SameFile(objectStore.rootInfo, aliasInfo) {
			for name, path := range map[string]string{
				"alias-root":       caseAlias,
				"alias-descendant": filepath.Join(caseAlias, "output"),
				"alias-ancestor":   filepath.Dir(caseAlias),
			} {
				t.Run(name, func(t *testing.T) {
					if err := objectStore.ValidateExternalPublicationPath(context.Background(), path); err == nil {
						t.Fatalf("physical filesystem alias %q was accepted", path)
					}
				})
			}
		}
	}
	symlinkAlias := filepath.Join(filepath.Dir(root), "store-symlink-alias")
	if err := os.Symlink(root, symlinkAlias); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"symlink-alias-root":       symlinkAlias,
		"symlink-alias-descendant": filepath.Join(symlinkAlias, "output"),
	} {
		t.Run(name, func(t *testing.T) {
			if err := objectStore.ValidateExternalPublicationPath(context.Background(), path); err == nil {
				t.Fatalf("deterministic physical store alias %q was accepted", path)
			}
		})
	}
	aliasHost := resolvedStoreTestTempDir(t)
	parentAlias := filepath.Join(aliasHost, "store-parent-symlink-alias")
	if err := os.Symlink(filepath.Dir(root), parentAlias); err != nil {
		t.Fatal(err)
	}
	if err := objectStore.ValidateExternalPublicationPath(context.Background(), parentAlias); err == nil {
		t.Fatalf("deterministic physical store-parent alias %q was accepted", parentAlias)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := objectStore.ValidateExternalPublicationPath(cancelled, sibling); err == nil {
		t.Fatal("cancelled external-path validation succeeded")
	}
}

func TestObjectStoreUsesOneCreateOncePublisherForDistinctSemanticKinds(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	objects := []SemanticObject{
		semanticObjectForTest(t, "FreshConfirmation", "confirmation"),
		semanticObjectForTest(t, "Choicepoint", "choicepoint"),
	}
	for _, object := range objects {
		authority, err := value.Publish(context.Background(), object)
		if err != nil {
			t.Fatal(err)
		}
		if err := value.Validate(context.Background(), object, authority); err != nil {
			t.Fatalf("%s authority refused: %v", object.Kind(), err)
		}
		path, _, err := value.existingObjectPath(object.Digest())
		if err != nil {
			t.Fatal(err)
		}
		hex := strings.TrimPrefix(object.Digest().String(), "sha256:")
		want := filepath.Join(root, objectDirectory, objectAlgorithm, hex[:2], hex)
		if path != want {
			t.Fatalf("object path = %q, want %q", path, want)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			t.Fatalf("object facts = %#v, %v", info, err)
		}
		body, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(body, object.CanonicalBytes()) {
			t.Fatalf("object bytes changed: %v", err)
		}
		second, err := value.Publish(context.Background(), object)
		if err != nil || value.Validate(context.Background(), object, second) != nil {
			t.Fatalf("idempotent publication failed: %v", err)
		}
		after, _ := os.Lstat(path)
		if !os.SameFile(info, after) {
			t.Fatal("idempotent publication replaced the immutable inode")
		}
	}
	for _, path := range []string{root, value.objects, value.digestRoot, value.studies, value.privateCaptures} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("private directory %s = %#v, %v", path, info, err)
		}
	}
}

func TestSemanticObjectRejectsWrongDomainKindDigestAndNoncanonicalBytes(t *testing.T) {
	valid := semanticObjectForTest(t, "Choicepoint", "exact")
	wrong := semanticObjectForTest(t, "FreshConfirmation", "exact")
	if _, err := NewSemanticObject("FreshConfirmation", valid.Digest(), valid.CanonicalBytes()); err == nil {
		t.Fatal("wrong kind domain was accepted")
	}
	if _, err := NewSemanticObject(valid.Kind(), wrong.Digest(), valid.CanonicalBytes()); err == nil {
		t.Fatal("wrong expected digest was accepted")
	}
	noncanonical := append([]byte(" "), valid.CanonicalBytes()...)
	if _, err := NewSemanticObject(valid.Kind(), valid.Digest(), noncanonical); err == nil {
		t.Fatal("noncanonical bytes were accepted")
	}
}

func TestObjectStoreRejectsIntermediateSymlinkAndCaseAlias(t *testing.T) {
	parent := resolvedStoreTestTempDir(t)
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenObjectStore(filepath.Join(link, "nested")); err == nil {
		t.Fatal("intermediate symlink was traversed")
	}

	value, _ := newObjectStoreForTest(t)
	object := semanticObjectForTest(t, "Choicepoint", "alias")
	hex := strings.TrimPrefix(object.Digest().String(), "sha256:")
	upper := strings.ToUpper(hex[:2])
	if upper == hex[:2] {
		t.Skip("digest prefix has no alphabetic character for case-alias probe")
	}
	if err := os.Mkdir(filepath.Join(value.digestRoot, upper), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := value.Publish(context.Background(), object); err == nil {
		t.Fatal("case-alias object shard was accepted")
	}
}

func TestObjectAuthorityIsStoreInstanceBoundAndReopensOnEveryValidation(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	object := semanticObjectForTest(t, "Choicepoint", "authority")
	authority, err := value.Publish(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Validate(context.Background(), object, authority); err == nil {
		t.Fatal("pre-restart object authority survived store-instance change")
	}
	path, _, err := value.existingObjectPath(object.Digest())
	if err != nil {
		t.Fatal(err)
	}
	corrupt := object.CanonicalBytes()
	corrupt[0] = '['
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := value.Validate(context.Background(), object, authority); err == nil {
		t.Fatal("post-issuance corruption survived exact reopen")
	}
}

func TestObjectStoreNeverReplacesExistingDestination(t *testing.T) {
	value, _ := newObjectStoreForTest(t)
	object := semanticObjectForTest(t, "Choicepoint", "create-once")
	if _, err := value.Publish(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	path, _, err := value.existingObjectPath(object.Digest())
	if err != nil {
		t.Fatal(err)
	}
	corrupt := object.CanonicalBytes()
	corrupt[len(corrupt)/2] ^= 1
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.Publish(context.Background(), object); err == nil {
		t.Fatal("publication replaced a corrupt existing destination")
	}
	after, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !bytes.Equal(retained, corrupt) {
		t.Fatal("failed publication changed the existing inode or bytes")
	}
}

func TestObjectStoreReadRehydratesExactObjectWithRestartBoundAuthority(t *testing.T) {
	original, root := newObjectStoreForTest(t)
	object := semanticObjectForTest(t, "Choicepoint", "restart-read")
	originalAuthority, err := original.Publish(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}

	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	reopened, restartedAuthority, err := restarted.Read(context.Background(), object.Kind(), object.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if !sameSemanticObject(reopened, object) {
		t.Fatal("restart read changed the exact semantic object")
	}
	if err := restarted.Validate(context.Background(), reopened, restartedAuthority); err != nil {
		t.Fatalf("restart-bound authority refused: %v", err)
	}
	if err := original.Validate(context.Background(), reopened, restartedAuthority); err == nil {
		t.Fatal("restart-bound authority crossed store-instance boundary")
	}
	if err := restarted.Validate(context.Background(), reopened, originalAuthority); err == nil {
		t.Fatal("pre-restart authority crossed store-instance boundary")
	}
	if _, _, err := restarted.Read(context.Background(), "FreshConfirmation", object.Digest()); err == nil {
		t.Fatal("restart read accepted the wrong semantic kind")
	}
	missing := semanticObjectForTest(t, "Choicepoint", "missing")
	if _, _, err := restarted.Read(context.Background(), missing.Kind(), missing.Digest()); err == nil {
		t.Fatal("restart read accepted an unpublished digest")
	}

	path, _, err := restarted.existingObjectPath(object.Digest())
	if err != nil {
		t.Fatal(err)
	}
	corrupt := object.CanonicalBytes()
	corrupt[len(corrupt)-1] = ']'
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := restarted.Read(context.Background(), object.Kind(), object.Digest()); err == nil {
		t.Fatal("restart read accepted corrupt object bytes")
	}
}
