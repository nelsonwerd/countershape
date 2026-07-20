//go:build darwin && cgo

package gitobj

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const c3SystemGit = "/usr/bin/git"

func c3ResolvedTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func c3RunGit(t testing.TB, repository string, arguments ...string) {
	t.Helper()
	home := filepath.Join(filepath.Dir(repository), "git-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(context.Background(), c3SystemGit, append([]string{
		"--no-pager", "--no-replace-objects", "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null",
	}, arguments...)...)
	command.Dir = repository
	command.Env = []string{
		"HOME=" + home,
		"TMPDIR=" + home,
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_AUTHOR_NAME=Countershape Fixture",
		"GIT_AUTHOR_EMAIL=fixture@countershape.invalid",
		"GIT_AUTHOR_DATE=2000-01-01T00:00:00Z",
		"GIT_COMMITTER_NAME=Countershape Fixture",
		"GIT_COMMITTER_EMAIL=fixture@countershape.invalid",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z",
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v: %s", arguments[0], err, output)
	}
}

func c3FixtureRepository(t *testing.T, format gitrepo.ObjectFormat) gitrepo.Repository {
	t.Helper()
	fixture, err := gitrepo.Init(context.Background(), c3SystemGit, c3ResolvedTempDir(t), format)
	if errors.Is(err, gitrepo.ErrUnsupportedObjectFormat) {
		t.Skipf("native Git does not support %s object repositories: %v", format, err)
	}
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func c3CommitRef(t *testing.T, fixture gitrepo.Repository, ref string, files []gitrepo.File, parent, message string) string {
	t.Helper()
	commit, err := fixture.CommitFiles(context.Background(), files, parent, message)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), ref, commit); err != nil {
		t.Fatal(err)
	}
	return commit
}

func c3OpenRepository(t *testing.T, root string) Repository {
	t.Helper()
	repository, err := OpenRepository(context.Background(), OpenConfig{
		GitExecutable: c3SystemGit,
		Repository:    root,
		ScratchRoot:   c3ResolvedTempDir(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	})
	return repository
}

func c3PinRef(t *testing.T, repository Repository, ref string) PinnedTree {
	t.Helper()
	pinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	return pinned
}

func c3Policy(t *testing.T, entries int, total, single int64) Policy {
	t.Helper()
	value, err := NewPolicy(entries, total, single)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func c3TargetParent(t *testing.T) string {
	t.Helper()
	parent := filepath.Join(c3ResolvedTempDir(t), "target")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	return parent
}

func c3Inspect(t *testing.T, pinned PinnedTree, policy Policy, target string) InspectedTree {
	t.Helper()
	value, err := Inspect(context.Background(), pinned, policy, target)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func c3SingleTargetFixture(t *testing.T) (gitrepo.Repository, Repository, PinnedTree, Policy, string) {
	t.Helper()
	fixture := c3FixtureRepository(t, gitrepo.SHA1)
	c3CommitRef(t, fixture, "refs/heads/target", []gitrepo.File{
		{Path: "README.txt", Mode: "100644", Content: []byte("pinned target\n")},
		{Path: "bin/run", Mode: "100755", Content: []byte("#!/bin/sh\nexit 0\n")},
	}, "", "target")
	repository := c3OpenRepository(t, fixture.Root)
	pinned := c3PinRef(t, repository, "refs/heads/target")
	materializationPolicy := c3Policy(t, 64, 1<<20, 1<<18)
	parent := c3TargetParent(t)
	return fixture, repository, pinned, materializationPolicy, parent
}

func TestC3SingleTargetMaterializesDirectlyFromInspectedAuthority(t *testing.T) {
	_, repository, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
	source := c3Inspect(t, pinned, materializationPolicy, parent)
	receipt, err := MaterializeSingleTarget(context.Background(), source, parent)
	if err != nil || !receipt.Valid() {
		t.Fatalf("direct single-target publication failed: %v", err)
	}
	if receipt.PublishedRoot != filepath.Join(parent, "candidate") || receipt.ManifestPath != filepath.Join(parent, "materialization.manifest.json") || len(receipt.Entries) != 2 {
		t.Fatalf("single-target receipt differs: %+v", receipt)
	}
	fresh, err := ReopenSingleTarget(context.Background(), source, parent)
	if err != nil || !fresh.Valid() || fresh.ManifestDigest != receipt.ManifestDigest || fresh.PortableTreeDigest != receipt.PortableTreeDigest {
		t.Fatalf("single-target reopen failed: %v", err)
	}
	equivalentFixture := c3FixtureRepository(t, gitrepo.SHA1)
	c3CommitRef(t, equivalentFixture, "refs/heads/target", []gitrepo.File{
		{Path: "README.txt", Mode: "100644", Content: []byte("pinned target\n")},
		{Path: "bin/run", Mode: "100755", Content: []byte("#!/bin/sh\nexit 0\n")},
	}, "", "target")
	equivalentRepository := c3OpenRepository(t, equivalentFixture.Root)
	equivalentPinned := c3PinRef(t, equivalentRepository, "refs/heads/target")
	if repository.Fingerprint() == equivalentRepository.Fingerprint() ||
		pinned.IdentityDigest() != equivalentPinned.IdentityDigest() ||
		pinned.Provenance().CommitOID != equivalentPinned.Provenance().CommitOID ||
		pinned.Provenance().TreeOID != equivalentPinned.Provenance().TreeOID {
		t.Fatal("content-portable tree identity did not survive an independently created exact object database")
	}
}

func TestC3SingleTargetMovingRefCannotChangePinnedBytes(t *testing.T) {
	fixture, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
	source := c3Inspect(t, pinned, materializationPolicy, parent)
	c3CommitRef(t, fixture, "refs/heads/target", []gitrepo.File{{Path: "README.txt", Mode: "100644", Content: []byte("moved ref\n")}}, "", "moved")
	receipt, err := MaterializeSingleTarget(context.Background(), source, parent)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(receipt.PublishedRoot, "README.txt"))
	if err != nil || !bytes.Equal(body, []byte("pinned target\n")) {
		t.Fatalf("moving ref changed pinned bytes: %q err=%v", body, err)
	}
}

func TestC3SingleTargetExcludesDirtyWorktreeBytes(t *testing.T) {
	root := c3ResolvedTempDir(t)
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o700); err != nil {
		t.Fatal(err)
	}
	c3RunGit(t, worktree, "init", "--object-format=sha1", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(worktree, "contract.test.mjs"), []byte("export const value = 'committed';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "support.json"), []byte("{\"source\":\"committed\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c3RunGit(t, worktree, "add", "--", "contract.test.mjs", "support.json")
	c3RunGit(t, worktree, "commit", "-m", "pinned target")
	repository := c3OpenRepository(t, worktree)
	pinned := c3PinRef(t, repository, "refs/heads/main")
	parent := c3TargetParent(t)
	source := c3Inspect(t, pinned, c3Policy(t, 8, 1<<20, 1<<18), parent)
	if err := os.WriteFile(filepath.Join(worktree, "contract.test.mjs"), []byte("export const value = 'dirty';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "untracked-secret.txt"), []byte("dirty untracked bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := MaterializeSingleTarget(context.Background(), source, parent)
	if err != nil || !receipt.Valid() {
		t.Fatalf("dirty-worktree exclusion publication failed: %v", err)
	}
	entrypoint, err := os.ReadFile(filepath.Join(receipt.PublishedRoot, "contract.test.mjs"))
	if err != nil || !bytes.Equal(entrypoint, []byte("export const value = 'committed';\n")) {
		t.Fatalf("materialized entrypoint did not come from pinned object bytes: %q err=%v", entrypoint, err)
	}
	if _, err := os.Lstat(filepath.Join(receipt.PublishedRoot, "untracked-secret.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("untracked worktree bytes entered the target: %v", err)
	}
}

func TestC3SingleTargetAmbiguityRequiresExplicitReopen(t *testing.T) {
	t.Run("candidate remains visible", func(t *testing.T) {
		_, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
		source := c3Inspect(t, pinned, materializationPolicy, parent)
		syncFailure := errors.New("injected single-target parent sync failure")
		rollbackFailure := errors.New("injected single-target rollback failure")
		renameCalls := 0
		receipt, err := materializeSingleTarget(context.Background(), source, parent, publicationOperations{
			rename: func(from, to string) error {
				renameCalls++
				if renameCalls == 1 {
					return renameExclusive(from, to)
				}
				return rollbackFailure
			},
			sync: func(string) error { return syncFailure },
		})
		code, classified := RefusalCodeOf(err)
		if receipt.Valid() || !classified || code != CodePublicationAmbiguous ||
			!errors.Is(err, syncFailure) || !errors.Is(err, rollbackFailure) || renameCalls != 2 {
			t.Fatalf("ambiguous publication facts differ: receipt=%#v code=%q classified=%v renames=%d err=%v", receipt, code, classified, renameCalls, err)
		}
		entries, readErr := os.ReadDir(parent)
		if readErr != nil || len(entries) != 2 || entries[0].Name() != "candidate" || entries[1].Name() != materializationManifestFilename {
			t.Fatalf("ambiguous publication did not retain exactly the reconcilable candidate and manifest: entries=%v err=%v", entries, readErr)
		}
		reopened, reopenErr := ReopenSingleTarget(context.Background(), source, parent)
		if reopenErr != nil || !reopened.Valid() {
			t.Fatalf("explicit full reopen did not reconcile the retained target: %v", reopenErr)
		}
	})

	t.Run("rollback absence is not durable", func(t *testing.T) {
		_, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
		source := c3Inspect(t, pinned, materializationPolicy, parent)
		publishSyncFailure := errors.New("injected post-publication parent sync failure")
		rollbackSyncFailure := errors.New("injected rollback parent sync failure")
		syncCalls := 0
		receipt, err := materializeSingleTarget(context.Background(), source, parent, publicationOperations{
			rename: renameExclusive,
			sync: func(string) error {
				syncCalls++
				if syncCalls == 1 {
					return publishSyncFailure
				}
				return rollbackSyncFailure
			},
		})
		code, classified := RefusalCodeOf(err)
		if receipt.Valid() || !classified || code != CodePublicationAmbiguous ||
			!errors.Is(err, publishSyncFailure) || !errors.Is(err, rollbackSyncFailure) || syncCalls != 2 {
			t.Fatalf("rollback-sync ambiguity facts differ: receipt=%#v code=%q classified=%v syncs=%d err=%v", receipt, code, classified, syncCalls, err)
		}
		entries, readErr := os.ReadDir(parent)
		if readErr != nil || len(entries) != 2 || entries[1].Name() != materializationManifestFilename ||
			!strings.HasPrefix(entries[0].Name(), ".countershape-stage-") {
			t.Fatalf("rollback-sync ambiguity did not retain exact stage and manifest: entries=%v err=%v", entries, readErr)
		}
		if reopened, reopenErr := ReopenSingleTarget(context.Background(), source, parent); reopenErr == nil || reopened.Valid() {
			t.Fatalf("candidate-absent ambiguity falsely reopened authority: %v", reopenErr)
		}
	})
}

func TestC3SingleTargetReopenRefusesEveryPublishedMutation(t *testing.T) {
	tests := map[string]func(*testing.T, MaterializationReceipt){
		"missing leaf": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.Remove(filepath.Join(receipt.PublishedRoot, "README.txt")); err != nil {
				t.Fatal(err)
			}
		},
		"file bytes": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.WriteFile(filepath.Join(receipt.PublishedRoot, "README.txt"), []byte("changed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"file mode": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.Chmod(filepath.Join(receipt.PublishedRoot, "README.txt"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"executable bytes": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.WriteFile(filepath.Join(receipt.PublishedRoot, "bin", "run"), []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"executable mode": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.Chmod(filepath.Join(receipt.PublishedRoot, "bin", "run"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"nested directory mode": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.Chmod(filepath.Join(receipt.PublishedRoot, "bin"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"nested directory symlink": func(t *testing.T, receipt MaterializationReceipt) {
			bin := filepath.Join(receipt.PublishedRoot, "bin")
			retained := filepath.Join(t.TempDir(), "retained-bin")
			if err := os.Rename(bin, retained); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(retained, bin); err != nil {
				t.Fatal(err)
			}
		},
		"symbolic leaf": func(t *testing.T, receipt MaterializationReceipt) {
			path := filepath.Join(receipt.PublishedRoot, "README.txt")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("bin/run", path); err != nil {
				t.Fatal(err)
			}
		},
		"manifest": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.WriteFile(receipt.ManifestPath, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"extra target member": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.WriteFile(filepath.Join(receipt.PublishedRoot, "extra"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"replaced target root": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.RemoveAll(receipt.PublishedRoot); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(receipt.PublishedRoot, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"extra parent member": func(t *testing.T, receipt MaterializationReceipt) {
			if err := os.WriteFile(filepath.Join(filepath.Dir(receipt.PublishedRoot), "extra"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
			source := c3Inspect(t, pinned, materializationPolicy, parent)
			receipt, err := MaterializeSingleTarget(context.Background(), source, parent)
			if err != nil {
				t.Fatal(err)
			}
			mutate(t, receipt)
			if reopened, err := ReopenSingleTarget(context.Background(), source, parent); err == nil || reopened.Valid() {
				t.Fatalf("published mutation retained authority: %v", err)
			}
		})
	}
	t.Run("wrong private parent", func(t *testing.T) {
		_, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
		source := c3Inspect(t, pinned, materializationPolicy, parent)
		if _, err := MaterializeSingleTarget(context.Background(), source, parent); err != nil {
			t.Fatal(err)
		}
		wrongParent := c3TargetParent(t)
		if reopened, err := ReopenSingleTarget(context.Background(), source, wrongParent); err == nil || reopened.Valid() {
			t.Fatalf("wrong private parent reopened authority: %v", err)
		}
	})
}

func TestC3SingleTargetRefusesUnsupportedTreeFormsBeforePublication(t *testing.T) {
	fixture := c3FixtureRepository(t, gitrepo.SHA1)
	c3CommitRef(t, fixture, "refs/heads/target", []gitrepo.File{{Path: "link", Mode: "120000", Content: []byte("target")}}, "", "symlink")
	repository := c3OpenRepository(t, fixture.Root)
	pinned := c3PinRef(t, repository, "refs/heads/target")
	parent := c3TargetParent(t)
	if source, err := Inspect(context.Background(), pinned, c3Policy(t, 8, 1<<20, 1<<18), parent); err == nil || source.Valid() {
		t.Fatalf("unsupported tree produced inspected authority: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatalf("refusal left publication effects: entries=%v err=%v", entries, err)
	}
}

func TestC3SingleTargetCannotPublishTwiceIntoOnePrivateParent(t *testing.T) {
	_, _, pinned, materializationPolicy, parent := c3SingleTargetFixture(t)
	source := c3Inspect(t, pinned, materializationPolicy, parent)
	if _, err := MaterializeSingleTarget(context.Background(), source, parent); err != nil {
		t.Fatal(err)
	}
	if receipt, err := MaterializeSingleTarget(context.Background(), source, parent); err == nil || receipt.Valid() {
		t.Fatalf("second materialization overwrote exact publication: %v", err)
	}
}
