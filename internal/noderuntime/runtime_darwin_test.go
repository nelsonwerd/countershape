//go:build darwin && arm64 && cgo

package noderuntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestC3NodeRuntimeDarwinLiveAdmissionAndRevalidation(t *testing.T) {
	const alias = "/opt/homebrew/bin/node"
	if _, err := os.Lstat(alias); err != nil {
		t.Skipf("exact Node fixture unavailable: %v", err)
	}
	probeParent := c3CanonicalTempDir(t)
	if err := os.Chmod(probeParent, 0o700); err != nil {
		t.Fatal(err)
	}
	retained, err := Admit(context.Background(), alias, probeParent)
	if err != nil || !retained.Valid() {
		t.Fatalf("live Node admission failed: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(alias)
	if err != nil || retained.Path() != resolved {
		t.Fatalf("descriptor path differs: got %q want %q err=%v", retained.Path(), resolved, err)
	}
	fresh, err := retained.Revalidate(context.Background())
	if err != nil || !fresh.Valid() || fresh.Path() != retained.Path() || fresh.ExecutableBytesDigest() != retained.ExecutableBytesDigest() {
		t.Fatalf("live Node revalidation failed: %v", err)
	}
}

func TestC3NodeRuntimeDarwinRejectsNonNodeExecutable(t *testing.T) {
	probeParent := c3CanonicalTempDir(t)
	if err := os.Chmod(probeParent, 0o700); err != nil {
		t.Fatal(err)
	}
	if value, err := Admit(context.Background(), "/bin/sh", probeParent); err == nil || value.Valid() {
		t.Fatalf("non-Node executable was admitted: %v", err)
	}
}

func TestC3NodeRuntimeDarwinRejectsNonPrivateProbeParent(t *testing.T) {
	parent := c3CanonicalTempDir(t)
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if value, err := Admit(context.Background(), "/opt/homebrew/bin/node", parent); err == nil || value.Valid() {
		t.Fatalf("nonprivate probe parent was admitted: %v", err)
	}
}
