//go:build darwin && cgo

package gitobj_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

// Mutation pair: symlink mode cannot enter the regular-file subset.
func TestSymlinkModeIsRejected(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "link", Mode: "120000", Content: []byte("target")}}, "", "symlink")
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_, err := gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t))
	refusalCode(t, err, gitobj.CodeUnsupportedMode)
}

func TestGitlinkModeIsRejected(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	childCommit := commitRef(t, fixture, "refs/heads/child", []gitrepo.File{{Path: "child.txt", Mode: "100644", Content: []byte("child\n")}}, "", "child")
	tree, err := fixture.WriteRawTree(context.Background(), []gitrepo.RawTreeEntry{{Mode: "160000", Name: []byte("submodule"), OID: childCommit}})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := fixture.WriteCommitLiteral(context.Background(), tree)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), "refs/heads/candidate", commit); err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_, err = gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t))
	refusalCode(t, err, gitobj.CodeUnsupportedMode)
}

func TestLFSPointerIsRejected(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	pointer := gitrepo.LFSPointer("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", 1234)
	commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "large.bin", Mode: "100644", Content: pointer}}, "", "lfs")
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_, err := gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t))
	refusalCode(t, err, gitobj.CodeLFSPointer)
}

// Mutation pair: path validation must precede every candidate-path write.
func TestUnsafePathRefusesBeforeAnyCandidateWrite(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	blob, err := fixture.WriteBlob(context.Background(), []byte("hostile\n"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := fixture.WriteRawTree(context.Background(), []gitrepo.RawTreeEntry{{Mode: "100644", Name: []byte(".git"), OID: blob}})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := fixture.WriteCommitLiteral(context.Background(), tree)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), "refs/heads/candidate", commit); err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	target := targetParent(t)
	_, err = gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), target)
	refusalCode(t, err, gitobj.CodeUnsafePath)
	entries, readErr := os.ReadDir(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("unsafe-path refusal left %d target entries", len(entries))
	}
}

func TestInvalidUTF8PathIsRejected(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	blob, err := fixture.WriteBlob(context.Background(), []byte("hostile\n"))
	if err != nil {
		t.Fatal(err)
	}
	tree, err := fixture.WriteRawTree(context.Background(), []gitrepo.RawTreeEntry{{Mode: "100644", Name: []byte{0xff, 'x'}, OID: blob}})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := fixture.WriteCommitLiteral(context.Background(), tree)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), "refs/heads/candidate", commit); err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	_, err = gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), targetParent(t))
	refusalCode(t, err, gitobj.CodeUnsafePath)
}

func TestMaterializationBudgetsRefuseBeforePublication(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{
		{Path: "a.txt", Mode: "100644", Content: []byte("aaaa")},
		{Path: "b.txt", Mode: "100644", Content: []byte("bbbb")},
	}, "", "budgets")
	repository := openRepository(t, fixture, systemGit)
	pinned := pinRef(t, repository, "refs/heads/candidate")
	for _, test := range []struct {
		name   string
		policy gitobj.Policy
	}{
		{name: "entry", policy: policy(t, 1, 16, 8)},
		{name: "single blob", policy: policy(t, 4, 16, 3)},
		{name: "aggregate", policy: policy(t, 4, 7, 4)},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := targetParent(t)
			_, err := gitobj.Inspect(context.Background(), pinned, test.policy, target)
			refusalCode(t, err, gitobj.CodeBudgetExceeded)
			entries, readErr := os.ReadDir(target)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("budget refusal left target state: entries=%d err=%v", len(entries), readErr)
			}
		})
	}
}

// Mutation pair: the executable bit remains part of portable identity.
func TestExecutableModeRemainsIdentityBearing(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/regular", []gitrepo.File{{Path: "run", Mode: "100644", Content: []byte("same\n")}}, "", "regular")
	commitRef(t, fixture, "refs/heads/executable", []gitrepo.File{{Path: "run", Mode: "100755", Content: []byte("same\n")}}, "", "executable")
	repository := openRepository(t, fixture, systemGit)
	regular := pinRef(t, repository, "refs/heads/regular")
	executable := pinRef(t, repository, "refs/heads/executable")
	materializationPolicy := policy(t, 8, 4096, 4096)
	target := targetParent(t)
	regularInspection := inspect(t, regular, materializationPolicy, target)
	executableInspection := inspect(t, executable, materializationPolicy, target)
	regularEntries := regularInspection.Entries()
	executableEntries := executableInspection.Entries()
	if len(regularEntries) != 1 || len(executableEntries) != 1 ||
		regularEntries[0].PortableBlobDigest == executableEntries[0].PortableBlobDigest {
		t.Fatal("100644 and 100755 collapsed to one portable blob identity")
	}
	if regularInspection.PortableTreeDigest() == executableInspection.PortableTreeDigest() {
		t.Fatal("100644 and 100755 collapsed to one portable identity")
	}
	selected, err := gitobj.SelectTrees(regular, executable)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := gitobj.NewCandidateSet(selected, materializationPolicy, regularInspection, executableInspection)
	if err != nil {
		t.Fatalf("mode-distinct candidates should remain admissible: %v", err)
	}
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/executable")
	receipt, err := (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, target)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(receipt.PublishedRoot, "run"))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o755 {
		t.Fatalf("executable materialization mode collapsed: info=%v err=%v", info, err)
	}
}

func TestDuplicateGitTreeAndPortableTreeAreRejected(t *testing.T) {
	t.Run("same Git tree", func(t *testing.T) {
		fixture := fixtureRepository(t, gitrepo.SHA1)
		commit := commitRef(t, fixture, "refs/heads/first", []gitrepo.File{{Path: "same.txt", Mode: "100644", Content: []byte("same\n")}}, "", "same")
		if err := fixture.UpdateRef(context.Background(), "refs/heads/second", commit); err != nil {
			t.Fatal(err)
		}
		repository := openRepository(t, fixture, systemGit)
		_, err := gitobj.SelectTrees(pinRef(t, repository, "refs/heads/first"), pinRef(t, repository, "refs/heads/second"))
		refusalCode(t, err, gitobj.CodeCandidateSetRejected)
	})

	t.Run("same portable tree across object formats", func(t *testing.T) {
		sha1Repository := fixtureRepository(t, gitrepo.SHA1)
		sha256Repository := fixtureRepository(t, gitrepo.SHA256)
		files := []gitrepo.File{{Path: "same.txt", Mode: "100644", Content: []byte("same\n")}}
		commitRef(t, sha1Repository, "refs/heads/candidate", files, "", "same")
		commitRef(t, sha256Repository, "refs/heads/candidate", files, "", "same")
		sha1Pin := pinRef(t, openRepository(t, sha1Repository, systemGit), "refs/heads/candidate")
		sha256Pin := pinRef(t, openRepository(t, sha256Repository, systemGit), "refs/heads/candidate")
		selected, err := gitobj.SelectTrees(sha1Pin, sha256Pin)
		if err != nil {
			t.Fatal(err)
		}
		materializationPolicy := policy(t, 8, 4096, 4096)
		target := targetParent(t)
		first := inspect(t, sha1Pin, materializationPolicy, target)
		second := inspect(t, sha256Pin, materializationPolicy, target)
		if first.PortableTreeDigest() != second.PortableTreeDigest() {
			t.Fatal("cross-format fixture did not produce one portable tree")
		}
		_, err = gitobj.NewCandidateSet(selected, materializationPolicy, first, second)
		refusalCode(t, err, gitobj.CodeCandidateSetRejected)
	})
}

func TestPromisorAndAlternatesRepositoriesAreRejected(t *testing.T) {
	t.Run("promisor", func(t *testing.T) {
		fixture := fixtureRepository(t, gitrepo.SHA1)
		commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("candidate\n")}}, "", "promisor")
		sentinel := filepath.Join(t.TempDir(), "network-sentinel")
		if err := fixture.ConfigurePromisor(context.Background(), "ext::/bin/sh -c ': > "+sentinel+"'"); err != nil {
			t.Fatal(err)
		}
		_, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{GitExecutable: systemGit, Repository: fixture.Root, ScratchRoot: t.TempDir()})
		refusalCode(t, err, gitobj.CodeRepositoryRejected)
		if _, statErr := os.Stat(sentinel); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("promisor refusal touched network sentinel: %v", statErr)
		}
	})

	t.Run("promisor pack marker", func(t *testing.T) {
		fixture := fixtureRepository(t, gitrepo.SHA1)
		commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{{Path: "app.txt", Mode: "100644", Content: []byte("candidate\n")}}, "", "promisor marker")
		if err := fixture.InstallPromisorMarker(); err != nil {
			t.Fatal(err)
		}
		_, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{GitExecutable: systemGit, Repository: fixture.Root, ScratchRoot: t.TempDir()})
		refusalCode(t, err, gitobj.CodeRepositoryRejected)
	})

	t.Run("alternates", func(t *testing.T) {
		fixture := fixtureRepository(t, gitrepo.SHA1)
		if err := fixture.InstallAlternates(t.TempDir()); err != nil {
			t.Fatal(err)
		}
		_, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{GitExecutable: systemGit, Repository: fixture.Root, ScratchRoot: t.TempDir()})
		refusalCode(t, err, gitobj.CodeAlternatesRejected)
	})
}

func TestTargetVolumeAliasesAreRejectedWhenTheVolumeAliases(t *testing.T) {
	for _, test := range []struct {
		name   string
		first  string
		second string
	}{
		{name: "case", first: "Case.txt", second: "case.txt"},
		{name: "Unicode normalization", first: "Caf\u00e9.txt", second: "Cafe\u0301.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := fixtureRepository(t, gitrepo.SHA1)
			commitRef(t, fixture, "refs/heads/candidate", []gitrepo.File{
				{Path: test.first, Mode: "100644", Content: []byte("one")},
				{Path: test.second, Mode: "100644", Content: []byte("two")},
			}, "", "aliases")
			repository := openRepository(t, fixture, systemGit)
			pinned := pinRef(t, repository, "refs/heads/candidate")
			target := targetParent(t)
			probe := filepath.Join(target, test.first)
			if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, aliasErr := os.Stat(filepath.Join(target, test.second))
			if err := os.Remove(probe); err != nil {
				t.Fatal(err)
			}
			_, err := gitobj.Inspect(context.Background(), pinned, policy(t, 8, 4096, 4096), target)
			if aliasErr == nil {
				refusalCode(t, err, gitobj.CodePathCollision)
				return
			}
			if err != nil {
				t.Fatalf("target preserves distinct names but inspection refused: %v", err)
			}
		})
	}
}
