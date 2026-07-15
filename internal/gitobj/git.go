package gitobj

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	gitTextOutputLimit            = 4 << 20
	gitTreeOutputLimit            = 64 << 20
	maxObjectPackDirectoryEntries = 8192 // closed physical-input bound, including non-pack entries
	objectPackDirectoryReadBatch  = 128  // memory and iteration page bound below the total count bound

	disableReplacementInterpretation  = true // MUTANT_U2_ENABLE_REPLACEMENT_REFS
	rejectPromisorAndDisableLazyFetch = true // MUTANT_U2_ALLOW_LAZY_FETCH
	rejectAlternateObjectDirectories  = true // MUTANT_U2_ALLOW_ALTERNATE_OBJECTS
)

type limitedBuffer struct {
	bytes.Buffer
	limit int
	over  bool
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := b.limit - b.Len()
	if remaining < len(data) {
		b.over = true
		if remaining > 0 {
			_, _ = b.Buffer.Write(data[:remaining])
		}
		return original, nil
	}
	_, _ = b.Buffer.Write(data)
	return original, nil
}

type closedGit struct {
	executable         string
	repository         string
	home               string
	directGitDirectory bool
}

func sparseGitEnvironment(home, executable string) []string {
	environment := []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, "xdg"),
		"TMPDIR=" + filepath.Join(home, "tmp"),
		"PATH=" + filepath.Dir(executable),
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/usr/bin/false",
		"SSH_ASKPASS=/usr/bin/false",
		"GIT_SSH_COMMAND=/usr/bin/false",
		"GIT_PROTOCOL_FROM_USER=0",
		"GIT_OPTIONAL_LOCKS=0",
	}
	if disableReplacementInterpretation {
		environment = append(environment, "GIT_NO_REPLACE_OBJECTS=1")
	}
	if rejectPromisorAndDisableLazyFetch {
		environment = append(environment, "GIT_NO_LAZY_FETCH=1")
	}
	return environment
}

func (g closedGit) baseArgs(inRepository bool) []string {
	args := []string{"--no-pager"}
	if disableReplacementInterpretation {
		args = append(args, "--no-replace-objects")
	}
	args = append(args, "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null", "-c", "credential.helper=")
	if inRepository {
		if g.directGitDirectory {
			args = append(args, "--git-dir="+g.repository)
		} else {
			args = append(args, "-C", g.repository)
		}
	}
	return args
}

func (g closedGit) command(ctx context.Context, inRepository bool, args ...string) *exec.Cmd {
	closedArgs := append(g.baseArgs(inRepository), args...)
	command := exec.CommandContext(ctx, g.executable, closedArgs...)
	command.Dir = g.home
	command.Env = sparseGitEnvironment(g.home, g.executable)
	return command
}

func (g closedGit) run(ctx context.Context, inRepository bool, outputLimit int, args ...string) ([]byte, error) {
	command := g.command(ctx, inRepository, args...)
	stdout := &limitedBuffer{limit: outputLimit}
	stderr := &limitedBuffer{limit: 64 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return nil, refuse(CodeGitCommandFailed, boundedGitFailure(args, stderr.String(), err), err)
	}
	if stdout.over || stderr.over {
		return nil, refuse(CodeGitCommandFailed, "Git plumbing output exceeded its closed bound", nil)
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func boundedGitFailure(args []string, stderr string, err error) string {
	operation := "unknown"
	if len(args) > 0 {
		operation = args[0]
	}
	text := strings.TrimSpace(stderr)
	if len(text) > 512 {
		text = text[:512]
	}
	if text == "" {
		text = err.Error()
	}
	return operation + ": " + text
}

type toolAdmission struct {
	path       string
	filesystem filesystemIdentity
	digest     domain.Digest
}

func admitExecutable(raw string) (toolAdmission, error) {
	if !filepath.IsAbs(raw) {
		return toolAdmission{}, refuse(CodeGitToolRejected, "Git executable must be absolute", nil)
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return toolAdmission{}, refuse(CodeGitToolRejected, "Git executable cannot be resolved", err)
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return toolAdmission{}, refuse(CodeGitToolRejected, "Git executable is not an executable regular file", err)
	}
	filesystem, err := filesystemIdentityAt(resolved)
	if err != nil {
		return toolAdmission{}, refuse(CodeGitToolRejected, "Git executable identity cannot be measured", err)
	}
	parsed, err := digestFile("GitExecutableBytes", resolved, 512<<20)
	if err != nil {
		return toolAdmission{}, refuse(CodeGitToolRejected, "Git executable bytes cannot be bounded and digested", err)
	}
	return toolAdmission{path: resolved, filesystem: filesystem, digest: parsed}, nil
}

func canonicalDirectory(raw string) (string, error) {
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("not a directory: %w", err)
	}
	return resolved, nil
}

func digestFile(kind, path string, limit int64) (domain.Digest, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return "", fmt.Errorf("identity file is not a regular file: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest, err := canon.DigestBytes(kind, data)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func rejectRepositoryPolicy(ctx context.Context, runner closedGit) error {
	output, err := runner.run(ctx, true, gitTextOutputLimit, "config", "--local", "--null", "--name-only", "--list", "--no-includes")
	if err != nil {
		return refuse(CodeRepositoryRejected, "repository-local configuration cannot be classified", err)
	}
	for _, raw := range bytes.Split(output, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		if !utf8.Valid(raw) || bytes.IndexByte(raw, '\n') >= 0 || bytes.IndexByte(raw, '\r') >= 0 {
			return refuse(CodeRepositoryRejected, "repository-local configuration key is malformed", nil)
		}
		key := strings.ToLower(string(raw))
		unsupportedExtension := strings.HasPrefix(key, "extensions.") && key != "extensions.objectformat"
		partialOrPromisor := rejectPromisorAndDisableLazyFetch && (key == "extensions.partialclone" ||
			(strings.HasPrefix(key, "remote.") && (strings.HasSuffix(key, ".promisor") || strings.HasSuffix(key, ".partialclonefilter"))))
		includedConfig := strings.HasPrefix(key, "include.") || strings.HasPrefix(key, "includeif.")
		if unsupportedExtension || partialOrPromisor || includedConfig {
			return refuse(CodeRepositoryRejected, "repository uses an unsupported local configuration capability: "+key, nil)
		}
	}
	return nil
}

func oneLine(output []byte) (string, error) {
	if len(output) == 0 || bytes.IndexByte(output, 0) >= 0 {
		return "", errors.New("empty or NUL-bearing Git output")
	}
	line := strings.TrimSuffix(string(output), "\n")
	if line == "" || strings.ContainsAny(line, "\r\n") || !utf8.ValidString(line) {
		return "", errors.New("Git output is not one UTF-8 line")
	}
	return line, nil
}

func OpenRepository(ctx context.Context, config OpenConfig) (Repository, error) {
	tool, err := admitExecutable(config.GitExecutable)
	if err != nil {
		return Repository{}, err
	}
	scratch, err := canonicalDirectory(config.ScratchRoot)
	if err != nil {
		return Repository{}, refuse(CodeInvalidConfig, "scratch root is unavailable", err)
	}
	home, err := os.MkdirTemp(scratch, ".countershape-git-home-")
	if err != nil {
		return Repository{}, refuse(CodeInvalidConfig, "private Git home allocation failed", err)
	}
	if err := os.Chmod(home, 0o700); err != nil {
		_ = os.RemoveAll(home)
		return Repository{}, refuse(CodeInvalidConfig, "private Git home mode failed", err)
	}
	opened := false
	defer func() {
		if !opened {
			_ = os.RemoveAll(home)
		}
	}()
	for _, directory := range []string{filepath.Join(home, "xdg"), filepath.Join(home, "tmp")} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			_ = os.RemoveAll(home)
			return Repository{}, refuse(CodeInvalidConfig, "private Git environment allocation failed", err)
		}
	}
	repository, err := canonicalDirectory(config.Repository)
	if err != nil {
		_ = os.RemoveAll(home)
		return Repository{}, refuse(CodeRepositoryRejected, "repository path is unavailable", err)
	}
	runner := closedGit{executable: tool.path, repository: repository, home: home}
	versionBytes, err := runner.run(ctx, false, gitTextOutputLimit, "version")
	if err != nil {
		return Repository{}, err
	}
	version, err := oneLine(versionBytes)
	if err != nil || !strings.HasPrefix(version, "git version ") {
		return Repository{}, refuse(CodeGitToolRejected, "Git version output is outside the closed profile", err)
	}
	commonBytes, err := runner.run(ctx, true, gitTextOutputLimit, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return Repository{}, err
	}
	commonRaw, err := oneLine(commonBytes)
	if err != nil {
		return Repository{}, refuse(CodeRepositoryRejected, "Git common directory output is malformed", err)
	}
	common, err := canonicalDirectory(commonRaw)
	if err != nil {
		return Repository{}, refuse(CodeRepositoryRejected, "Git common directory is unavailable", err)
	}
	objectBytes, err := runner.run(ctx, true, gitTextOutputLimit, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	if err != nil {
		return Repository{}, err
	}
	objectRaw, err := oneLine(objectBytes)
	if err != nil {
		return Repository{}, refuse(CodeRepositoryRejected, "Git object directory output is malformed", err)
	}
	objectDirectory, err := canonicalDirectory(objectRaw)
	if err != nil {
		return Repository{}, refuse(CodeRepositoryRejected, "Git object directory is unavailable", err)
	}
	// All object/config operations after discovery address the measured Git
	// directory directly and never rediscover through a working tree.
	runner.repository = common
	runner.directGitDirectory = true
	if rejectAlternateObjectDirectories {
		if err := rejectAlternates(objectDirectory); err != nil {
			return Repository{}, err
		}
	}
	if err := rejectPromisorMarkers(objectDirectory); err != nil {
		return Repository{}, err
	}
	formatBytes, err := runner.run(ctx, true, gitTextOutputLimit, "rev-parse", "--show-object-format=storage")
	if err != nil {
		return Repository{}, err
	}
	formatRaw, err := oneLine(formatBytes)
	format := ObjectFormat(formatRaw)
	if err != nil || !format.valid() {
		return Repository{}, refuse(CodeUnsupportedFormat, "repository object format is not SHA-1 or SHA-256", err)
	}
	commonIdentity, err := filesystemIdentityAt(common)
	if err != nil {
		return Repository{}, refuse(CodeUnsupportedPlatform, "cannot measure Git common directory", err)
	}
	objectIdentity, err := filesystemIdentityAt(objectDirectory)
	if err != nil {
		return Repository{}, refuse(CodeUnsupportedPlatform, "cannot measure Git object directory", err)
	}
	if err := rejectRepositoryPolicy(ctx, runner); err != nil {
		return Repository{}, err
	}
	configPath := filepath.Join(common, "config")
	configDigest, err := digestFile("RepositoryLocalConfigBytes", configPath, 4<<20)
	if err != nil {
		return Repository{}, refuse(CodeRepositoryRejected, "repository-local config bytes cannot be measured", err)
	}
	fingerprint, err := digestIdentity("RepositoryHandleReceipt", struct {
		SchemaVersion string             `json:"schema_version"`
		Kind          string             `json:"kind"`
		CommonPath    string             `json:"common_directory_path"`
		ObjectPath    string             `json:"object_directory_path"`
		ObjectFormat  ObjectFormat       `json:"object_format"`
		Common        filesystemIdentity `json:"common_directory"`
		Objects       filesystemIdentity `json:"object_directory"`
		GitPath       string             `json:"git_executable_path"`
		GitFilesystem filesystemIdentity `json:"git_executable_filesystem"`
		GitDigest     string             `json:"git_executable_digest"`
		GitVersion    string             `json:"git_version"`
		ConfigPath    string             `json:"config_path"`
		ConfigDigest  string             `json:"config_digest"`
		PolicyProfile string             `json:"repository_policy_profile"`
	}{
		domain.SchemaVersion, "RepositoryHandleReceipt", common, objectDirectory, format, commonIdentity, objectIdentity,
		tool.path, tool.filesystem, tool.digest.String(), version, configPath, configDigest.String(), "closed-primary-object-database/v1",
	})
	if err != nil {
		return Repository{}, err
	}
	result := Repository{state: &repositoryState{
		runner: runner, fingerprint: fingerprint, objectFormat: format, commonDirectory: common,
		objectDirectory: objectDirectory, commonFilesystem: commonIdentity, objectFilesystem: objectIdentity,
		gitExecutableIdentity: tool.filesystem, gitExecutableDigest: tool.digest,
		repositoryConfigPath: configPath, repositoryConfigDigest: configDigest,
		gitVersion: version, repositoryDisplayPath: repository,
	}}
	opened = true
	return result, nil
}

// Close revokes every copy of this repository capability and removes its
// private Git HOME. It never deletes the caller-owned object database.
func (r Repository) Close() error {
	if r.state == nil || !r.state.closed.CompareAndSwap(false, true) {
		return nil
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	err := os.RemoveAll(r.state.runner.home)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func rejectAlternates(objectDirectory string) error {
	path := filepath.Join(objectDirectory, "info", "alternates")
	_, err := os.Lstat(path)
	if err == nil {
		return refuse(CodeAlternatesRejected, "objects/info/alternates exists", nil)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return refuse(CodeAlternatesRejected, "cannot prove alternates are absent", err)
	}
	return nil
}

func rejectPromisorMarkers(objectDirectory string) error {
	packDirectory := filepath.Join(objectDirectory, "pack")
	handle, err := os.Open(packDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return refuse(CodeRepositoryRejected, "object pack directory cannot be classified", err)
	}
	classificationErr := classifyPromisorPackDirectory(handle)
	closeErr := handle.Close()
	if classificationErr != nil {
		return classificationErr
	}
	if closeErr != nil {
		return refuse(CodeRepositoryRejected, "object pack directory cannot be closed after classification", closeErr)
	}
	return nil
}

type boundedDirectoryEntryReader interface {
	ReadDir(int) ([]os.DirEntry, error)
}

func classifyPromisorPackDirectory(reader boundedDirectoryEntryReader) error {
	classified := 0
	for {
		remaining := maxObjectPackDirectoryEntries - classified
		request := objectPackDirectoryReadBatch
		if request > remaining+1 {
			request = remaining + 1
		}
		entries, err := reader.ReadDir(request)
		if len(entries) > request {
			return refuse(CodeRepositoryRejected, "object pack directory reader exceeded its requested bound", nil)
		}
		for _, entry := range entries {
			classified++
			if classified > maxObjectPackDirectoryEntries {
				return refuse(CodeRepositoryRejected, "object pack directory exceeds its closed entry-count bound", nil)
			}
			if entry == nil {
				return refuse(CodeRepositoryRejected, "object pack directory returned an invalid entry", nil)
			}
			if strings.HasSuffix(entry.Name(), ".promisor") {
				return refuse(CodeRepositoryRejected, "object database contains a promisor pack marker", nil)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return refuse(CodeRepositoryRejected, "object pack directory cannot be classified", err)
		}
		if len(entries) == 0 {
			return refuse(CodeRepositoryRejected, "object pack directory classification made no progress", nil)
		}
	}
}

func (state *repositoryState) revalidate() error {
	if state.closed.Load() {
		return refuse(CodeSourceChanged, "Git repository capability is closed", nil)
	}
	common, err := filesystemIdentityAt(state.commonDirectory)
	if err != nil || !common.Equal(state.commonFilesystem) {
		return refuse(CodeSourceChanged, "Git common directory identity changed", err)
	}
	objects, err := filesystemIdentityAt(state.objectDirectory)
	if err != nil || !objects.Equal(state.objectFilesystem) {
		return refuse(CodeSourceChanged, "Git object directory identity changed", err)
	}
	if rejectAlternateObjectDirectories {
		if err := rejectAlternates(state.objectDirectory); err != nil {
			return err
		}
	}
	if err := rejectPromisorMarkers(state.objectDirectory); err != nil {
		return err
	}
	gitIdentity, err := filesystemIdentityAt(state.runner.executable)
	if err != nil || !gitIdentity.Equal(state.gitExecutableIdentity) {
		return refuse(CodeSourceChanged, "Git executable identity changed", err)
	}
	gitDigest, err := digestFile("GitExecutableBytes", state.runner.executable, 512<<20)
	if err != nil || gitDigest != state.gitExecutableDigest {
		return refuse(CodeSourceChanged, "Git executable bytes changed", err)
	}
	configDigest, err := digestFile("RepositoryLocalConfigBytes", state.repositoryConfigPath, 4<<20)
	if err != nil || configDigest != state.repositoryConfigDigest {
		return refuse(CodeSourceChanged, "repository-local config changed", err)
	}
	return nil
}

func (state *repositoryState) run(ctx context.Context, outputLimit int, args ...string) ([]byte, error) {
	if err := state.revalidate(); err != nil {
		return nil, err
	}
	return state.runner.run(ctx, true, outputLimit, args...)
}

func validDisplayRef(ref string) bool {
	if len(ref) < 1 || len(ref) > 1024 || !utf8.ValidString(ref) || strings.HasPrefix(ref, "-") || strings.TrimSpace(ref) != ref {
		return false
	}
	if !validOID(ObjectSHA1, ref) && !validOID(ObjectSHA256, ref) &&
		!strings.HasPrefix(ref, "refs/heads/") && !strings.HasPrefix(ref, "refs/tags/") {
		return false
	}
	if ref == "@" || strings.Contains(ref, "..") || strings.Contains(ref, "@{") || strings.HasSuffix(ref, ".") ||
		strings.ContainsAny(ref, "~^:?*[\\") || strings.HasSuffix(ref, "/") || strings.Contains(ref, "//") {
		return false
	}
	if !validOID(ObjectSHA1, ref) && !validOID(ObjectSHA256, ref) {
		for _, component := range strings.Split(ref, "/") {
			if component == "" || strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
				return false
			}
		}
	}
	for _, character := range ref {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return false
		}
	}
	return true
}

func (r Repository) Pin(ctx context.Context, displayRef string) (PinnedTree, error) {
	if !r.Valid() || !validDisplayRef(displayRef) {
		return PinnedTree{}, refuse(CodeInvalidRef, "display ref is outside the closed profile", nil)
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if err := r.state.revalidate(); err != nil {
		return PinnedTree{}, err
	}
	commitBytes, err := r.state.run(ctx, gitTextOutputLimit, "rev-parse", "--verify", "--end-of-options", displayRef+"^{commit}")
	if err != nil {
		return PinnedTree{}, refuse(CodeInvalidRef, "display ref did not resolve to a commit", err)
	}
	commitOID, err := oneLine(commitBytes)
	if err != nil || !validOID(r.state.objectFormat, commitOID) {
		return PinnedTree{}, refuse(CodeInvalidRef, "resolved commit OID is malformed", err)
	}
	verifiedCommit, err := r.state.readVerifiedObject(ctx, "commit", commitOID, 16<<20)
	if err != nil {
		return PinnedTree{}, err
	}
	treeOID, err := treeOIDFromCommit(r.state.objectFormat, verifiedCommit)
	if err != nil {
		return PinnedTree{}, err
	}
	if err := r.state.verifyObjectOnly(ctx, "tree", treeOID, gitTreeOutputLimit); err != nil {
		return PinnedTree{}, err
	}
	identity, err := digestIdentity("PinnedTreeIdentity", struct {
		SchemaVersion string       `json:"schema_version"`
		Kind          string       `json:"kind"`
		ObjectFormat  ObjectFormat `json:"object_format"`
		CommitOID     string       `json:"commit_oid"`
		TreeOID       string       `json:"tree_oid"`
	}{domain.SchemaVersion, "PinnedTreeIdentity", r.state.objectFormat, commitOID, treeOID})
	if err != nil {
		return PinnedTree{}, err
	}
	return PinnedTree{repository: r.state, identity: identity, commitOID: commitOID, treeOID: treeOID, displayRef: displayRef}, nil
}

func (state *repositoryState) objectSize(ctx context.Context, oid string) (int64, error) {
	output, err := state.run(ctx, gitTextOutputLimit, "cat-file", "-s", oid)
	if err != nil {
		return 0, refuse(CodeMissingObject, "blob size is unavailable", err)
	}
	line, err := oneLine(output)
	if err != nil {
		return 0, refuse(CodeMissingObject, "blob size output is malformed", err)
	}
	size, err := strconv.ParseInt(line, 10, 64)
	if err != nil || size < 0 {
		return 0, refuse(CodeMissingObject, "blob size is invalid", err)
	}
	return size, nil
}

func (state *repositoryState) streamObject(ctx context.Context, objectType, oid string, expectedSize int64, destination io.Writer) error {
	if !validOID(state.objectFormat, oid) || expectedSize < 0 {
		return refuse(CodeMissingObject, "invalid object request", nil)
	}
	if objectType != "blob" && objectType != "tree" && objectType != "commit" && objectType != "tag" {
		return refuse(CodeMissingObject, "unsupported object type request", nil)
	}
	if err := state.revalidate(); err != nil {
		return err
	}
	command := state.runner.command(ctx, true, "cat-file", objectType, oid)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return refuse(CodeGitCommandFailed, "cannot allocate Git object pipe", err)
	}
	stderr := &limitedBuffer{limit: 64 << 10}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		return refuse(CodeMissingObject, "object stream did not start", err)
	}
	limited := &io.LimitedReader{R: stdout, N: expectedSize + 1}
	written, copyErr := io.Copy(destination, limited)
	drainErr := stdout.Close()
	waitErr := command.Wait()
	if copyErr != nil || drainErr != nil || waitErr != nil || written != expectedSize || limited.N != 1 || stderr.over {
		cause := copyErr
		if cause == nil {
			cause = drainErr
		}
		if cause == nil {
			cause = waitErr
		}
		return refuse(CodeMissingObject, fmt.Sprintf("object stream mismatch: got %d bytes, want %d", written, expectedSize), cause)
	}
	if err := state.revalidate(); err != nil {
		return err
	}
	return nil
}
