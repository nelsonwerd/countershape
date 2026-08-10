package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	maximumSourceBytes        = 1 << 20
	maximumStudyArtifactBytes = 16 << 20
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
	mu       sync.Mutex
	root     *os.Root
	rootPath string
	closed   bool
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
		opened.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || opened.Size() != int64(len(exact)) {
		return errors.New("written evidence artifact is not exact and private")
	}
	terminal, err := workspace.root.Lstat(relative)
	if err != nil || terminal.Mode()&os.ModeSymlink != 0 || !terminal.Mode().IsRegular() || !os.SameFile(opened, terminal) {
		return errors.New("evidence artifact identity changed after writing")
	}
	return nil
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
	defer workspace.mu.Unlock()
	if workspace.closed {
		return errors.New("evidence workspace was already closed")
	}
	workspace.closed = true
	err := workspace.root.Close()
	workspace.root = nil
	return err
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
