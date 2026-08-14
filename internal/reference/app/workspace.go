package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	maximumSourceBytes        = 1 << 20
	maximumStudyArtifactBytes = 16 << 20
	maximumStudyEvidenceBytes = 64 << 20
	maximumStudyEvidenceFiles = 4096
	maximumStudyEvidenceDirs  = 1024
)

type InputError struct {
	Code   string
	Detail string
}

func (e *InputError) Error() string { return e.Code + ": " + e.Detail }

type sourceInput struct {
	displayPath string
	argument    string
	exact       []byte
}

// EvidenceWorkspace binds exclusive writes to the already-existing evidence
// directory admitted by the frozen harness. It never creates, replaces, or
// removes that root. All child creation is relative, no-overwrite, and private.
type EvidenceWorkspace struct {
	mu         sync.Mutex
	root       *os.Root
	rootPath   string
	closed     bool
	validation *studyEvidenceValidation
}

type studyEvidenceValidation struct {
	snapshot StudyEvidenceSnapshot
	commit   StudyTerminalCommit
}

func newEvidenceWorkspace(path string, expected os.FileInfo) (*EvidenceWorkspace, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("open evidence workspace: %w", err)
	}
	opened, err := root.Lstat(".")
	if err != nil || !opened.IsDir() || opened.Mode()&os.ModeSymlink != 0 ||
		opened.Mode().Perm() != 0o700 || hasSpecialMode(opened.Mode()) ||
		opened.Mode() != expected.Mode() || !os.SameFile(expected, opened) {
		_ = root.Close()
		return nil, errors.New("evidence workspace identity changed while opening")
	}
	return &EvidenceWorkspace{root: root, rootPath: path}, nil
}

// CreateDirectory creates exactly one private relative directory. Parents
// must already exist and must be private, non-link directories.
func (workspace *EvidenceWorkspace) CreateDirectory(relative string) error {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if err := workspace.readyRelative(relative); err != nil {
		return err
	}
	if err := workspace.requirePrivateParents(relative); err != nil {
		return err
	}
	if err := workspace.root.Mkdir(relative, 0o700); err != nil {
		return fmt.Errorf("create evidence directory: %w", err)
	}
	created, err := workspace.root.Lstat(relative)
	if err != nil || !created.IsDir() || created.Mode()&os.ModeSymlink != 0 || created.Mode().Perm() != 0o700 ||
		created.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return errors.New("created evidence directory is not exact and private")
	}
	return nil
}

// WriteFileExclusive writes one bounded artifact without following or
// replacing a terminal path. A failed write is left in place so the attempt
// cannot be mistaken for complete evidence.
func (workspace *EvidenceWorkspace) WriteFileExclusive(relative string, exact []byte) error {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if len(exact) < 1 || len(exact) > maximumStudyArtifactBytes {
		return errors.New("evidence artifact must contain 1..16777216 bytes")
	}
	if err := workspace.readyRelative(relative); err != nil {
		return err
	}
	if err := workspace.requirePrivateParents(relative); err != nil {
		return err
	}
	handle, err := workspace.root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create evidence artifact: %w", err)
	}
	written, writeErr := handle.Write(exact)
	if writeErr == nil && written != len(exact) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = handle.Sync()
	}
	opened, statErr := handle.Stat()
	closeErr := handle.Close()
	if writeErr != nil {
		return fmt.Errorf("write evidence artifact: %w", writeErr)
	}
	if statErr != nil || closeErr != nil || !opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 ||
		opened.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || !singleLink(opened) || opened.Size() != int64(len(exact)) {
		return errors.New("written evidence artifact is not exact and private")
	}
	terminal, err := workspace.root.Lstat(relative)
	if err != nil || terminal.Mode()&os.ModeSymlink != 0 || !terminal.Mode().IsRegular() || !singleLink(terminal) || !os.SameFile(opened, terminal) {
		return errors.New("evidence artifact identity changed after writing")
	}
	return nil
}

// VerifyExactManifest independently reopens the complete terminal workspace
// as raw bytes. It verifies only private filesystem shape and byte identity;
// it does not parse artifacts or manufacture study-semantic authority.
func (workspace *EvidenceWorkspace) VerifyExactManifest(manifest EvidenceManifest) error {
	if workspace == nil {
		return errors.New("evidence workspace is closed")
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if workspace.root == nil || workspace.closed {
		return errors.New("evidence workspace is closed")
	}
	expectedDirectories, expectedFiles, err := validateEvidenceManifest(manifest)
	if err != nil {
		return err
	}
	observedDirectories, observedFiles, err := workspace.inspectEvidenceTree()
	if err != nil {
		return err
	}
	if !equalStrings(observedDirectories, expectedDirectories) || !equalStrings(observedFiles, sortedEvidenceFilePaths(expectedFiles)) {
		return errors.New("evidence workspace terminal roster differs from the exact manifest")
	}
	openedFiles := make([]os.FileInfo, 0, len(expectedFiles))
	for _, expected := range expectedFiles {
		opened, err := workspace.verifyEvidenceFile(expected)
		if err != nil {
			return err
		}
		for _, previous := range openedFiles {
			if os.SameFile(previous, opened) {
				return errors.New("evidence manifest files must not be hard-link aliases")
			}
		}
		openedFiles = append(openedFiles, opened)
	}
	terminalDirectories, terminalFiles, err := workspace.inspectEvidenceTree()
	if err != nil {
		return err
	}
	if !equalStrings(terminalDirectories, expectedDirectories) || !equalStrings(terminalFiles, sortedEvidenceFilePaths(expectedFiles)) {
		return errors.New("evidence workspace changed after manifest verification")
	}
	return nil
}

// SnapshotEvidence captures the complete terminal tree through the retained
// os.Root capability. It is deliberately ignorant of the frozen 111-file
// domain protocol: it establishes only bounded private filesystem shape, raw
// bytes, and byte identities for a stricter consumer to interpret.
func (workspace *EvidenceWorkspace) SnapshotEvidence() (StudyEvidenceSnapshot, error) {
	if workspace == nil {
		return StudyEvidenceSnapshot{}, errors.New("evidence workspace is closed")
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if workspace.root == nil || workspace.closed {
		return StudyEvidenceSnapshot{}, errors.New("evidence workspace is closed")
	}
	return workspace.snapshotEvidenceLocked()
}

func (workspace *EvidenceWorkspace) snapshotEvidenceLocked() (StudyEvidenceSnapshot, error) {
	directories, files, err := workspace.inspectEvidenceTree()
	if err != nil {
		return StudyEvidenceSnapshot{}, err
	}
	if len(files) == 0 || len(files) > maximumStudyEvidenceFiles || len(directories) > maximumStudyEvidenceDirs {
		return StudyEvidenceSnapshot{}, errors.New("evidence workspace entry count is outside the bounded snapshot protocol")
	}
	entries := make([]StudyEvidenceEntry, 0, len(files))
	openedFiles := make([]os.FileInfo, 0, len(files))
	var totalBytes int64
	for _, relative := range files {
		entry, opened, readErr := workspace.snapshotEvidenceFile(relative)
		if readErr != nil {
			return StudyEvidenceSnapshot{}, readErr
		}
		if totalBytes > maximumStudyEvidenceBytes-entry.bytes {
			return StudyEvidenceSnapshot{}, errors.New("evidence workspace exceeds the bounded snapshot byte budget")
		}
		for _, previous := range openedFiles {
			if os.SameFile(previous, opened) {
				return StudyEvidenceSnapshot{}, errors.New("evidence workspace files must not be hard-link aliases")
			}
		}
		totalBytes += entry.bytes
		entries = append(entries, entry)
		openedFiles = append(openedFiles, opened)
	}
	terminalDirectories, terminalFiles, err := workspace.inspectEvidenceTree()
	if err != nil {
		return StudyEvidenceSnapshot{}, err
	}
	if !equalStrings(directories, terminalDirectories) || !equalStrings(files, terminalFiles) {
		return StudyEvidenceSnapshot{}, errors.New("evidence workspace changed after terminal snapshot")
	}
	terminalFilesInfo := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		opened, verifyErr := workspace.verifyEvidenceFile(EvidenceFile{
			Path: entry.path, Bytes: entry.bytes, SHA256: entry.sha256,
		})
		if verifyErr != nil {
			return StudyEvidenceSnapshot{}, verifyErr
		}
		for _, previous := range terminalFilesInfo {
			if os.SameFile(previous, opened) {
				return StudyEvidenceSnapshot{}, errors.New("evidence workspace files must not be hard-link aliases")
			}
		}
		terminalFilesInfo = append(terminalFilesInfo, opened)
	}
	closedDirectories, closedFiles, err := workspace.inspectEvidenceTree()
	if err != nil {
		return StudyEvidenceSnapshot{}, err
	}
	if !equalStrings(directories, closedDirectories) || !equalStrings(files, closedFiles) {
		return StudyEvidenceSnapshot{}, errors.New("evidence workspace changed during terminal byte verification")
	}
	return StudyEvidenceSnapshot{
		directories: append([]string(nil), directories...),
		files:       cloneStudyEvidenceEntries(entries),
		totalBytes:  totalBytes,
	}, nil
}

func (workspace *EvidenceWorkspace) bindValidatedSnapshot(snapshot StudyEvidenceSnapshot, commit StudyTerminalCommit) error {
	if workspace == nil {
		return errors.New("evidence workspace is closed")
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if workspace.root == nil || workspace.closed {
		return errors.New("evidence workspace is closed")
	}
	if commit == nil {
		return errors.New("terminal evidence commit is absent")
	}
	if workspace.validation != nil {
		return errors.New("terminal evidence snapshot was already bound")
	}
	workspace.validation = &studyEvidenceValidation{
		snapshot: cloneStudyEvidenceSnapshot(snapshot),
		commit:   commit,
	}
	return nil
}

func (workspace *EvidenceWorkspace) snapshotEvidenceFile(relative string) (StudyEvidenceEntry, os.FileInfo, error) {
	before, err := workspace.root.Lstat(relative)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Mode().Perm() != 0o600 ||
		hasSpecialMode(before.Mode()) || !singleLink(before) || before.Size() < 1 || before.Size() > maximumStudyArtifactBytes {
		return StudyEvidenceEntry{}, nil, fmt.Errorf("evidence file is not exact and private: %s", relative)
	}
	handle, err := workspace.root.Open(relative)
	if err != nil {
		return StudyEvidenceEntry{}, nil, fmt.Errorf("open evidence file: %w", err)
	}
	opened, statErr := handle.Stat()
	exact, readErr := io.ReadAll(io.LimitReader(handle, maximumStudyArtifactBytes+1))
	openedAfter, afterErr := handle.Stat()
	closeErr := handle.Close()
	terminal, terminalErr := workspace.root.Lstat(relative)
	if statErr != nil || readErr != nil || afterErr != nil || closeErr != nil || terminalErr != nil ||
		!opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 || hasSpecialMode(opened.Mode()) ||
		!singleLink(opened) || !singleLink(openedAfter) || !singleLink(terminal) ||
		!os.SameFile(before, opened) || !os.SameFile(opened, openedAfter) || !os.SameFile(openedAfter, terminal) ||
		!terminal.Mode().IsRegular() || terminal.Mode().Perm() != 0o600 || hasSpecialMode(terminal.Mode()) ||
		opened.Mode() != openedAfter.Mode() || openedAfter.Mode() != terminal.Mode() ||
		opened.ModTime() != openedAfter.ModTime() || openedAfter.ModTime() != terminal.ModTime() ||
		int64(len(exact)) != before.Size() || terminal.Size() != before.Size() {
		return StudyEvidenceEntry{}, nil, fmt.Errorf("evidence file changed during snapshot: %s", relative)
	}
	digest := sha256.Sum256(exact)
	return StudyEvidenceEntry{
		path: relative, mode: "0600", bytes: int64(len(exact)),
		sha256: hex.EncodeToString(digest[:]), exact: append([]byte(nil), exact...),
	}, terminal, nil
}

func cloneStudyEvidenceEntries(entries []StudyEvidenceEntry) []StudyEvidenceEntry {
	result := make([]StudyEvidenceEntry, len(entries))
	for index, entry := range entries {
		result[index] = entry
		result[index].exact = append([]byte(nil), entry.exact...)
	}
	return result
}

func cloneStudyEvidenceSnapshot(snapshot StudyEvidenceSnapshot) StudyEvidenceSnapshot {
	return StudyEvidenceSnapshot{
		directories: append([]string(nil), snapshot.directories...),
		files:       cloneStudyEvidenceEntries(snapshot.files),
		totalBytes:  snapshot.totalBytes,
	}
}

func sameStudyEvidenceSnapshot(left, right StudyEvidenceSnapshot) bool {
	if left.totalBytes != right.totalBytes || !equalStrings(left.directories, right.directories) || len(left.files) != len(right.files) {
		return false
	}
	for index := range left.files {
		leftEntry := left.files[index]
		rightEntry := right.files[index]
		if leftEntry.path != rightEntry.path || leftEntry.mode != rightEntry.mode || leftEntry.bytes != rightEntry.bytes ||
			leftEntry.sha256 != rightEntry.sha256 || !bytes.Equal(leftEntry.exact, rightEntry.exact) {
			return false
		}
	}
	return true
}

func validateEvidenceManifest(manifest EvidenceManifest) ([]string, []EvidenceFile, error) {
	directories := append([]string(nil), manifest.Directories...)
	files := append([]EvidenceFile(nil), manifest.Files...)
	if len(files) == 0 {
		return nil, nil, errors.New("evidence manifest must contain at least one file")
	}
	seen := make(map[string]struct{}, len(directories)+len(files))
	for _, directory := range directories {
		if !validWorkspaceRelativePath(directory) {
			return nil, nil, errors.New("evidence manifest directory path is invalid")
		}
		if _, duplicate := seen[directory]; duplicate {
			return nil, nil, errors.New("evidence manifest paths must be unique")
		}
		seen[directory] = struct{}{}
	}
	for _, file := range files {
		if !validWorkspaceRelativePath(file.Path) || file.Bytes < 1 || file.Bytes > maximumStudyArtifactBytes || !validEvidenceSHA256(file.SHA256) {
			return nil, nil, errors.New("evidence manifest file identity is invalid")
		}
		if _, duplicate := seen[file.Path]; duplicate {
			return nil, nil, errors.New("evidence manifest paths must be unique")
		}
		seen[file.Path] = struct{}{}
	}
	sort.Strings(directories)
	sort.Slice(files, func(left, right int) bool { return files[left].Path < files[right].Path })
	return directories, files, nil
}

func validEvidenceSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func (workspace *EvidenceWorkspace) inspectEvidenceTree() ([]string, []string, error) {
	root, err := workspace.root.Lstat(".")
	if err != nil || !root.IsDir() || root.Mode()&os.ModeSymlink != 0 || root.Mode().Perm() != 0o700 || hasSpecialMode(root.Mode()) {
		return nil, nil, errors.New("evidence workspace root is not exact and private")
	}
	var directories, files []string
	var walk func(string) error
	walk = func(relative string) error {
		handle, err := workspace.root.Open(relative)
		if err != nil {
			return fmt.Errorf("open evidence directory: %w", err)
		}
		opened, statErr := handle.Stat()
		entries, readErr := handle.ReadDir(-1)
		openedAfter, afterErr := handle.Stat()
		closeErr := handle.Close()
		terminal, terminalErr := workspace.root.Lstat(relative)
		if statErr != nil || readErr != nil || afterErr != nil || closeErr != nil || terminalErr != nil ||
			!opened.IsDir() || opened.Mode().Perm() != 0o700 || hasSpecialMode(opened.Mode()) ||
			!os.SameFile(opened, openedAfter) || !os.SameFile(openedAfter, terminal) ||
			!terminal.IsDir() || terminal.Mode().Perm() != 0o700 || hasSpecialMode(terminal.Mode()) ||
			opened.Mode() != openedAfter.Mode() || openedAfter.Mode() != terminal.Mode() ||
			opened.ModTime() != openedAfter.ModTime() || openedAfter.ModTime() != terminal.ModTime() {
			return errors.New("evidence directory identity changed during inspection")
		}
		sort.Slice(entries, func(left, right int) bool { return entries[left].Name() < entries[right].Name() })
		for _, entry := range entries {
			child := entry.Name()
			if relative != "." {
				child = filepath.Join(relative, child)
			}
			if !validWorkspaceRelativePath(child) {
				return errors.New("evidence workspace contains an invalid path")
			}
			metadata, err := workspace.root.Lstat(child)
			if err != nil || metadata.Mode()&os.ModeSymlink != 0 {
				return errors.New("evidence workspace contains a link or unreadable path")
			}
			switch {
			case metadata.IsDir():
				if metadata.Mode().Perm() != 0o700 || hasSpecialMode(metadata.Mode()) {
					return errors.New("evidence directory is not exact and private")
				}
				directories = append(directories, child)
				if len(directories) > maximumStudyEvidenceDirs {
					return errors.New("evidence workspace contains too many directories")
				}
				if err := walk(child); err != nil {
					return err
				}
			case metadata.Mode().IsRegular():
				if metadata.Mode().Perm() != 0o600 || hasSpecialMode(metadata.Mode()) {
					return errors.New("evidence file is not exact and private")
				}
				files = append(files, child)
				if len(files) > maximumStudyEvidenceFiles {
					return errors.New("evidence workspace contains too many files")
				}
			default:
				return errors.New("evidence workspace contains a special file")
			}
		}
		return nil
	}
	if err := walk("."); err != nil {
		return nil, nil, err
	}
	sort.Strings(directories)
	sort.Strings(files)
	return directories, files, nil
}

func (workspace *EvidenceWorkspace) verifyEvidenceFile(expected EvidenceFile) (os.FileInfo, error) {
	before, err := workspace.root.Lstat(expected.Path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Mode().Perm() != 0o600 ||
		hasSpecialMode(before.Mode()) || !singleLink(before) || before.Size() != expected.Bytes {
		return nil, fmt.Errorf("evidence file identity differs from manifest: %s", expected.Path)
	}
	handle, err := workspace.root.Open(expected.Path)
	if err != nil {
		return nil, fmt.Errorf("open evidence file: %w", err)
	}
	opened, statErr := handle.Stat()
	digest := sha256.New()
	observedBytes, readErr := io.Copy(digest, io.LimitReader(handle, maximumStudyArtifactBytes+1))
	openedAfter, afterErr := handle.Stat()
	closeErr := handle.Close()
	terminal, terminalErr := workspace.root.Lstat(expected.Path)
	if statErr != nil || readErr != nil || afterErr != nil || closeErr != nil || terminalErr != nil ||
		!opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 || hasSpecialMode(opened.Mode()) ||
		!singleLink(opened) || !singleLink(openedAfter) || !singleLink(terminal) ||
		!os.SameFile(before, opened) || !os.SameFile(opened, openedAfter) || !os.SameFile(openedAfter, terminal) ||
		!terminal.Mode().IsRegular() || terminal.Mode().Perm() != 0o600 || hasSpecialMode(terminal.Mode()) ||
		opened.Mode() != openedAfter.Mode() || openedAfter.Mode() != terminal.Mode() ||
		opened.ModTime() != openedAfter.ModTime() || openedAfter.ModTime() != terminal.ModTime() ||
		observedBytes != expected.Bytes || terminal.Size() != expected.Bytes {
		return nil, fmt.Errorf("evidence file changed during inspection: %s", expected.Path)
	}
	if hex.EncodeToString(digest.Sum(nil)) != expected.SHA256 {
		return nil, fmt.Errorf("evidence file digest differs from manifest: %s", expected.Path)
	}
	return terminal, nil
}

func sortedEvidenceFilePaths(files []EvidenceFile) []string {
	paths := make([]string, len(files))
	for index, file := range files {
		paths[index] = file.Path
	}
	return paths
}

func equalStrings(left, right []string) bool {
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

func (workspace *EvidenceWorkspace) readyRelative(relative string) error {
	if workspace == nil || workspace.root == nil || workspace.closed {
		return errors.New("evidence workspace is closed")
	}
	if !validWorkspaceRelativePath(relative) {
		return errors.New("evidence path must be one clean relative path")
	}
	return nil
}

func (workspace *EvidenceWorkspace) requirePrivateParents(relative string) error {
	parent := filepath.Dir(relative)
	if parent == "." {
		return nil
	}
	current := ""
	for _, part := range strings.Split(parent, string(filepath.Separator)) {
		if current == "" {
			current = part
		} else {
			current = filepath.Join(current, part)
		}
		metadata, err := workspace.root.Lstat(current)
		if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 || metadata.Mode().Perm() != 0o700 ||
			metadata.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return errors.New("evidence artifact parent is not an existing private directory")
		}
	}
	return nil
}

func (workspace *EvidenceWorkspace) close() error {
	workspace.mu.Lock()
	if workspace.closed {
		workspace.mu.Unlock()
		return errors.New("evidence workspace was already closed")
	}
	var validationErr error
	var commit StudyTerminalCommit
	if workspace.validation != nil {
		terminal, err := workspace.snapshotEvidenceLocked()
		if err != nil || !sameStudyEvidenceSnapshot(workspace.validation.snapshot, terminal) {
			validationErr = &InputError{
				Code:   "STUDY_EVIDENCE_CLOSURE_REFUSED",
				Detail: "the evidence tree changed after its validated terminal snapshot",
			}
		} else {
			commit = workspace.validation.commit
		}
	}
	workspace.closed = true
	closeErr := workspace.root.Close()
	workspace.root = nil
	workspace.validation = nil
	workspace.mu.Unlock()
	if validationErr == nil && closeErr == nil && commit != nil {
		if err := invokeStudyTerminalCommit(commit); err != nil {
			validationErr = &InputError{
				Code:   "STUDY_EVIDENCE_CLOSURE_REFUSED",
				Detail: "the terminal evidence commit could not be staged",
			}
		}
	}
	return errors.Join(validationErr, closeErr)
}

func invokeStudyTerminalCommit(commit StudyTerminalCommit) (returnErr error) {
	defer func() {
		if recover() != nil {
			returnErr = errors.New("terminal evidence commit panicked")
		}
	}()
	if commit == nil {
		return errors.New("terminal evidence commit is absent")
	}
	return commit()
}

func validWorkspaceRelativePath(path string) bool {
	if path == "" || len(path) > 4096 || !utf8.ValidString(path) || filepath.IsAbs(path) || filepath.Clean(path) != path || strings.Contains(path, "\\") {
		return false
	}
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	for _, character := range path {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) || unicode.Is(unicode.Zl, character) || unicode.Is(unicode.Zp, character) {
			return false
		}
	}
	return true
}

func readSourceInput(path string, stdin io.Reader, workingDirectory string) (sourceInput, error) {
	if !validInputPath(path) {
		return sourceInput{}, &InputError{Code: "INPUT_PATH_INVALID", Detail: "--spec must name one bounded path or '-' for stdin"}
	}
	if path == "-" {
		if stdin == nil {
			return sourceInput{}, &InputError{Code: "INPUT_READ_FAILED", Detail: "stdin is unavailable"}
		}
		exact, err := readBounded(stdin)
		if err != nil {
			return sourceInput{}, err
		}
		return sourceInput{displayPath: "stdin", argument: path, exact: exact}, nil
	}

	root := workingDirectory
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return sourceInput{}, &InputError{Code: "WORKSPACE_UNAVAILABLE", Detail: "the current working directory is unavailable"}
		}
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return sourceInput{}, &InputError{Code: "WORKSPACE_INVALID", Detail: "the working directory must be clean and absolute"}
	}
	absolute := path
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, path)
	}
	absolute = filepath.Clean(absolute)

	before, err := os.Lstat(absolute)
	if err != nil {
		return sourceInput{}, &InputError{Code: "INPUT_OPEN_FAILED", Detail: "the source path could not be opened"}
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return sourceInput{}, &InputError{Code: "INPUT_NOT_REGULAR", Detail: "the source path must be a regular file, not a link or special file"}
	}
	if before.Size() < 1 || before.Size() > maximumSourceBytes {
		return sourceInput{}, &InputError{Code: "INPUT_SIZE_INVALID", Detail: "the source must contain 1..1048576 bytes"}
	}

	handle, err := os.Open(absolute)
	if err != nil {
		return sourceInput{}, &InputError{Code: "INPUT_OPEN_FAILED", Detail: "the source path could not be opened"}
	}
	defer handle.Close()
	opened, err := handle.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) ||
		before.Size() != opened.Size() || before.ModTime() != opened.ModTime() {
		return sourceInput{}, &InputError{Code: "INPUT_CHANGED", Detail: "the source identity changed while it was opened"}
	}
	exact, err := readBounded(handle)
	if err != nil {
		return sourceInput{}, err
	}
	openedAfter, statErr := handle.Stat()
	pathAfter, pathErr := os.Lstat(absolute)
	if statErr != nil || pathErr != nil || !pathAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(opened, openedAfter) || !os.SameFile(openedAfter, pathAfter) ||
		opened.Size() != openedAfter.Size() || opened.ModTime() != openedAfter.ModTime() {
		return sourceInput{}, &InputError{Code: "INPUT_CHANGED", Detail: "the source identity changed while it was read"}
	}
	return sourceInput{displayPath: path, argument: path, exact: exact}, nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	exact, err := io.ReadAll(io.LimitReader(reader, maximumSourceBytes+1))
	if err != nil {
		return nil, &InputError{Code: "INPUT_READ_FAILED", Detail: "the source bytes could not be read"}
	}
	if len(exact) < 1 || len(exact) > maximumSourceBytes {
		return nil, &InputError{Code: "INPUT_SIZE_INVALID", Detail: "the source must contain 1..1048576 bytes"}
	}
	return exact, nil
}

func validInputPath(path string) bool {
	if path == "-" {
		return true
	}
	if path == "" || len(path) > 4096 || !utf8.ValidString(path) || strings.IndexByte(path, 0) >= 0 {
		return false
	}
	for _, character := range path {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) || unicode.Is(unicode.Zl, character) || unicode.Is(unicode.Zp, character) {
			return false
		}
	}
	return true
}

func isInputError(err error) bool {
	var target *InputError
	return errors.As(err, &target)
}
