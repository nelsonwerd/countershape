//go:build darwin && cgo

package gitobj

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameExclusiveDoesNotOverwriteExistingDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "value"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The destination must remain empty: ordinary Darwin rename is allowed to
	// replace an empty directory, whereas a non-empty directory would refuse the
	// operation even if RENAME_EXCL were accidentally removed.
	if err := renameExclusive(source, destination); err == nil {
		t.Fatal("exclusive rename overwrote an existing destination")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		t.Fatalf("existing destination changed: entries=%d err=%v", len(entries), err)
	}
	if _, err := os.Stat(filepath.Join(source, "value")); err != nil {
		t.Fatalf("failed exclusive rename consumed source: %v", err)
	}
}

func TestPostRenameSyncFailureRollsBackVisibleDestination(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	finalRoot := filepath.Join(root, "candidate")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	syncCalls := 0
	outcome, err := publishStagedDirectory(stage, finalRoot, root, publicationOperations{
		rename: renameExclusive,
		sync: func(string) error {
			syncCalls++
			if syncCalls == 1 {
				return errors.New("injected parent sync failure")
			}
			return nil
		},
	})
	if err == nil || outcome != publicationAbsent || syncCalls != 2 {
		t.Fatalf("rollback outcome=%v syncCalls=%d err=%v", outcome, syncCalls, err)
	}
	if _, err := os.Lstat(finalRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed publication left destination visible: %v", err)
	}
	if _, err := os.Stat(stage); err != nil {
		t.Fatalf("rollback did not restore stage: %v", err)
	}
}

func TestPostRenameRollbackFailureIsTypedAmbiguous(t *testing.T) {
	renames := 0
	outcome, err := publishStagedDirectory("stage", "candidate", "parent", publicationOperations{
		rename: func(string, string) error {
			renames++
			if renames == 1 {
				return nil
			}
			return errors.New("injected rollback failure")
		},
		sync: func(string) error { return errors.New("injected sync failure") },
	})
	if err == nil || outcome != publicationAmbiguous || renames != 2 {
		t.Fatalf("ambiguous outcome=%v renames=%d err=%v", outcome, renames, err)
	}
}

func TestPostRenameRollbackSyncFailureIsTypedAmbiguous(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	finalRoot := filepath.Join(root, "candidate")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	syncCalls := 0
	outcome, err := publishStagedDirectory(stage, finalRoot, root, publicationOperations{
		rename: renameExclusive,
		sync: func(string) error {
			syncCalls++
			return errors.New("injected parent sync failure")
		},
	})
	if err == nil || outcome != publicationAmbiguous || syncCalls != 2 {
		t.Fatalf("rollback-sync outcome=%v syncCalls=%d err=%v", outcome, syncCalls, err)
	}
	if _, statErr := os.Lstat(finalRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rollback did not remove visible destination: %v", statErr)
	}
	if _, statErr := os.Stat(stage); statErr != nil {
		t.Fatalf("rollback did not restore stage: %v", statErr)
	}
}
