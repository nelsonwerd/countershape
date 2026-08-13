// Package reference exposes the closed local fixtures used by the U7
// reference application. Fixture admission is deliberately independent of
// internal/reference so product orchestration can depend on this package
// without creating an authority cycle.
package reference

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
)

// CLIRole is the closed precedence-fixture role. It remains an alias so the
// reference study and the sealed substrate fixture use one role vocabulary.
type CLIRole = clifixture.CandidateRole

const (
	ConfigFirst      = clifixture.ConfigFirst
	EnvironmentFirst = clifixture.EnvironmentFirst
	ArgvFirst        = clifixture.ArgvFirst

	CLIEntrypoint = clifixture.Entrypoint
)

const (
	cliFixtureSchema   = "countershape/u7-cli-fixture/v1"
	cliFixtureDomain   = "cli"
	cliFixtureMainRef  = "refs/heads/main"
	cliManifestPath    = "authority/cli/manifest.json"
	cliIgnoredBinary   = "countershape"
	cliGitignore       = "/countershape\n"
	cliManifest        = `{"schema_version":"countershape/u7-cli-fixture/v1","domain":"cli","entrypoint":"fixture.mjs","roles":["config-first","env-first","argv-first"]}`
	cliGitConfig       = "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n\tlogallrefupdates = true\n"
	maxCLIRepositories = 8
)

var (
	// ErrCLIFixtureMalformed identifies a fixture whose filesystem, Git refs,
	// commits, trees, or candidate bytes differ from the closed U7 profile.
	ErrCLIFixtureMalformed = errors.New("U7 CLI fixture is malformed")
	// ErrCLIFixtureDomain identifies a well-formed fixture marker belonging to
	// a different study domain.
	ErrCLIFixtureDomain = errors.New("U7 fixture belongs to another domain")
	// ErrCLIFixtureSharedRoot prevents two live studies from sharing one
	// mutable fixture root inside a process.
	ErrCLIFixtureSharedRoot = errors.New("U7 CLI fixture root is already open")
)

var cliRootRegistry = struct {
	sync.Mutex
	open map[string]struct{}
}{open: make(map[string]struct{})}

// CLIFixtureConfig names every ambient capability needed to open a prepared
// fixture. Callers must pass the admitted absolute Git executable and a
// private scratch directory; neither is discovered from PATH or HOME.
type CLIFixtureConfig struct {
	Root          string
	GitExecutable string
	ScratchRoot   string
	// RepositoryCount admits a bounded roster of independent opaque Git
	// capabilities over the same exact fixture. Zero preserves the historical
	// single-capability behavior.
	RepositoryCount int
}

type cliFixtureState struct {
	mu           sync.Mutex
	closeOnce    sync.Once
	closeErr     error
	root         string
	repositories []gitobj.Repository
	refs         map[CLIRole]string
	closed       bool
}

// CLIFixture is a live, process-local admission of one driver-created Git
// worktree. Copies share close state and repository authority.
type CLIFixture struct{ state *cliFixtureState }

type cliManifestEnvelope struct {
	SchemaVersion string    `json:"schema_version"`
	Domain        string    `json:"domain"`
	Entrypoint    string    `json:"entrypoint"`
	Roles         []CLIRole `json:"roles"`
}

// CLIRoles returns a defensive copy of the closed display roster.
func CLIRoles() []CLIRole {
	return append([]CLIRole(nil), clifixture.Roles()...)
}

// OpenCLIFixture admits and verifies one fixture prepared by
// tools/run-u7-cli-study.mjs. It checks the exact private worktree, main
// authority tree, candidate refs, parentless commit bytes, and repository
// configuration before returning the opaque Git capability.
func OpenCLIFixture(ctx context.Context, config CLIFixtureConfig) (CLIFixture, error) {
	repositoryCount := config.RepositoryCount
	if repositoryCount == 0 {
		repositoryCount = 1
	}
	if repositoryCount < 1 || repositoryCount > maxCLIRepositories {
		return CLIFixture{}, fmt.Errorf("%w: repository count", ErrCLIFixtureMalformed)
	}
	root, err := canonicalPrivateDirectory(config.Root)
	if err != nil {
		return CLIFixture{}, fmt.Errorf("%w: root: %v", ErrCLIFixtureMalformed, err)
	}
	if err := reserveCLIRoot(root); err != nil {
		return CLIFixture{}, err
	}
	reserved := true
	defer func() {
		if reserved {
			releaseCLIRoot(root)
		}
	}()

	if err := validateCLIWorktree(root); err != nil {
		return CLIFixture{}, err
	}
	if err := validateCLIRepositoryConfig(root); err != nil {
		return CLIFixture{}, fmt.Errorf("%w: Git config: %v", ErrCLIFixtureMalformed, err)
	}
	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: config.GitExecutable,
		Repository:    root,
		ScratchRoot:   config.ScratchRoot,
	})
	if err != nil {
		return CLIFixture{}, fmt.Errorf("%w: open repository: %v", ErrCLIFixtureMalformed, err)
	}
	repositories := []gitobj.Repository{repository}
	keepRepositories := false
	defer func() {
		if !keepRepositories {
			for _, opened := range repositories {
				_ = opened.Close()
			}
		}
	}()
	if repository.ObjectFormat() != gitobj.ObjectSHA1 {
		return CLIFixture{}, fmt.Errorf("%w: object format %q", ErrCLIFixtureMalformed, repository.ObjectFormat())
	}

	policy, err := gitobj.NewPolicy(32, 8<<20, 2<<20)
	if err != nil {
		return CLIFixture{}, fmt.Errorf("%w: inspection policy: %v", ErrCLIFixtureMalformed, err)
	}
	head, err := repository.Pin(ctx, cliFixtureMainRef)
	if err != nil {
		return CLIFixture{}, fmt.Errorf("%w: HEAD: %v", ErrCLIFixtureMalformed, err)
	}
	inspectedHead, err := gitobj.Inspect(ctx, head, policy, config.ScratchRoot)
	if err != nil {
		return CLIFixture{}, fmt.Errorf("%w: inspect HEAD: %v", ErrCLIFixtureMalformed, err)
	}
	if head.Provenance().DisplayRef != cliFixtureMainRef {
		return CLIFixture{}, fmt.Errorf("%w: HEAD provenance", ErrCLIFixtureMalformed)
	}
	expectedHead := expectedCLIHeadFiles()
	if err := validateInspectedFiles(inspectedHead.Entries(), expectedHead); err != nil {
		return CLIFixture{}, fmt.Errorf("%w: HEAD: %v", ErrCLIFixtureMalformed, err)
	}
	wantHeadTree, err := exactGitTreeOID(expectedHead)
	if err != nil || head.Provenance().TreeOID != wantHeadTree {
		return CLIFixture{}, fmt.Errorf("%w: HEAD tree", ErrCLIFixtureMalformed)
	}

	refs := make(map[CLIRole]string, len(CLIRoles()))
	commitAuthority := map[string]cliCommitAuthority{
		cliFixtureMainRef: {
			oid: head.Provenance().CommitOID, tree: head.Provenance().TreeOID,
			message: "Countershape U7 CLI fixture authority",
		},
	}
	seenCommits := map[string]struct{}{head.Provenance().CommitOID: {}}
	seenTrees := map[string]struct{}{head.Provenance().TreeOID: {}}
	for _, role := range CLIRoles() {
		ref := "refs/heads/" + string(role)
		pinned, pinErr := repository.Pin(ctx, ref)
		if pinErr != nil {
			return CLIFixture{}, fmt.Errorf("%w: %s: %v", ErrCLIFixtureMalformed, ref, pinErr)
		}
		provenance := pinned.Provenance()
		if provenance.DisplayRef != ref {
			return CLIFixture{}, fmt.Errorf("%w: %s provenance", ErrCLIFixtureMalformed, ref)
		}
		if _, duplicate := seenCommits[provenance.CommitOID]; duplicate {
			return CLIFixture{}, fmt.Errorf("%w: duplicate commit for %s", ErrCLIFixtureMalformed, role)
		}
		if _, duplicate := seenTrees[provenance.TreeOID]; duplicate {
			return CLIFixture{}, fmt.Errorf("%w: duplicate tree for %s", ErrCLIFixtureMalformed, role)
		}
		seenCommits[provenance.CommitOID] = struct{}{}
		seenTrees[provenance.TreeOID] = struct{}{}
		inspected, inspectErr := gitobj.Inspect(ctx, pinned, policy, config.ScratchRoot)
		if inspectErr != nil {
			return CLIFixture{}, fmt.Errorf("%w: inspect %s: %v", ErrCLIFixtureMalformed, ref, inspectErr)
		}
		expected, expectedErr := expectedCLICandidateFiles(role)
		if expectedErr != nil {
			return CLIFixture{}, fmt.Errorf("%w: expected %s: %v", ErrCLIFixtureMalformed, role, expectedErr)
		}
		if inspectErr := validateInspectedFiles(inspected.Entries(), expected); inspectErr != nil {
			return CLIFixture{}, fmt.Errorf("%w: %s: %v", ErrCLIFixtureMalformed, ref, inspectErr)
		}
		wantTree, treeErr := exactGitTreeOID(expected)
		if treeErr != nil || provenance.TreeOID != wantTree {
			return CLIFixture{}, fmt.Errorf("%w: %s tree", ErrCLIFixtureMalformed, ref)
		}
		refs[role] = ref
		commitAuthority[ref] = cliCommitAuthority{
			oid: provenance.CommitOID, tree: provenance.TreeOID,
			message: "Countershape U7 CLI candidate: " + string(role),
		}
	}
	if err := validateCLIGitAuthority(root, commitAuthority); err != nil {
		return CLIFixture{}, fmt.Errorf("%w: Git authority: %v", ErrCLIFixtureMalformed, err)
	}
	for index := 1; index < repositoryCount; index++ {
		additional, openErr := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
			GitExecutable: config.GitExecutable,
			Repository:    root,
			ScratchRoot:   config.ScratchRoot,
		})
		if openErr != nil {
			return CLIFixture{}, fmt.Errorf("%w: open repository %d: %v", ErrCLIFixtureMalformed, index+1, openErr)
		}
		repositories = append(repositories, additional)
		if additional.ObjectFormat() != repository.ObjectFormat() ||
			additional.Fingerprint() != repository.Fingerprint() {
			return CLIFixture{}, fmt.Errorf("%w: repository roster authority", ErrCLIFixtureMalformed)
		}
	}

	state := &cliFixtureState{root: root, repositories: repositories, refs: refs}
	keepRepositories = true
	reserved = false
	return CLIFixture{state: state}, nil
}

// Valid reports whether the fixture still owns its admitted repository/root
// pair in this process.
func (f CLIFixture) Valid() bool {
	if f.state == nil {
		return false
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	return !f.state.closed && validCLIRepositoryRoster(f.state.repositories)
}

// Root returns the canonical private worktree root.
func (f CLIFixture) Root() string {
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
func (f CLIFixture) Entrypoint() string {
	if !f.Valid() {
		return ""
	}
	return CLIEntrypoint
}

// Roles returns a defensive copy of this fixture's closed role roster.
func (f CLIFixture) Roles() []CLIRole {
	if !f.Valid() {
		return nil
	}
	return CLIRoles()
}

// Ref returns the one fully qualified display ref assigned to role.
func (f CLIFixture) Ref(role CLIRole) (string, error) {
	if f.state == nil {
		return "", fmt.Errorf("%w: fixture is closed", ErrCLIFixtureMalformed)
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !validCLIRepositoryRoster(f.state.repositories) {
		return "", fmt.Errorf("%w: fixture is closed", ErrCLIFixtureMalformed)
	}
	ref, ok := f.state.refs[role]
	if !ok {
		return "", fmt.Errorf("%w: unsupported role %q", ErrCLIFixtureMalformed, role)
	}
	return ref, nil
}

// Refs returns a defensive copy of the role-to-display-ref map.
func (f CLIFixture) Refs() map[CLIRole]string {
	if f.state == nil {
		return nil
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !validCLIRepositoryRoster(f.state.repositories) {
		return nil
	}
	result := make(map[CLIRole]string, len(f.state.refs))
	for role, ref := range f.state.refs {
		result[role] = ref
	}
	return result
}

// Repository returns the admitted opaque Git repository capability. The
// returned value shares lifetime with the fixture and becomes invalid at Close.
func (f CLIFixture) Repository() gitobj.Repository {
	if f.state == nil {
		return gitobj.Repository{}
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !validCLIRepositoryRoster(f.state.repositories) {
		return gitobj.Repository{}
	}
	return f.state.repositories[0]
}

// Repositories returns a defensive copy of the admitted independent Git
// capability roster. Every capability shares the fixture lifetime but has an
// independent repository lock and private Git environment.
func (f CLIFixture) Repositories() []gitobj.Repository {
	if f.state == nil {
		return nil
	}
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	if f.state.closed || !validCLIRepositoryRoster(f.state.repositories) {
		return nil
	}
	return append([]gitobj.Repository(nil), f.state.repositories...)
}

// Close releases every admitted Git capability and then the process-local
// shared-root guard. All copies wait on the same revocation cell.
// It is idempotent across all copies of the fixture value.
func (f CLIFixture) Close() error {
	if f.state == nil {
		return nil
	}
	f.state.closeOnce.Do(func() {
		f.state.mu.Lock()
		f.state.closed = true
		repositories := append([]gitobj.Repository(nil), f.state.repositories...)
		root := f.state.root
		f.state.mu.Unlock()
		var closeErr error
		for _, repository := range repositories {
			closeErr = errors.Join(closeErr, repository.Close())
		}
		releaseCLIRoot(root)
		f.state.mu.Lock()
		f.state.closeErr = closeErr
		f.state.mu.Unlock()
	})
	f.state.mu.Lock()
	defer f.state.mu.Unlock()
	return f.state.closeErr
}

func validCLIRepositoryRoster(repositories []gitobj.Repository) bool {
	if len(repositories) < 1 || len(repositories) > maxCLIRepositories {
		return false
	}
	fingerprint := repositories[0].Fingerprint()
	format := repositories[0].ObjectFormat()
	if !repositories[0].Valid() || !fingerprint.Valid() || format == "" {
		return false
	}
	seen := map[gitobj.Repository]struct{}{repositories[0]: {}}
	for _, repository := range repositories[1:] {
		if !repository.Valid() || repository.Fingerprint() != fingerprint || repository.ObjectFormat() != format {
			return false
		}
		if _, duplicate := seen[repository]; duplicate {
			return false
		}
		seen[repository] = struct{}{}
	}
	return true
}

func reserveCLIRoot(root string) error {
	cliRootRegistry.Lock()
	defer cliRootRegistry.Unlock()
	if _, exists := cliRootRegistry.open[root]; exists {
		return fmt.Errorf("%w: %s", ErrCLIFixtureSharedRoot, root)
	}
	cliRootRegistry.open[root] = struct{}{}
	return nil
}

func releaseCLIRoot(root string) {
	cliRootRegistry.Lock()
	delete(cliRootRegistry.open, root)
	cliRootRegistry.Unlock()
}

func validateCLIWorktree(root string) error {
	manifestBytes, err := readRegularFile(filepath.Join(root, filepath.FromSlash(cliManifestPath)), 16<<10)
	if err != nil {
		return fmt.Errorf("%w: manifest: %v", ErrCLIFixtureMalformed, err)
	}
	var manifest cliManifestEnvelope
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("%w: manifest JSON", ErrCLIFixtureMalformed)
	}
	if manifest.Domain != cliFixtureDomain {
		return fmt.Errorf("%w: %q", ErrCLIFixtureDomain, manifest.Domain)
	}
	if !bytes.Equal(manifestBytes, []byte(cliManifest)) || manifest.SchemaVersion != cliFixtureSchema ||
		manifest.Entrypoint != CLIEntrypoint || !equalCLIRoles(manifest.Roles, CLIRoles()) {
		return fmt.Errorf("%w: manifest bytes", ErrCLIFixtureMalformed)
	}

	expected := expectedCLIHeadFiles()
	allowedDirectories := map[string]struct{}{"authority": {}, "authority/cli": {}}
	for _, role := range CLIRoles() {
		allowedDirectories["authority/cli/"+string(role)] = struct{}{}
	}
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
				return errors.New(".git is not a private real directory")
			}
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link %s", relative)
		}
		if entry.IsDir() {
			info, infoErr := entry.Info()
			if _, admitted := allowedDirectories[relative]; !admitted {
				return fmt.Errorf("unexpected directory %s", relative)
			}
			if infoErr != nil || info.Mode().Perm() != 0o700 || hasSpecialMode(info.Mode()) {
				return fmt.Errorf("non-private directory %s", relative)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular entry %s", relative)
		}
		if relative == cliIgnoredBinary {
			info, infoErr := entry.Info()
			if infoErr != nil || info.Mode().Perm() != 0o700 || hasSpecialMode(info.Mode()) {
				return errors.New("ignored reference binary is not exact executable mode 0700")
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
		return fmt.Errorf("%w: %v", ErrCLIFixtureMalformed, err)
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("%w: worktree file roster %d/%d", ErrCLIFixtureMalformed, len(seen), len(expected))
	}
	return nil
}

func expectedCLIHeadFiles() map[string]fixtureFile {
	result := map[string]fixtureFile{
		".gitignore":    {mode: "100644", content: []byte(cliGitignore)},
		cliManifestPath: {mode: "100644", content: []byte(cliManifest)},
	}
	for _, role := range CLIRoles() {
		files, err := expectedCLICandidateFiles(role)
		if err != nil {
			panic(err)
		}
		for path, file := range files {
			result["authority/cli/"+string(role)+"/"+path] = fixtureFile{
				mode: file.mode, content: append([]byte(nil), file.content...),
			}
		}
	}
	return result
}

func expectedCLICandidateFiles(role CLIRole) (map[string]fixtureFile, error) {
	files, err := clifixture.CandidateFiles(role)
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

func equalCLIRoles(left, right []CLIRole) bool {
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

type cliCommitAuthority struct {
	oid     string
	tree    string
	message string
}

func validateCLIRepositoryConfig(root string) error {
	path := filepath.Join(root, ".git", "config")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o600 || hasSpecialMode(info.Mode()) {
		return errors.New("config is not a private regular file")
	}
	content, err := readRegularFile(path, int64(len(cliGitConfig)))
	if err != nil || !bytes.Equal(content, []byte(cliGitConfig)) {
		return errors.New("config bytes differ")
	}
	return nil
}

func validateCLIGitAuthority(root string, commits map[string]cliCommitAuthority) error {
	if len(commits) != len(CLIRoles())+1 {
		return errors.New("commit authority roster is incomplete")
	}
	if err := validateCLIRepositoryConfig(root); err != nil {
		return err
	}
	headPath := filepath.Join(root, ".git", "HEAD")
	headInfo, err := os.Lstat(headPath)
	if err != nil || !headInfo.Mode().IsRegular() || headInfo.Mode().Perm() != 0o600 || hasSpecialMode(headInfo.Mode()) {
		return errors.New("HEAD is not a private regular file")
	}
	head, err := readRegularFile(headPath, 256)
	if err != nil || !bytes.Equal(head, []byte("ref: "+cliFixtureMainRef+"\n")) {
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
		if objectErr != nil || !bytes.Equal(commit, exactCLICommit(authority.tree, authority.message)) {
			return fmt.Errorf("commit %s is not the exact parentless authority", ref)
		}
	}
	seen := make(map[string]struct{}, len(expectedRefs))
	refsRoot := filepath.Join(root, ".git", "refs")
	refsInfo, err := os.Lstat(refsRoot)
	if err != nil || !refsInfo.IsDir() || refsInfo.Mode().Perm() != 0o700 || hasSpecialMode(refsInfo.Mode()) {
		return errors.New("refs root is not an exact private directory")
	}
	allowedRefDirectories := map[string]struct{}{
		"heads": {},
		"tags":  {},
	}
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
			if _, admitted := allowedRefDirectories[relative]; !admitted {
				return fmt.Errorf("unexpected ref directory %s", relative)
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

func exactCLICommit(tree, message string) []byte {
	return []byte(fmt.Sprintf(
		"tree %s\nauthor Countershape U7 Fixture <u7-fixture@countershape.invalid> 946684800 +0000\n"+
			"committer Countershape U7 Fixture <u7-fixture@countershape.invalid> 946684800 +0000\n\n%s\n",
		tree, message,
	))
}

type exactTreeNode struct {
	directories map[string]*exactTreeNode
	files       map[string]fixtureFile
}

func exactGitTreeOID(files map[string]fixtureFile) (string, error) {
	root := &exactTreeNode{directories: make(map[string]*exactTreeNode), files: make(map[string]fixtureFile)}
	for path, file := range files {
		if path == "" || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(path)) != path || strings.HasPrefix(path, "../") {
			return "", fmt.Errorf("invalid fixture path %q", path)
		}
		parts := strings.Split(path, "/")
		node := root
		for _, part := range parts[:len(parts)-1] {
			if part == "" {
				return "", fmt.Errorf("invalid fixture path %q", path)
			}
			next := node.directories[part]
			if next == nil {
				next = &exactTreeNode{directories: make(map[string]*exactTreeNode), files: make(map[string]fixtureFile)}
				node.directories[part] = next
			}
			node = next
		}
		name := parts[len(parts)-1]
		if name == "" || (file.mode != "100644" && file.mode != "100755") {
			return "", fmt.Errorf("invalid fixture entry %q", path)
		}
		node.files[name] = file
	}
	return exactTreeNodeOID(root)
}

func exactTreeNodeOID(node *exactTreeNode) (string, error) {
	type entry struct {
		name      string
		mode      string
		oid       string
		directory bool
	}
	entries := make([]entry, 0, len(node.directories)+len(node.files))
	for name, child := range node.directories {
		oid, err := exactTreeNodeOID(child)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry{name: name, mode: "40000", oid: oid, directory: true})
	}
	for name, file := range node.files {
		entries = append(entries, entry{name: name, mode: file.mode, oid: gitBlobOID(file.content)})
	}
	sort.Slice(entries, func(left, right int) bool {
		leftName, rightName := entries[left].name, entries[right].name
		if entries[left].directory {
			leftName += "/"
		}
		if entries[right].directory {
			rightName += "/"
		}
		return leftName < rightName
	})
	var payload bytes.Buffer
	for _, entry := range entries {
		decoded, err := hex.DecodeString(entry.oid)
		if err != nil || len(decoded) != 20 {
			return "", errors.New("tree entry OID is malformed")
		}
		_, _ = fmt.Fprintf(&payload, "%s %s%c", entry.mode, entry.name, byte(0))
		_, _ = payload.Write(decoded)
	}
	return gitObjectOID("tree", payload.Bytes()), nil
}

func gitObjectOID(kind string, payload []byte) string {
	// SHA-1 is the fixture repository's explicitly admitted Git object format.
	hash := sha1.New()
	_, _ = fmt.Fprintf(hash, "%s %d%c", kind, len(payload), byte(0))
	_, _ = hash.Write(payload)
	return hex.EncodeToString(hash.Sum(nil))
}
