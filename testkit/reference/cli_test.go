package reference

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
)

const (
	cliTestGit  = "/usr/bin/git"
	cliTestNode = "/opt/homebrew/bin/node"
)

var exactCLIRefAuthority = map[string]struct {
	commit string
	tree   string
}{
	"refs/heads/main":         {commit: "e628cce8a2ff0fc291d30912d286ece6f8aaa6b8", tree: "608532e0573a7296d435b9255bba0386bd5d2079"},
	"refs/heads/config-first": {commit: "57cfa28aefb73707f14de730ec71a280ac158188", tree: "59ca6f56e5eeb54b3ef73e43d62cef33ef63f6f4"},
	"refs/heads/env-first":    {commit: "8fe3cd62f5d96137eb18c0888f48ef5bccf854e8", tree: "5e2a9db04d9c649fb996545aa0f93719f5efb8e0"},
	"refs/heads/argv-first":   {commit: "d6ea4ec0b45e6aad38c22ece698f7e2deeb59482", tree: "d8547c2ba9632c8cc5056e4ff5c62ce6da7fd630"},
}

func TestCLIDriverCreatesDeterministicCleanParentlessRepository(t *testing.T) {
	roots := []string{prepareCLIFixture(t, 1), prepareCLIFixture(t, 2), prepareCLIFixture(t, 3)}
	for _, root := range roots {
		if status := cliGitBytes(t, root, nil, "status", "--porcelain=v2", "-z", "--untracked-files=all"); len(status) != 0 {
			t.Fatalf("fixture %s is dirty: %q", root, status)
		}
		if got := strings.TrimSpace(string(cliGitBytes(t, root, nil, "rev-list", "--count", "HEAD"))); got != "1" {
			t.Fatalf("HEAD commit count = %q, want 1", got)
		}
		head := cliGitLine(t, root, nil, "rev-parse", "HEAD")
		if got := strings.Fields(string(cliGitBytes(t, root, nil, "rev-list", "--parents", "-n", "1", "HEAD"))); len(got) != 1 || got[0] != head {
			t.Fatalf("HEAD is not parentless: %q", got)
		}
		assertCLIDriverTrackedAuthority(t, root)
		for ref, want := range exactCLIRefAuthority {
			if got := cliGitLine(t, root, nil, "rev-parse", ref); got != want.commit {
				t.Fatalf("%s commit = %s, want %s", ref, got, want.commit)
			}
			if got := cliGitLine(t, root, nil, "rev-parse", ref+"^{tree}"); got != want.tree {
				t.Fatalf("%s tree = %s, want %s", ref, got, want.tree)
			}
		}
	}
	for ref := range exactCLIRefAuthority {
		firstCommit := cliGitLine(t, roots[0], nil, "rev-parse", ref)
		firstTree := cliGitLine(t, roots[0], nil, "rev-parse", ref+"^{tree}")
		for _, root := range roots[1:] {
			if got := cliGitLine(t, root, nil, "rev-parse", ref); got != firstCommit {
				t.Fatalf("deterministic %s commit drifted: %s != %s", ref, firstCommit, got)
			}
			if got := cliGitLine(t, root, nil, "rev-parse", ref+"^{tree}"); got != firstTree {
				t.Fatalf("deterministic %s tree drifted: %s != %s", ref, firstTree, got)
			}
		}
	}
}

func TestCLIFixtureDefensiveAccessAndPinnedRefSurvivesMove(t *testing.T) {
	root := prepareCLIFixture(t, 1)
	fixture := openCLIFixture(t, root)
	defer fixture.Close()

	if !fixture.Valid() || fixture.Root() != root || fixture.Entrypoint() != clifixture.Entrypoint {
		t.Fatalf("opened fixture identity is incomplete: valid=%t root=%q entrypoint=%q", fixture.Valid(), fixture.Root(), fixture.Entrypoint())
	}
	if repositories := fixture.Repositories(); len(repositories) != 1 || repositories[0].Fingerprint() != fixture.Repository().Fingerprint() {
		t.Fatalf("default repository roster = %v, want one primary capability", repositories)
	}
	roles := fixture.Roles()
	roles[0] = ArgvFirst
	if got := fixture.Roles()[0]; got != ConfigFirst {
		t.Fatalf("role roster was not defensive: %q", got)
	}
	refs := fixture.Refs()
	refs[ConfigFirst] = "refs/heads/argv-first"
	ref, err := fixture.Ref(ConfigFirst)
	if err != nil || ref != "refs/heads/config-first" {
		t.Fatalf("ref map was not defensive: ref=%q err=%v", ref, err)
	}
	if _, err := fixture.Ref(CLIRole("unknown")); err == nil {
		t.Fatal("unknown role was accepted")
	}

	repository := fixture.Repository()
	pinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	before := pinned.Provenance()
	replacement := cliGitLine(t, root, nil, "rev-parse", "refs/heads/argv-first")
	cliGitBytes(t, root, nil, "update-ref", ref, replacement)
	if after := pinned.Provenance(); after != before || !pinned.Valid() {
		t.Fatalf("pinned candidate changed after display-ref move: before=%+v after=%+v", before, after)
	}
	repinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if repinned.Provenance().CommitOID != replacement || repinned.Provenance().TreeOID == before.TreeOID {
		t.Fatalf("new pin did not observe moved ref: old=%+v new=%+v", before, repinned.Provenance())
	}
}

func TestCLIFixtureAdmitsBoundedIndependentRepositoryRoster(t *testing.T) {
	root := prepareCLIFixture(t, 1)
	fixture, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
		Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
		RepositoryCount: maxCLIRepositories,
	})
	if err != nil {
		t.Fatal(err)
	}
	repositories := fixture.Repositories()
	if !fixture.Valid() || len(repositories) != maxCLIRepositories || fixture.Repository().Fingerprint() != repositories[0].Fingerprint() {
		t.Fatalf("repository roster admission = valid:%t count:%d", fixture.Valid(), len(repositories))
	}
	fingerprint := repositories[0].Fingerprint()
	format := repositories[0].ObjectFormat()
	for index, repository := range repositories {
		if !repository.Valid() || repository.Fingerprint() != fingerprint || repository.ObjectFormat() != format {
			t.Fatalf("repository %d authority differs from the admitted roster", index)
		}
	}
	defensive := fixture.Repositories()
	defensive[0] = gitobj.Repository{}
	if got := fixture.Repositories(); len(got) != maxCLIRepositories || !got[0].Valid() || got[0].Fingerprint() != fingerprint {
		t.Fatalf("repository roster was not defensive: %v", got)
	}
	ref, err := fixture.Ref(ArgvFirst)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := repositories[0].Pin(context.Background(), ref)
	if err != nil || !pinned.Valid() {
		t.Fatalf("pin primary roster capability: valid=%t err=%v", pinned.Valid(), err)
	}
	if err := repositories[1].Close(); err != nil {
		t.Fatal(err)
	}
	if fixture.Valid() || !repositories[0].Valid() || !repositories[2].Valid() || fixture.Repositories() != nil {
		t.Fatal("closing one independent capability did not poison only the fixture roster")
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	for index, repository := range repositories {
		if repository.Valid() {
			t.Fatalf("repository %d survived fixture close", index)
		}
	}
	if pinned.Valid() {
		t.Fatal("pinned tree survived fixture roster close")
	}

	for _, count := range []int{-1, maxCLIRepositories + 1} {
		if _, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
			Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
			RepositoryCount: count,
		}); !errors.Is(err, ErrCLIFixtureMalformed) {
			t.Fatalf("repository count %d error = %v", count, err)
		}
	}
}

func TestCLIFixtureRefusesMalformedCrossDomainAndSharedRoot(t *testing.T) {
	t.Run("shared root", func(t *testing.T) {
		root := prepareCLIFixture(t, 1)
		first := openCLIFixture(t, root)
		defer first.Close()
		_, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
			Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
		})
		if !errors.Is(err, ErrCLIFixtureSharedRoot) {
			t.Fatalf("shared root error = %v", err)
		}
	})

	t.Run("cross domain", func(t *testing.T) {
		root := prepareCLIFixture(t, 1)
		manifest := filepath.Join(root, filepath.FromSlash(cliManifestPath))
		foreign := strings.Replace(cliManifest, `"domain":"cli"`, `"domain":"other"`, 1)
		if err := os.WriteFile(manifest, []byte(foreign), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
			Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
		})
		if !errors.Is(err, ErrCLIFixtureDomain) {
			t.Fatalf("cross-domain error = %v", err)
		}
	})

	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "malformed marker", mutate: func(t *testing.T, root string) {
			writeCLIFile(t, filepath.Join(root, filepath.FromSlash(cliManifestPath)), []byte("{"), 0o600)
		}},
		{name: "missing candidate ref", mutate: func(t *testing.T, root string) {
			cliGitBytes(t, root, nil, "update-ref", "-d", "refs/heads/argv-first")
		}},
		{name: "extra ref", mutate: func(t *testing.T, root string) {
			cliGitBytes(t, root, nil, "update-ref", "refs/heads/extra", cliGitLine(t, root, nil, "rev-parse", "HEAD"))
		}},
		{name: "packed refs", mutate: func(t *testing.T, root string) {
			cliGitBytes(t, root, nil, "pack-refs", "--all")
		}},
		{name: "parented same-tree candidate", mutate: func(t *testing.T, root string) {
			ref := "refs/heads/config-first"
			original := cliGitLine(t, root, nil, "rev-parse", ref)
			tree := cliGitLine(t, root, nil, "rev-parse", ref+"^{tree}")
			parented := cliGitLine(t, root, nil, "commit-tree", tree, "-p", original, "-m", "Countershape U7 CLI candidate: config-first")
			cliGitBytes(t, root, nil, "update-ref", ref, parented)
		}},
		{name: "wrong candidate message", mutate: func(t *testing.T, root string) {
			ref := "refs/heads/config-first"
			tree := cliGitLine(t, root, nil, "rev-parse", ref+"^{tree}")
			commit := cliGitLine(t, root, nil, "commit-tree", tree, "-m", "wrong message")
			cliGitBytes(t, root, nil, "update-ref", ref, commit)
		}},
		{name: "candidate executable mode drift", mutate: func(t *testing.T, root string) {
			mutateCLICandidateTree(t, root, ConfigFirst, "100644", false, false)
		}},
		{name: "candidate symbolic link", mutate: func(t *testing.T, root string) {
			mutateCLICandidateTree(t, root, ConfigFirst, "100755", true, false)
		}},
		{name: "candidate extra file", mutate: func(t *testing.T, root string) {
			mutateCLICandidateTree(t, root, ConfigFirst, "100755", false, true)
		}},
		{name: "untracked worktree input", mutate: func(t *testing.T, root string) {
			writeCLIFile(t, filepath.Join(root, "shared-state.json"), []byte("{}"), 0o600)
		}},
		{name: "extra worktree directory", mutate: func(t *testing.T, root string) {
			if err := os.Mkdir(filepath.Join(root, "empty-extra"), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "worktree symbolic link", mutate: func(t *testing.T, root string) {
			path := filepath.Join(root, "authority", "cli", string(ConfigFirst), "candidate-role.json")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, ".gitignore"), path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symbolic candidate ref", mutate: func(t *testing.T, root string) {
			path := filepath.Join(root, ".git", "refs", "heads", string(ConfigFirst))
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(string(ArgvFirst), path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "worktree mode drift", mutate: func(t *testing.T, root string) {
			if err := os.Chmod(filepath.Join(root, ".gitignore"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "config drift", mutate: func(t *testing.T, root string) {
			writeCLIFile(t, filepath.Join(root, ".git", "config"), []byte(cliGitConfig+"\n"), 0o600)
		}},
		{name: "wrong symbolic HEAD", mutate: func(t *testing.T, root string) {
			writeCLIFile(t, filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/config-first\n"), 0o600)
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			root := prepareCLIFixture(t, 1)
			test.mutate(t, root)
			assertMalformedCLIFixture(t, root)
		})
	}

	for _, mutation := range []struct {
		name string
		path func(string) string
		mode os.FileMode
		make bool
	}{
		{name: "special fixture root", path: func(root string) string { return root }, mode: 0o700 | os.ModeSticky},
		{name: "special nested directory", path: func(root string) string { return filepath.Join(root, "authority") }, mode: 0o700 | os.ModeSetgid},
		{name: "special tracked regular", path: func(root string) string { return filepath.Join(root, ".gitignore") }, mode: 0o600 | os.ModeSetuid},
		{name: "special tracked executable", path: func(root string) string {
			return filepath.Join(root, "authority", "cli", string(ConfigFirst), CLIEntrypoint)
		}, mode: 0o700 | os.ModeSticky},
		{name: "special ignored binary", path: func(root string) string { return filepath.Join(root, cliIgnoredBinary) }, mode: 0o700 | os.ModeSetgid, make: true},
	} {
		mutation := mutation
		t.Run(mutation.name, func(t *testing.T) {
			root := prepareCLIFixture(t, 1)
			path := mutation.path(root)
			if mutation.make {
				writeCLIFile(t, path, []byte("binary"), 0o700)
			}
			if err := os.Chmod(path, mutation.mode); err != nil {
				t.Fatal(err)
			}
			assertMalformedCLIFixture(t, root)
		})
	}
}

func TestCLIFixtureConcurrentCopiesCloseAndReopen(t *testing.T) {
	root := prepareCLIFixture(t, 1)
	fixture := openCLIFixture(t, root)
	copies := make([]CLIFixture, 8)
	for index := range copies {
		copies[index] = fixture
	}
	start := make(chan struct{})
	failures := make(chan error, len(copies)*32)
	var wait sync.WaitGroup
	for index, copy := range copies {
		index, copy := index, copy
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			if index%3 == 0 {
				if err := copy.Close(); err != nil {
					failures <- fmt.Errorf("close copy %d: %w", index, err)
				}
				return
			}
			for iteration := 0; iteration < 32; iteration++ {
				_ = copy.Valid()
				if observed := copy.Root(); observed != "" && observed != root {
					failures <- fmt.Errorf("copy %d root = %q", index, observed)
				}
				if observed := copy.Entrypoint(); observed != "" && observed != CLIEntrypoint {
					failures <- fmt.Errorf("copy %d entrypoint = %q", index, observed)
				}
				if roles := copy.Roles(); roles != nil && len(roles) != len(CLIRoles()) {
					failures <- fmt.Errorf("copy %d roles = %v", index, roles)
				}
				if refs := copy.Refs(); refs != nil && len(refs) != len(CLIRoles()) {
					failures <- fmt.Errorf("copy %d refs = %v", index, refs)
				}
				ref, err := copy.Ref(ConfigFirst)
				if err == nil && ref != "refs/heads/config-first" {
					failures <- fmt.Errorf("copy %d ref = %q", index, ref)
				}
				_ = copy.Repository().Valid()
			}
		}()
	}
	close(start)
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if fixture.Valid() || fixture.Root() != "" {
		t.Fatal("fixture remained live after concurrent close")
	}

	reopened, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
		Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
	})
	if err != nil || !reopened.Valid() {
		t.Fatalf("reopen after concurrent close: valid=%t err=%v", reopened.Valid(), err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCLIDriverRequiresExactPrepareInvocationAndNoSideChannel(t *testing.T) {
	driver := cliDriverPath(t)
	info, err := os.Lstat(driver)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
		t.Fatalf("driver mode = %v, want regular 0644", info.Mode())
	}

	for _, arguments := range [][]string{
		{"--prepare", "--domain", "other", "--ordinal", "1", "--fixture-root", filepath.Join(cliCanonicalTempDirectory(t), "fixture")},
		{"--prepare", "--domain", "cli", "--ordinal", "0", "--fixture-root", filepath.Join(cliCanonicalTempDirectory(t), "fixture")},
		{"--prepare", "--domain", "cli", "--ordinal", "4", "--fixture-root", filepath.Join(cliCanonicalTempDirectory(t), "fixture")},
		{"--prepare", "--domain", "cli", "--ordinal", "01", "--fixture-root", filepath.Join(cliCanonicalTempDirectory(t), "fixture")},
		{"--prepare", "--domain", "cli", "--ordinal", "1", "--fixture-root", "relative"},
		{"--prepare", "--domain", "cli", "--ordinal", "1", "--fixture-root", filepath.Join(cliCanonicalTempDirectory(t), "fixture"), "--evidence-root", filepath.Join(cliCanonicalTempDirectory(t), "unused")},
	} {
		command := exec.Command(cliTestNode, append([]string{driver}, arguments...)...)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err == nil || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "U7_CLI_STUDY_DRIVER:") || !strings.HasSuffix(stderr.String(), "\n") {
			t.Fatalf("invalid invocation result: args=%q err=%v stdout=%q stderr=%q", arguments, err, stdout.Bytes(), stderr.Bytes())
		}
	}

	sideRoot := cliCanonicalTempDirectory(t)
	sentinel := filepath.Join(sideRoot, "sentinel")
	writeCLIFile(t, sentinel, []byte("unchanged"), 0o600)
	fixtureRoot := filepath.Join(cliCanonicalTempDirectory(t), "fixture")
	command := exec.Command(cliTestNode, driver, "--prepare", "--domain", "cli", "--ordinal", "3", "--fixture-root", fixtureRoot)
	command.Env = append(os.Environ(), "COUNTERSHAPE_EVIDENCE_ROOT="+sideRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("valid prepare result: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	entries, err := os.ReadDir(sideRoot)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
		t.Fatalf("unrelated root changed: entries=%v err=%v", entries, err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "unchanged" {
		t.Fatalf("sentinel changed: bytes=%q err=%v", got, err)
	}
}

func assertCLIDriverTrackedAuthority(t *testing.T, root string) {
	t.Helper()
	wantPaths := []string{".gitignore", cliManifestPath}
	for _, role := range CLIRoles() {
		prefix := "authority/cli/" + string(role) + "/"
		wantPaths = append(wantPaths, prefix+"candidate-role.json", prefix+CLIEntrypoint)
		ref := "refs/heads/" + string(role)
		candidateRole := cliGitBytes(t, root, nil, "show", ref+":candidate-role.json")
		trackedRole := cliGitBytes(t, root, nil, "show", "HEAD:"+prefix+"candidate-role.json")
		candidateProgram := cliGitBytes(t, root, nil, "show", ref+":"+CLIEntrypoint)
		trackedProgram := cliGitBytes(t, root, nil, "show", "HEAD:"+prefix+CLIEntrypoint)
		if !bytes.Equal(candidateRole, trackedRole) || !bytes.Equal(candidateProgram, trackedProgram) ||
			!bytes.Equal(candidateProgram, clifixture.Program()) {
			t.Fatalf("candidate %s was not derived from tracked portable bytes", role)
		}
		parents := strings.Fields(string(cliGitBytes(t, root, nil, "rev-list", "--parents", "-n", "1", ref)))
		if len(parents) != 1 {
			t.Fatalf("candidate %s commit is not parentless: %q", role, parents)
		}
		rows := strings.Split(strings.TrimSpace(string(cliGitBytes(t, root, nil, "ls-tree", "-r", ref))), "\n")
		if len(rows) != 2 || !strings.HasPrefix(rows[0], "100644 blob ") || !strings.HasSuffix(rows[0], "\tcandidate-role.json") ||
			!strings.HasPrefix(rows[1], "100755 blob ") || !strings.HasSuffix(rows[1], "\t"+CLIEntrypoint) {
			t.Fatalf("candidate %s tree rows = %v", role, rows)
		}
	}
	sort.Strings(wantPaths)
	gotRows := strings.Split(strings.TrimSpace(string(cliGitBytes(t, root, nil, "ls-tree", "-r", "--name-only", "HEAD"))), "\n")
	sort.Strings(gotRows)
	if fmt.Sprint(gotRows) != fmt.Sprint(wantPaths) {
		t.Fatalf("HEAD tracked roster = %v, want %v", gotRows, wantPaths)
	}
}

func mutateCLICandidateTree(t *testing.T, root string, role CLIRole, programMode string, linkRole, extra bool) {
	t.Helper()
	ref := "refs/heads/" + string(role)
	roleOID := cliGitLine(t, root, nil, "rev-parse", ref+":candidate-role.json")
	programOID := cliGitLine(t, root, nil, "rev-parse", ref+":"+CLIEntrypoint)
	roleMode := "100644"
	if linkRole {
		roleMode = "120000"
	}
	var input bytes.Buffer
	_, _ = fmt.Fprintf(&input, "%s blob %s\tcandidate-role.json%c", roleMode, roleOID, byte(0))
	_, _ = fmt.Fprintf(&input, "%s blob %s\t%s%c", programMode, programOID, CLIEntrypoint, byte(0))
	if extra {
		extraOID := cliGitLine(t, root, []byte("extra"), "hash-object", "-w", "--stdin")
		_, _ = fmt.Fprintf(&input, "100644 blob %s\textra.txt%c", extraOID, byte(0))
	}
	tree := cliGitLine(t, root, input.Bytes(), "mktree", "-z")
	commit := cliGitLine(t, root, nil, "commit-tree", tree, "-m", "Countershape U7 CLI candidate: "+string(role))
	cliGitBytes(t, root, nil, "update-ref", ref, commit)
}

func assertMalformedCLIFixture(t *testing.T, root string) {
	t.Helper()
	_, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
		Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
	})
	if !errors.Is(err, ErrCLIFixtureMalformed) {
		t.Fatalf("malformed fixture error = %v", err)
	}
}

func openCLIFixture(t *testing.T, root string) CLIFixture {
	t.Helper()
	fixture, err := OpenCLIFixture(context.Background(), CLIFixtureConfig{
		Root: root, GitExecutable: cliTestGit, ScratchRoot: cliCanonicalTempDirectory(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("close fixture: %v", err)
		}
	})
	return fixture
}

func prepareCLIFixture(t *testing.T, ordinal int) string {
	t.Helper()
	root := filepath.Join(cliCanonicalTempDirectory(t), "fixture")
	command := exec.Command(cliTestNode, cliDriverPath(t),
		"--prepare", "--domain", "cli", "--ordinal", fmt.Sprint(ordinal), "--fixture-root", root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("prepare fixture: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	return root
}

func cliDriverPath(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../..", "tools/run-u7-cli-study.mjs"))
}

func cliCanonicalTempDirectory(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func cliGitLine(t *testing.T, root string, input []byte, args ...string) string {
	t.Helper()
	line := strings.TrimSuffix(string(cliGitBytes(t, root, input, args...)), "\n")
	if line == "" || strings.ContainsAny(line, "\r\n\x00") {
		t.Fatalf("git %s did not return one line: %q", args[0], line)
	}
	return line
}

func cliGitBytes(t *testing.T, root string, input []byte, args ...string) []byte {
	t.Helper()
	closed := append([]string{"-C", root, "--no-pager", "--no-replace-objects", "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null"}, args...)
	command := exec.Command(cliTestGit, closed...)
	command.Env = []string{
		"HOME=" + root, "TMPDIR=" + root, "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1",
		"GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_AUTHOR_NAME=Countershape U7 Fixture",
		"GIT_AUTHOR_EMAIL=u7-fixture@countershape.invalid", "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z",
		"GIT_COMMITTER_NAME=Countershape U7 Fixture", "GIT_COMMITTER_EMAIL=u7-fixture@countershape.invalid",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z",
	}
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("git %s: err=%v stderr=%q", args[0], err, stderr.Bytes())
	}
	return append([]byte(nil), stdout.Bytes()...)
}

func writeCLIFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
