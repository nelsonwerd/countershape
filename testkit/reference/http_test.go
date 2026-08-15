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

	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

const (
	testGit  = "/usr/bin/git"
	testNode = "/opt/homebrew/bin/node"
)

func TestHTTPDriverCreatesDeterministicCleanParentlessRepository(t *testing.T) {
	first := prepareHTTPFixture(t, 1)
	second := prepareHTTPFixture(t, 2)

	for _, root := range []string{first, second} {
		if status := gitBytes(t, root, "status", "--porcelain=v2", "-z", "--untracked-files=all"); len(status) != 0 {
			t.Fatalf("fixture %s is dirty: %q", root, status)
		}
		if got := strings.TrimSpace(string(gitBytes(t, root, "rev-list", "--count", "HEAD"))); got != "1" {
			t.Fatalf("HEAD commit count = %q, want 1", got)
		}
		head := gitLine(t, root, "rev-parse", "HEAD")
		if got := strings.Fields(string(gitBytes(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))); len(got) != 1 || got[0] != head {
			t.Fatalf("HEAD is not parentless: %q", got)
		}
		assertDriverTrackedAuthority(t, root)
	}

	if firstHead, secondHead := gitLine(t, first, "rev-parse", "HEAD"), gitLine(t, second, "rev-parse", "HEAD"); firstHead != secondHead {
		t.Fatalf("deterministic HEAD drifted: %s != %s", firstHead, secondHead)
	}
	if firstTree, secondTree := gitLine(t, first, "rev-parse", "HEAD^{tree}"), gitLine(t, second, "rev-parse", "HEAD^{tree}"); firstTree != secondTree {
		t.Fatalf("deterministic HEAD tree drifted: %s != %s", firstTree, secondTree)
	}
	for _, role := range HTTPRoles() {
		ref := "refs/heads/" + string(role)
		if left, right := gitLine(t, first, "rev-parse", ref), gitLine(t, second, "rev-parse", ref); left != right {
			t.Fatalf("deterministic %s commit drifted: %s != %s", role, left, right)
		}
		if left, right := gitLine(t, first, "rev-parse", ref+"^{tree}"), gitLine(t, second, "rev-parse", ref+"^{tree}"); left != right {
			t.Fatalf("deterministic %s tree drifted: %s != %s", role, left, right)
		}
	}
}

func TestHTTPFixtureDefensiveAccessAndPinnedRefSurvivesMove(t *testing.T) {
	root := prepareHTTPFixture(t, 1)
	fixture := openHTTPFixture(t, root)
	defer fixture.Close()

	if !fixture.Valid() || fixture.Root() != root || fixture.Entrypoint() != httpfixture.PortableEntrypoint {
		t.Fatalf("opened fixture identity is incomplete: valid=%t root=%q entrypoint=%q", fixture.Valid(), fixture.Root(), fixture.Entrypoint())
	}
	roles := fixture.Roles()
	roles[0] = Alternating
	if got := fixture.Roles()[0]; got != Forbidden {
		t.Fatalf("role roster was not defensive: %q", got)
	}
	refs := fixture.Refs()
	refs[Forbidden] = "refs/heads/alternating"
	ref, err := fixture.Ref(Forbidden)
	if err != nil || ref != "refs/heads/forbidden" {
		t.Fatalf("ref map was not defensive: ref=%q err=%v", ref, err)
	}
	if _, err := fixture.Ref("unknown"); err == nil {
		t.Fatal("unknown role was accepted")
	}
	seed := HTTPSeedJSON()
	seed[0] ^= 0xff
	if bytes.Equal(seed, HTTPSeedJSON()) || HTTPSeedFilename != httpfixture.SeedFilename {
		t.Fatal("seed API returned shared bytes or the wrong filename")
	}

	repository := fixture.Repository()
	pinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	before := pinned.Provenance()
	replacement := gitLine(t, root, "rev-parse", "refs/heads/metadata-disclosure")
	gitBytes(t, root, "update-ref", ref, replacement)
	if after := pinned.Provenance(); after != before || !pinned.Valid() {
		t.Fatalf("pinned candidate changed after display-ref move: before=%+v after=%+v", before, after)
	}
	repinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if repinned.Provenance().CommitOID != replacement || repinned.Provenance().TreeOID == before.TreeOID {
		t.Fatalf("new pin did not observe the moved ref: old=%+v new=%+v", before, repinned.Provenance())
	}
}

func TestHTTPFixtureRefusesMalformedCrossDomainAndSharedRoot(t *testing.T) {
	t.Run("shared root", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		first := openHTTPFixture(t, root)
		defer first.Close()
		_, err := OpenHTTPFixture(context.Background(), HTTPFixtureConfig{
			Root: root, GitExecutable: testGit, ScratchRoot: canonicalTempDirectory(t),
		})
		if !errors.Is(err, ErrHTTPFixtureSharedRoot) {
			t.Fatalf("shared root error = %v", err)
		}
	})

	t.Run("cross domain", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		manifest := filepath.Join(root, filepath.FromSlash(httpManifestPath))
		foreign := strings.Replace(httpManifest, `"domain":"http"`, `"domain":"other"`, 1)
		if err := os.WriteFile(manifest, []byte(foreign), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := OpenHTTPFixture(context.Background(), HTTPFixtureConfig{
			Root: root, GitExecutable: testGit, ScratchRoot: canonicalTempDirectory(t),
		})
		if !errors.Is(err, ErrHTTPFixtureDomain) {
			t.Fatalf("cross-domain error = %v", err)
		}
	})

	t.Run("malformed marker", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(httpManifestPath)), []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		assertMalformedHTTPFixture(t, root)
	})

	t.Run("missing candidate ref", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		gitBytes(t, root, "update-ref", "-d", "refs/heads/alternating")
		assertMalformedHTTPFixture(t, root)
	})

	t.Run("extra ref", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		gitBytes(t, root, "update-ref", "refs/heads/extra", gitLine(t, root, "rev-parse", "HEAD"))
		assertMalformedHTTPFixture(t, root)
	})

	t.Run("parented same-tree candidate", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		ref := "refs/heads/forbidden"
		original := gitLine(t, root, "rev-parse", ref)
		tree := gitLine(t, root, "rev-parse", ref+"^{tree}")
		parented := strings.TrimSpace(string(gitBytes(t, root, "commit-tree", tree, "-p", original, "-m", "parented same tree")))
		gitBytes(t, root, "update-ref", ref, parented)
		assertMalformedHTTPFixture(t, root)
	})

	t.Run("untracked worktree input", func(t *testing.T) {
		root := prepareHTTPFixture(t, 1)
		if err := os.WriteFile(filepath.Join(root, "shared-state.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		assertMalformedHTTPFixture(t, root)
	})

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
			return filepath.Join(root, "authority", "http", string(Forbidden), filepath.FromSlash(HTTPEntrypoint))
		}, mode: 0o700 | os.ModeSticky},
		{name: "special ignored binary", path: func(root string) string { return filepath.Join(root, httpIgnoredBinary) }, mode: 0o700 | os.ModeSetgid, make: true},
	} {
		mutation := mutation
		t.Run(mutation.name, func(t *testing.T) {
			root := prepareHTTPFixture(t, 1)
			path := mutation.path(root)
			if mutation.make {
				if err := os.WriteFile(path, []byte("binary"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			// Darwin silently clears setgid when a private TMPDIR inherits a
			// group the current user does not belong to. Bind these hostile
			// targets to the process group so the requested special bit is
			// actually present before fixture admission inspects it.
			if mutation.mode&os.ModeSetgid != 0 {
				if err := os.Chown(path, -1, os.Getgid()); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(path, mutation.mode); err != nil {
				t.Fatal(err)
			}
			assertMalformedHTTPFixture(t, root)
		})
	}
}

func TestHTTPFixtureConcurrentCopiesCloseAndReopen(t *testing.T) {
	root := prepareHTTPFixture(t, 1)
	fixture := openHTTPFixture(t, root)
	copies := make([]HTTPFixture, 8)
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
				if observed := copy.Entrypoint(); observed != "" && observed != HTTPEntrypoint {
					failures <- fmt.Errorf("copy %d entrypoint = %q", index, observed)
				}
				if roles := copy.Roles(); roles != nil && len(roles) != len(HTTPRoles()) {
					failures <- fmt.Errorf("copy %d roles = %v", index, roles)
				}
				if refs := copy.Refs(); refs != nil && len(refs) != len(HTTPRoles()) {
					failures <- fmt.Errorf("copy %d refs = %v", index, refs)
				}
				ref, err := copy.Ref(Forbidden)
				if err == nil && ref != "refs/heads/forbidden" {
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

	reopened, err := OpenHTTPFixture(context.Background(), HTTPFixtureConfig{
		Root: root, GitExecutable: testGit, ScratchRoot: canonicalTempDirectory(t),
	})
	if err != nil || !reopened.Valid() {
		t.Fatalf("reopen after concurrent close: valid=%t err=%v", reopened.Valid(), err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPDriverRequiresExactPrepareInvocationAndMode(t *testing.T) {
	driver := httpDriverPath(t)
	info, err := os.Lstat(driver)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
		t.Fatalf("driver mode = %v, want regular 0644", info.Mode())
	}

	root := filepath.Join(canonicalTempDirectory(t), "fixture")
	command := exec.Command(testNode, driver, "--prepare", "--domain", "other", "--ordinal", "1", "--fixture-root", root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err == nil || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "U7_HTTP_STUDY_DRIVER:") {
		t.Fatalf("invalid invocation result: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
}

func assertDriverTrackedAuthority(t *testing.T, root string) {
	t.Helper()
	wantPaths := []string{".gitignore", httpManifestPath}
	for _, role := range HTTPRoles() {
		prefix := "authority/http/" + string(role) + "/"
		wantPaths = append(wantPaths, prefix+"candidate-role.json", prefix+HTTPEntrypoint)
		ref := "refs/heads/" + string(role)
		candidateRole := gitBytes(t, root, "show", ref+":candidate-role.json")
		trackedRole := gitBytes(t, root, "show", "HEAD:"+prefix+"candidate-role.json")
		candidateProgram := gitBytes(t, root, "show", ref+":"+HTTPEntrypoint)
		trackedProgram := gitBytes(t, root, "show", "HEAD:"+prefix+HTTPEntrypoint)
		if !bytes.Equal(candidateRole, trackedRole) || !bytes.Equal(candidateProgram, trackedProgram) ||
			!bytes.Equal(candidateProgram, httpfixture.PortableProgram()) {
			t.Fatalf("candidate %s was not derived from tracked portable bytes", role)
		}
		parents := strings.Fields(string(gitBytes(t, root, "rev-list", "--parents", "-n", "1", ref)))
		if len(parents) != 1 {
			t.Fatalf("candidate %s commit is not parentless: %q", role, parents)
		}
	}
	sort.Strings(wantPaths)
	gotRows := strings.Split(strings.TrimSpace(string(gitBytes(t, root, "ls-tree", "-r", "--name-only", "HEAD"))), "\n")
	sort.Strings(gotRows)
	if fmt.Sprint(gotRows) != fmt.Sprint(wantPaths) {
		t.Fatalf("HEAD tracked roster = %v, want %v", gotRows, wantPaths)
	}
}

func assertMalformedHTTPFixture(t *testing.T, root string) {
	t.Helper()
	_, err := OpenHTTPFixture(context.Background(), HTTPFixtureConfig{
		Root: root, GitExecutable: testGit, ScratchRoot: canonicalTempDirectory(t),
	})
	if !errors.Is(err, ErrHTTPFixtureMalformed) {
		t.Fatalf("malformed fixture error = %v", err)
	}
}

func openHTTPFixture(t *testing.T, root string) HTTPFixture {
	t.Helper()
	fixture, err := OpenHTTPFixture(context.Background(), HTTPFixtureConfig{
		Root: root, GitExecutable: testGit, ScratchRoot: canonicalTempDirectory(t),
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

func prepareHTTPFixture(t *testing.T, ordinal int) string {
	t.Helper()
	root := filepath.Join(canonicalTempDirectory(t), "fixture")
	command := exec.Command(testNode, httpDriverPath(t),
		"--prepare", "--domain", "http", "--ordinal", fmt.Sprint(ordinal), "--fixture-root", root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("prepare fixture: err=%v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	return root
}

func httpDriverPath(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../..", "tools/run-u7-http-study.mjs"))
}

func canonicalTempDirectory(t *testing.T) string {
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

func gitLine(t *testing.T, root string, args ...string) string {
	t.Helper()
	line := strings.TrimSuffix(string(gitBytes(t, root, args...)), "\n")
	if line == "" || strings.ContainsAny(line, "\r\n\x00") {
		t.Fatalf("git %s did not return one line: %q", args[0], line)
	}
	return line
}

func gitBytes(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	closed := append([]string{"-C", root, "--no-pager", "--no-replace-objects", "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null"}, args...)
	command := exec.Command(testGit, closed...)
	command.Env = []string{
		"HOME=" + root,
		"TMPDIR=" + root,
		"PATH=/usr/bin:/bin",
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0",
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
