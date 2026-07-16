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

func newObjectStoreForTest(t *testing.T) (*ObjectStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "semantic-store")
	value, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return value, root
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
	parent := t.TempDir()
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
