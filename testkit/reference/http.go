// Package reference exposes the closed local fixtures used by the U7
// reference application. Fixture admission is deliberately independent of
// internal/reference so product orchestration can depend on this package
// without creating an authority cycle.
package reference

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

// CandidateRole is the closed HTTP fixture role. It remains an alias so the
// reference study and the earlier HTTP study use exactly one role vocabulary.
type CandidateRole = httpfixture.CandidateRole

const (
	Forbidden          = httpfixture.Forbidden
	ConcealNotFound    = httpfixture.ConcealNotFound
	MetadataDisclosure = httpfixture.MetadataDisclosure
	Alternating        = httpfixture.Alternating

	HTTPEntrypoint   = httpfixture.PortableEntrypoint
	HTTPSeedFilename = httpfixture.SeedFilename
)

const (
	httpFixtureSchema  = "countershape/u7-http-fixture/v1"
	httpFixtureDomain  = "http"
	httpFixtureMainRef = "refs/heads/main"
	httpManifestPath   = "authority/http/manifest.json"
	httpIgnoredBinary  = "countershape"
	httpGitignore      = "/countershape\n"
	httpManifest       = `{"schema_version":"countershape/u7-http-fixture/v1","domain":"http","entrypoint":"fixture/server_child_bind.mjs","roles":["forbidden","conceal-not-found","metadata-disclosure","alternating"]}`
)

var (
	// ErrHTTPFixtureMalformed identifies a fixture whose filesystem, Git refs,
	// or candidate bytes do not match the closed U7 HTTP fixture profile.
	ErrHTTPFixtureMalformed = errors.New("U7 HTTP fixture is malformed")
	// ErrHTTPFixtureDomain identifies a well-formed fixture marker belonging to
	// a different study domain.
	ErrHTTPFixtureDomain = errors.New("U7 fixture belongs to another domain")
	// ErrHTTPFixtureSharedRoot prevents two live studies from sharing one
	// mutable fixture root inside a process.
	ErrHTTPFixtureSharedRoot = errors.New("U7 HTTP fixture root is already open")
)

var httpRootRegistry = struct {
	sync.Mutex
	open map[string]struct{}
}{open: make(map[string]struct{})}

// HTTPFixtureConfig names every ambient capability needed to open a driver
// fixture. Callers must pass the admitted absolute Git executable and a
// private scratch directory; neither is discovered from PATH or HOME.
type HTTPFixtureConfig struct {
	Root          string
	GitExecutable string
	ScratchRoot   string
}

type httpFixtureState struct {
	mu         sync.Mutex
	root       string
	repository gitobj.Repository
	refs       map[CandidateRole]string
	closed     bool
}

// HTTPFixture is a live, process-local admission of one driver-created Git
// worktree. Copies share close state and repository authority.
type HTTPFixture struct{ state *httpFixtureState }

type httpManifestEnvelope struct {
	SchemaVersion string          `json:"schema_version"`
	Domain        string          `json:"domain"`
	Entrypoint    string          `json:"entrypoint"`
	Roles         []CandidateRole `json:"roles"`
}

type fixtureFile struct {
	mode    string
	content []byte
}

// HTTPRoles returns a defensive copy of the closed display roster.
func HTTPRoles() []CandidateRole {
	return append([]CandidateRole(nil), httpfixture.Roles()...)
}

// HTTPSeedJSON returns a defensive copy of the exact reference stimulus seed.
func HTTPSeedJSON() []byte {
	return httpfixture.SeedJSON()
}

// OpenHTTPFixture admits and verifies one fixture prepared by
// tools/run-u7-http-study.mjs. It checks the exact worktree bytes, the tracked
// HEAD authority copy, and each candidate ref through the hardened Git object
// reader before returning the repository capability.
func OpenHTTPFixture(ctx context.Context, config HTTPFixtureConfig) (HTTPFixture, error) {
	root, err := canonicalPrivateDirectory(config.Root)
	if err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: root: %v", ErrHTTPFixtureMalformed, err)
	}
	if err := reserveHTTPRoot(root); err != nil {
		return HTTPFixture{}, err
	}
	reserved := true
	defer func() {
		if reserved {
			releaseHTTPRoot(root)
		}
	}()

	if err := validateHTTPWorktree(root); err != nil {
		return HTTPFixture{}, err
	}
	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: config.GitExecutable,
		Repository:    root,
		ScratchRoot:   config.ScratchRoot,
	})
	if err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: open repository: %v", ErrHTTPFixtureMalformed, err)
	}
	keepRepository := false
	defer func() {
		if !keepRepository {
			_ = repository.Close()
		}
	}()
	if repository.ObjectFormat() != gitobj.ObjectSHA1 {
		return HTTPFixture{}, fmt.Errorf("%w: object format %q", ErrHTTPFixtureMalformed, repository.ObjectFormat())
	}

	policy, err := gitobj.NewPolicy(32, 8<<20, 2<<20)
	if err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: inspection policy: %v", ErrHTTPFixtureMalformed, err)
	}
	head, err := repository.Pin(ctx, httpFixtureMainRef)
	if err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: HEAD: %v", ErrHTTPFixtureMalformed, err)
	}
	inspectedHead, err := gitobj.Inspect(ctx, head, policy, config.ScratchRoot)
	if err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: inspect HEAD: %v", ErrHTTPFixtureMalformed, err)
	}
	if head.Provenance().DisplayRef != httpFixtureMainRef {
		return HTTPFixture{}, fmt.Errorf("%w: HEAD provenance", ErrHTTPFixtureMalformed)
	}
	if err := validateInspectedFiles(inspectedHead.Entries(), expectedHTTPHeadFiles()); err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: HEAD: %v", ErrHTTPFixtureMalformed, err)
	}

	refs := make(map[CandidateRole]string, len(HTTPRoles()))
	commitAuthority := map[string]httpCommitAuthority{
		httpFixtureMainRef: {
			oid: head.Provenance().CommitOID, tree: head.Provenance().TreeOID,
			message: "Countershape U7 HTTP fixture authority",
		},
	}
	seenCommits := map[string]struct{}{head.Provenance().CommitOID: {}}
	seenTrees := map[string]struct{}{head.Provenance().TreeOID: {}}
	for _, role := range HTTPRoles() {
		ref := "refs/heads/" + string(role)
		pinned, pinErr := repository.Pin(ctx, ref)
		if pinErr != nil {
			return HTTPFixture{}, fmt.Errorf("%w: %s: %v", ErrHTTPFixtureMalformed, ref, pinErr)
		}
		provenance := pinned.Provenance()
		if provenance.DisplayRef != ref {
			return HTTPFixture{}, fmt.Errorf("%w: %s provenance", ErrHTTPFixtureMalformed, ref)
		}
		if _, duplicate := seenCommits[provenance.CommitOID]; duplicate {
			return HTTPFixture{}, fmt.Errorf("%w: duplicate commit for %s", ErrHTTPFixtureMalformed, role)
		}
		if _, duplicate := seenTrees[provenance.TreeOID]; duplicate {
			return HTTPFixture{}, fmt.Errorf("%w: duplicate tree for %s", ErrHTTPFixtureMalformed, role)
		}
		seenCommits[provenance.CommitOID] = struct{}{}
		seenTrees[provenance.TreeOID] = struct{}{}
		inspected, inspectErr := gitobj.Inspect(ctx, pinned, policy, config.ScratchRoot)
		if inspectErr != nil {
			return HTTPFixture{}, fmt.Errorf("%w: inspect %s: %v", ErrHTTPFixtureMalformed, ref, inspectErr)
		}
		expected, expectedErr := expectedHTTPCandidateFiles(role)
		if expectedErr != nil {
			return HTTPFixture{}, fmt.Errorf("%w: expected %s: %v", ErrHTTPFixtureMalformed, role, expectedErr)
		}
		if inspectErr := validateInspectedFiles(inspected.Entries(), expected); inspectErr != nil {
			return HTTPFixture{}, fmt.Errorf("%w: %s: %v", ErrHTTPFixtureMalformed, ref, inspectErr)
		}
		refs[role] = ref
		commitAuthority[ref] = httpCommitAuthority{
			oid: provenance.CommitOID, tree: provenance.TreeOID,
			message: "Countershape U7 HTTP candidate: " + string(role),
		}
	}
	if err := validateHTTPGitAuthority(root, commitAuthority); err != nil {
		return HTTPFixture{}, fmt.Errorf("%w: Git authority: %v", ErrHTTPFixtureMalformed, err)
	}

	state := &httpFixtureState{root: root, repository: repository, refs: refs}
	keepRepository = true
	reserved = false
	return HTTPFixture{state: state}, nil
}

// Valid reports whether the fixture still owns its admitted repository/root
// pair in this process.
func (f HTTPFixture) Valid() bool {
	if f.state == nil {
		return false
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	return !f.state.closed && f.state.repository.Valid()
}

// Root returns the canonical private worktree root.
func (f HTTPFixture) Root() string {
	if f.state == nil {
		return ""
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed {
		return ""
	}
	return f.state.root
}

// Entrypoint returns the candidate-root-relative portable Node entrypoint.
func (f HTTPFixture) Entrypoint() string {
	if !f.Valid() {
		return ""
	}
	return HTTPEntrypoint
}

// Roles returns a defensive copy of this fixture's closed role roster.
func (f HTTPFixture) Roles() []CandidateRole {
	if !f.Valid() {
		return nil
	}
	return HTTPRoles()
}

// Ref returns the one fully qualified display ref assigned to role.
func (f HTTPFixture) Ref(role CandidateRole) (string, error) {
	if f.state == nil {
		return "", fmt.Errorf("%w: fixture is closed", ErrHTTPFixtureMalformed)
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !f.state.repository.Valid() {
		return "", fmt.Errorf("%w: fixture is closed", ErrHTTPFixtureMalformed)
	}
	ref, ok := f.state.refs[role]
	if !ok {
		return "", fmt.Errorf("%w: unsupported role %q", ErrHTTPFixtureMalformed, role)
	}
	return ref, nil
}

// Refs returns a defensive copy of the role-to-display-ref map.
func (f HTTPFixture) Refs() map[CandidateRole]string {
	if f.state == nil {
		return nil
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !f.state.repository.Valid() {
		return nil
	}
	result := make(map[CandidateRole]string, len(f.state.refs))
	for role, ref := range f.state.refs {
		result[role] = ref
	}
	return result
}

// Repository returns the admitted opaque Git repository capability. The
// returned value shares lifetime with the fixture and becomes invalid at Close.
func (f HTTPFixture) Repository() gitobj.Repository {
	if f.state == nil {
		return gitobj.Repository{}
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed {
		return gitobj.Repository{}
	}
	return f.state.repository
}

// Close releases the Git capability and the process-local shared-root guard.
// It is idempotent across all copies of the fixture value.
func (f HTTPFixture) Close() error {
	if f.state == nil {
		return nil
	}
	f.state.mu.Lock()
	if f.state.closed {
		f.state.mu.Unlock()
		return nil
	}
	f.state.closed = true
	repository := f.state.repository
	root := f.state.root
	f.state.mu.Unlock()
	err := repository.Close()
	releaseHTTPRoot(root)
	return err
}

func reserveHTTPRoot(root string) error {
	httpRootRegistry.Lock()
	defer httpRootRegistry.Unlock()
	if _, exists := httpRootRegistry.open[root]; exists {
		return fmt.Errorf("%w: %s", ErrHTTPFixtureSharedRoot, root)
	}
	httpRootRegistry.open[root] = struct{}{}
	return nil
}

func releaseHTTPRoot(root string) {
	httpRootRegistry.Lock()
	delete(httpRootRegistry.open, root)
	httpRootRegistry.Unlock()
}

func canonicalPrivateDirectory(root string) (string, error) {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", errors.New("path is not clean and absolute")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return "", errors.New("path is not canonical and symlink-free")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 || hasSpecialMode(info.Mode()) {
		return "", errors.New("directory is not a private real directory")
	}
	return root, nil
}

func validateHTTPWorktree(root string) error {
	manifestBytes, err := readRegularFile(filepath.Join(root, filepath.FromSlash(httpManifestPath)), 16<<10)
	if err != nil {
		return fmt.Errorf("%w: manifest: %v", ErrHTTPFixtureMalformed, err)
	}
	var manifest httpManifestEnvelope
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("%w: manifest JSON", ErrHTTPFixtureMalformed)
	}
	if manifest.Domain != httpFixtureDomain {
		return fmt.Errorf("%w: %q", ErrHTTPFixtureDomain, manifest.Domain)
	}
	if !bytes.Equal(manifestBytes, []byte(httpManifest)) || manifest.SchemaVersion != httpFixtureSchema ||
		manifest.Entrypoint != HTTPEntrypoint || !equalRoles(manifest.Roles, HTTPRoles()) {
		return fmt.Errorf("%w: manifest bytes", ErrHTTPFixtureMalformed)
	}

	expected := expectedHTTPHeadFiles()
	seen := make(map[string]struct{}, len(expected))
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if relative == "." {
			return nil
		}
		relative = filepath.ToSlash(relative)
		if relative == ".git" {
			info, infoErr := entry.Info()
			if infoErr != nil || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 ||
				info.Mode().Perm() != 0o700 || hasSpecialMode(info.Mode()) {
				return errors.New(".git is not a real directory")
			}
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link %s", relative)
		}
		if entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr != nil || info.Mode().Perm() != 0o700 || hasSpecialMode(info.Mode()) {
				return fmt.Errorf("non-private directory %s", relative)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular entry %s", relative)
		}
		if relative == httpIgnoredBinary {
			info, infoErr := entry.Info()
			if infoErr != nil || info.Mode().Perm()&0o111 == 0 || hasSpecialMode(info.Mode()) {
				return errors.New("ignored reference binary is not executable")
			}
			return nil
		}
		want, ok := expected[relative]
		if !ok {
			return fmt.Errorf("unexpected worktree file %s", relative)
		}
		actual, readErr := readRegularFile(path, int64(len(want.content)))
		if readErr != nil || !bytes.Equal(actual, want.content) {
			return fmt.Errorf("worktree bytes differ for %s", relative)
		}
		info, infoErr := entry.Info()
		wantPerm := os.FileMode(0o600)
		if want.mode == "100755" {
			wantPerm = 0o700
		}
		if infoErr != nil || info.Mode().Perm() != wantPerm || hasSpecialMode(info.Mode()) {
			return fmt.Errorf("worktree mode differs for %s", relative)
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHTTPFixtureMalformed, err)
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("%w: worktree file roster %d/%d", ErrHTTPFixtureMalformed, len(seen), len(expected))
	}
	return nil
}

func readRegularFile(path string, maximum int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || hasSpecialMode(before.Mode()) ||
		before.Size() < 0 || before.Size() > maximum {
		return nil, errors.New("file is unavailable, nonregular, or oversized")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytesRead, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(bytesRead)) != before.Size() {
		return nil, errors.New("file read changed or exceeded its bound")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() || before.Mode() != after.Mode() {
		return nil, errors.New("file changed while being read")
	}
	return bytesRead, nil
}

func expectedHTTPHeadFiles() map[string]fixtureFile {
	result := map[string]fixtureFile{
		".gitignore":     {mode: "100644", content: []byte(httpGitignore)},
		httpManifestPath: {mode: "100644", content: []byte(httpManifest)},
	}
	for _, role := range HTTPRoles() {
		files, err := expectedHTTPCandidateFiles(role)
		if err != nil {
			panic(err)
		}
		for path, file := range files {
			result["authority/http/"+string(role)+"/"+path] = fixtureFile{
				mode: file.mode, content: append([]byte(nil), file.content...),
			}
		}
	}
	return result
}

func expectedHTTPCandidateFiles(role CandidateRole) (map[string]fixtureFile, error) {
	files, err := httpfixture.PortableCandidateFiles(role)
	if err != nil {
		return nil, err
	}
	result := make(map[string]fixtureFile, len(files))
	for _, file := range files {
		if _, duplicate := result[file.Path]; duplicate {
			return nil, fmt.Errorf("duplicate fixture path %q", file.Path)
		}
		result[file.Path] = fixtureFile{mode: file.Mode, content: append([]byte(nil), file.Content...)}
	}
	return result, nil
}

func validateInspectedFiles(entries []gitobj.ManifestEntry, expected map[string]fixtureFile) error {
	paths := make([]string, 0, len(expected))
	for path := range expected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(entries) != len(paths) {
		return fmt.Errorf("entry count %d != %d", len(entries), len(paths))
	}
	for index, path := range paths {
		want := expected[path]
		entry := entries[index]
		if entry.Path != path || entry.Mode != want.mode || entry.Bytes != int64(len(want.content)) || entry.GitOID != gitBlobOID(want.content) {
			return fmt.Errorf("entry %d differs for %s", index, path)
		}
	}
	return nil
}

func gitBlobOID(content []byte) string {
	hash := sha1.New() // Git SHA-1 object identity; portable authority remains in internal/gitobj.
	_, _ = fmt.Fprintf(hash, "blob %d%c", len(content), byte(0))
	_, _ = hash.Write(content)
	return hex.EncodeToString(hash.Sum(nil))
}

type httpCommitAuthority struct {
	oid     string
	tree    string
	message string
}

func validateHTTPGitAuthority(root string, commits map[string]httpCommitAuthority) error {
	if len(commits) != len(HTTPRoles())+1 {
		return errors.New("commit authority roster is incomplete")
	}
	head, err := readRegularFile(filepath.Join(root, ".git", "HEAD"), 256)
	if err != nil || !bytes.Equal(head, []byte("ref: "+httpFixtureMainRef+"\n")) {
		return errors.New("HEAD is not the exact main symbolic ref")
	}
	if _, err := os.Lstat(filepath.Join(root, ".git", "packed-refs")); err == nil || !os.IsNotExist(err) {
		return errors.New("packed refs are not admitted")
	}
	expectedRefs := make(map[string]string, len(commits))
	for ref, authority := range commits {
		if !validSHA1OID(authority.oid) || !validSHA1OID(authority.tree) || authority.message == "" {
			return errors.New("commit authority is malformed")
		}
		relative := filepath.ToSlash(strings.TrimPrefix(ref, "refs/"))
		expectedRefs[relative] = authority.oid
		commit, objectErr := readLooseGitObject(root, authority.oid, "commit", 16<<10)
		if objectErr != nil || !bytes.Equal(commit, exactHTTPCommit(authority.tree, authority.message)) {
			return fmt.Errorf("commit %s is not the exact parentless authority", ref)
		}
	}
	seen := make(map[string]struct{}, len(expectedRefs))
	refsRoot := filepath.Join(root, ".git", "refs")
	err = filepath.WalkDir(refsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == refsRoot {
			return nil
		}
		relative, relErr := filepath.Rel(refsRoot, path)
		if relErr != nil {
			return relErr
		}
		relative = filepath.ToSlash(relative)
		info, infoErr := entry.Info()
		if infoErr != nil || entry.Type()&os.ModeSymlink != 0 || hasSpecialMode(info.Mode()) {
			return fmt.Errorf("ref entry %s is not exact", relative)
		}
		if entry.IsDir() {
			if info.Mode().Perm() != 0o700 {
				return fmt.Errorf("ref directory %s is not private", relative)
			}
			return nil
		}
		if !entry.Type().IsRegular() || info.Mode().Perm() != 0o600 {
			return fmt.Errorf("ref %s is not a private regular file", relative)
		}
		want, present := expectedRefs[relative]
		if !present {
			return fmt.Errorf("unexpected ref %s", relative)
		}
		exact, readErr := readRegularFile(path, 128)
		if readErr != nil || !bytes.Equal(exact, []byte(want+"\n")) {
			return fmt.Errorf("ref %s differs", relative)
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil || len(seen) != len(expectedRefs) {
		return fmt.Errorf("ref roster %d/%d: %v", len(seen), len(expectedRefs), err)
	}
	return nil
}

func readLooseGitObject(root, oid, kind string, maximum int64) ([]byte, error) {
	if !validSHA1OID(oid) || kind == "" || maximum < 1 {
		return nil, errors.New("loose object request is malformed")
	}
	compressed, err := readRegularFile(filepath.Join(root, ".git", "objects", oid[:2], oid[2:]), maximum)
	if err != nil {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, errors.New("loose object compression is malformed")
	}
	decompressed, readErr := io.ReadAll(io.LimitReader(reader, maximum+128))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || int64(len(decompressed)) > maximum+127 {
		return nil, errors.New("loose object exceeds its bound")
	}
	headerEnd := bytes.IndexByte(decompressed, 0)
	if headerEnd < 1 {
		return nil, errors.New("loose object header is malformed")
	}
	payload := decompressed[headerEnd+1:]
	wantHeader := fmt.Sprintf("%s %d", kind, len(payload))
	if string(decompressed[:headerEnd]) != wantHeader {
		return nil, errors.New("loose object header differs")
	}
	digest := sha1.Sum(decompressed) // Git SHA-1 object identity, verified over header plus payload.
	if hex.EncodeToString(digest[:]) != oid {
		return nil, errors.New("loose object digest differs")
	}
	return append([]byte(nil), payload...), nil
}

func exactHTTPCommit(tree, message string) []byte {
	return []byte(fmt.Sprintf(
		"tree %s\nauthor Countershape U7 Fixture <u7-fixture@countershape.invalid> 946684800 +0000\n"+
			"committer Countershape U7 Fixture <u7-fixture@countershape.invalid> 946684800 +0000\n\n%s\n",
		tree, message,
	))
}

func validSHA1OID(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func hasSpecialMode(mode os.FileMode) bool {
	return mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0
}

func executableMode(mode os.FileMode) bool {
	return mode.Perm()&0o111 != 0
}

func equalRoles(left, right []CandidateRole) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
