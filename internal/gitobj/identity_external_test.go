//go:build darwin && cgo

package gitobj_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

func TestNativeGitObjectFormatsAreRehashed(t *testing.T) {
	for _, format := range []gitrepo.ObjectFormat{gitrepo.SHA1, gitrepo.SHA256} {
		t.Run(string(format), func(t *testing.T) {
			fixture := fixtureRepository(t, format)
			commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("format\n")}}, "", "format")
			commitRef(t, fixture, "refs/heads/control", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("control\n")}}, "", "control")
			repository := openRepository(t, fixture, systemGit)
			pinned := pinRef(t, repository, "refs/heads/candidate")
			control := pinRef(t, repository, "refs/heads/control")
			materializationPolicy := policy(t, 8, 4096, 4096)
			target := targetParent(t)
			inspected := inspect(t, pinned, materializationPolicy, target)
			if !inspected.Valid() || len(inspected.Entries()) != 1 {
				t.Fatalf("%s inspection is incomplete", format)
			}
			want := gitobj.ObjectSHA1
			if format == gitrepo.SHA256 {
				want = gitobj.ObjectSHA256
			}
			if repository.ObjectFormat() != want {
				t.Fatalf("got object format %s, want %s", repository.ObjectFormat(), want)
			}
			selected, err := gitobj.SelectTrees(pinned, control)
			if err != nil {
				t.Fatal(err)
			}
			declaration, err := gitobj.InspectSelected(context.Background(), selected, materializationPolicy, target)
			if err != nil {
				t.Fatal(err)
			}
			bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/candidate")
			receipt, err := (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, target)
			if err != nil {
				t.Fatal(err)
			}
			payload, err := os.ReadFile(filepath.Join(receipt.PublishedRoot, "app.txt"))
			if err != nil || string(payload) != "format\n" || !receipt.Valid() || receipt.ObjectFormat != want {
				t.Fatalf("%s materialization is incomplete: payload=%q receipt=%+v err=%v", format, payload, receipt, err)
			}
		})
	}
}

func TestSHA256FixtureDoesNotRelabelArbitraryToolFailureAsUnsupported(t *testing.T) {
	root := t.TempDir()
	broken := filepath.Join(root, "broken-git")
	if err := os.WriteFile(broken, []byte("#!/bin/sh\nexit 72\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := gitrepo.Init(context.Background(), broken, root, gitrepo.SHA256)
	if err == nil || errors.Is(err, gitrepo.ErrUnsupportedObjectFormat) {
		t.Fatalf("arbitrary Git failure was mislabeled as unsupported SHA-256: %v", err)
	}
}

func TestMovingRefDoesNotChangePinnedIdentity(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	firstCommit := commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("first\n")}}, "", "first")
	repository := openRepository(t, fixture, systemGit)
	pinnedFirst := pinRef(t, repository, "refs/heads/candidate")
	firstIdentity := pinnedFirst.IdentityDigest()

	secondCommit := commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("second\n")}}, firstCommit, "second")
	if secondCommit == firstCommit {
		t.Fatal("moving-ref fixture did not move")
	}
	oldInspection := inspect(t, pinnedFirst, policy(t, 8, 4096, 4096), targetParent(t))
	pinnedSecond := pinRef(t, repository, "refs/heads/candidate")
	newInspection := inspect(t, pinnedSecond, policy(t, 8, 4096, 4096), targetParent(t))
	if pinnedFirst.IdentityDigest() != firstIdentity || pinnedFirst.Provenance().CommitOID != firstCommit {
		t.Fatal("stored pin changed after the display ref moved")
	}
	if oldInspection.PortableTreeDigest() == newInspection.PortableTreeDigest() {
		t.Fatal("fixture trees did not preserve their distinct bytes")
	}
}

func TestRepositoryCloseRevokesEveryDerivedExecutionCapability(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/first", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("first\n")}}, "", "first")
	commitRef(t, fixture, "refs/heads/second", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("second\n")}}, "", "second")
	repository := openRepository(t, fixture, systemGit)
	first := pinRef(t, repository, "refs/heads/first")
	second := pinRef(t, repository, "refs/heads/second")
	selected, err := gitobj.SelectTrees(first, second)
	if err != nil {
		t.Fatal(err)
	}
	materializationPolicy := policy(t, 8, 4096, 4096)
	declaration, err := gitobj.InspectSelected(context.Background(), selected, materializationPolicy, targetParent(t))
	if err != nil {
		t.Fatal(err)
	}
	bound := bindCandidates(t, declaration, materializationPolicy)
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	if repository.Valid() || first.Valid() || second.Valid() || selected.Valid() || declaration.Valid() {
		t.Fatal("repository close did not revoke every derived source capability")
	}
	for _, candidate := range bound {
		if candidate.Valid() {
			t.Fatal("closed repository left a bound materialization capability valid")
		}
	}
}

// Mutation pair: replacement objects must remain disabled during pinning.
func TestReplacementRefsNeverAffectPinnedIdentity(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	original := commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("original\n")}}, "", "original")
	replacement := commitRef(t, fixture, "refs/heads/replacement", []gitrepo.File{{Path: "choice.txt", Mode: "100644", Content: []byte("replacement\n")}}, "", "replacement")
	if err := fixture.InstallReplacement(context.Background(), original, replacement); err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	if pinned.Provenance().CommitOID != original {
		t.Fatalf("replacement changed selected commit: got %s, want %s", pinned.Provenance().CommitOID, original)
	}
	if _, err := gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t)); err != nil {
		t.Fatalf("replacement-disabled source should inspect: %v", err)
	}
}

// Mutation pair: every Git plumbing process must disable lazy fetching.
func TestLazyFetchEnvironmentIsDisabled(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("offline\n")}}, "", "offline")
	wrapperRoot := t.TempDir()
	logPath := filepath.Join(wrapperRoot, "environment.log")
	wrapper, err := gitrepo.WriteGitWrapper(wrapperRoot, systemGit, logPath, false)
	if err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, wrapper)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_ = inspect(t, pinned, policy(t, 8, 4096, 4096), targetParent(t))
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(logBytes))
	if len(lines) == 0 {
		t.Fatal("Git wrapper observed no commands")
	}
	for _, line := range lines {
		if line != "1" {
			t.Fatalf("Git command observed GIT_NO_LAZY_FETCH=%q", line)
		}
	}
}

// Mutation pair: streamed blob bytes must be independently rehashed.
func TestBlobRehashRejectsTamperedGitStream(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("alpha\n")}}, "", "tamper")
	wrapperRoot := t.TempDir()
	wrapper, err := gitrepo.WriteGitWrapper(wrapperRoot, systemGit, filepath.Join(wrapperRoot, "environment.log"), true)
	if err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, wrapper)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_, err = gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t))
	refusalCode(t, err, gitobj.CodeObjectHashMismatch)
}

func TestCommitAndTreeRehashRejectTamperedGitStreams(t *testing.T) {
	for _, objectType := range []string{"commit", "tree"} {
		t.Run(objectType, func(t *testing.T) {
			fixture := fixtureRepository(t, gitrepo.SHA1)
			commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "nested/app.txt", Mode: "100644", Content: []byte("alpha\n")}}, "", objectType)
			wrapperRoot := t.TempDir()
			wrapper, err := gitrepo.WriteGitObjectTamperWrapper(
				wrapperRoot, systemGit, filepath.Join(wrapperRoot, "environment.log"), objectType,
			)
			if err != nil {
				t.Fatal(err)
			}
			repository := openRepository(t, fixture, wrapper)
			_, err = repository.Pin(context.Background(), "refs/heads/candidate")
			refusalCode(t, err, gitobj.CodeObjectHashMismatch)
		})
	}
}
